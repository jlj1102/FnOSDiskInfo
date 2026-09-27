package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const version = "0.1.0"

//go:embed web
var webFiles embed.FS

type disksResponse struct {
	Version   string    `json:"version"`
	UpdatedAt time.Time `json:"updated_at"`
	Error     string    `json:"error,omitempty"`
	Disks     []Disk    `json:"disks"`
}

// collector runs privileged (root): it invokes smartctl and writes the
// normalized snapshot to $data/cache.json. The web service never runs smartctl.
type collector struct {
	mu       sync.Mutex
	smartctl string
	dataDir  string
	interval time.Duration
}

func newCollector(dataDir string, interval time.Duration) *collector {
	return &collector{dataDir: dataDir, interval: interval}
}

func (c *collector) cachePath() string { return filepath.Join(c.dataDir, "cache.json") }
func (c *collector) kickPath() string  { return filepath.Join(c.dataDir, "rescan") }

func (c *collector) run() {
	c.collect()
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	last := time.Now()
	for range tick.C {
		kicked := os.Remove(c.kickPath()) == nil
		if kicked || time.Since(last) >= c.interval {
			c.collect()
			last = time.Now()
		}
	}
}

func (c *collector) collect() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.smartctl == "" {
		c.smartctl = findSmartctl()
	}
	resp := disksResponse{Version: version, UpdatedAt: time.Now(), Disks: []Disk{}}
	switch {
	case c.smartctl == "":
		resp.Error = "smartctl not found"
	default:
		devices, err := scanDevices(c.smartctl)
		if err != nil {
			resp.Error = err.Error()
			break
		}
		for _, dev := range devices {
			j, err := readSmart(c.smartctl, dev)
			if err != nil {
				resp.Disks = append(resp.Disks, Disk{
					ID:     "dev-" + sanitize(strings.TrimPrefix(dev.Path, "/dev/")),
					Device: dev.Path,
					Health: "unknown",
					Error:  err.Error(),
				})
				continue
			}
			resp.Disks = append(resp.Disks, normalize(dev, j))
		}
	}

	raw, err := json.Marshal(resp)
	if err != nil {
		log.Printf("cache marshal: %v", err)
		return
	}
	tmp := c.cachePath() + ".tmp"
	if err := os.WriteFile(tmp, raw, 0644); err != nil {
		log.Printf("cache write: %v", err)
		return
	}
	if err := os.Rename(tmp, c.cachePath()); err != nil {
		log.Printf("cache rename: %v", err)
	}
}

func readCache(path string) (disksResponse, error) {
	var resp disksResponse
	raw, err := os.ReadFile(path)
	if err != nil {
		return resp, err
	}
	return resp, json.Unmarshal(raw, &resp)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func handler(dataDir string) http.Handler {
	cachePath := filepath.Join(dataDir, "cache.json")
	kickPath := filepath.Join(dataDir, "rescan")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/disks", func(w http.ResponseWriter, r *http.Request) {
		resp, err := readCache(cachePath)
		if err != nil {
			writeJSON(w, http.StatusServiceUnavailable, disksResponse{
				Version: version,
				Error:   "collector has not produced a cache yet",
			})
			return
		}
		writeJSON(w, http.StatusOK, resp)
	})
	mux.HandleFunc("GET /api/disks/{id}", func(w http.ResponseWriter, r *http.Request) {
		resp, err := readCache(cachePath)
		if err == nil {
			for _, d := range resp.Disks {
				if d.ID == r.PathValue("id") {
					writeJSON(w, http.StatusOK, d)
					return
				}
			}
		}
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "unknown disk id"})
	})
	mux.HandleFunc("GET /api/disks/{id}/smart", func(w http.ResponseWriter, r *http.Request) {
		resp, err := readCache(cachePath)
		if err == nil {
			for _, d := range resp.Disks {
				if d.ID == r.PathValue("id") {
					attrs := d.Attributes
					if attrs == nil {
						attrs = []Attribute{}
					}
					writeJSON(w, http.StatusOK, map[string]any{"id": d.ID, "device": d.Device, "attributes": attrs})
					return
				}
			}
		}
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "unknown disk id"})
	})
	mux.HandleFunc("POST /api/disks/rescan", func(w http.ResponseWriter, r *http.Request) {
		_ = os.WriteFile(kickPath, []byte("1"), 0644)
		writeJSON(w, http.StatusAccepted, map[string]bool{"ok": true})
	})

	sub, err := fs.Sub(webFiles, "web")
	if err != nil {
		log.Fatal(err)
	}
	mux.Handle("/", http.FileServerFS(sub))
	return mux
}

func main() {
	args := os.Args[1:]
	mode := "all"
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		mode = args[0]
		args = args[1:]
	}

	flags := flag.NewFlagSet("cdi-server", flag.ExitOnError)
	port := flags.Int("port", 7817, "HTTP listen port")
	data := flags.String("data", ".", "writable data directory")
	interval := flags.Duration("interval", 10*time.Second, "collection interval")
	_ = flags.Parse(args)

	if err := os.MkdirAll(*data, 0755); err != nil {
		log.Fatal(err)
	}
	if f, err := os.OpenFile(filepath.Join(*data, "cdi.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err == nil {
		log.SetOutput(io.MultiWriter(os.Stderr, f))
		defer f.Close()
	}

	addr := fmt.Sprintf(":%d", *port)
	switch mode {
	case "collect":
		newCollector(*data, *interval).run()
	case "serve":
		log.Printf("cdifnos %s serving on %s", version, addr)
		if err := http.ListenAndServe(addr, handler(*data)); err != nil {
			log.Fatal(err)
		}
	case "all":
		go newCollector(*data, *interval).run()
		log.Printf("cdifnos %s listening on %s (single process)", version, addr)
		if err := http.ListenAndServe(addr, handler(*data)); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatalf("unknown mode %q (want collect|serve)", mode)
	}
}
