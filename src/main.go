package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const version = "0.7.3"

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
	settings settings
	setMtime time.Time
	disks    []Disk
	kickCh   chan struct{}

	lastHistory      map[string]historyPoint
	lastHistoryWrite map[string]time.Time
	prevHealth       map[string]string
	prevTemp         map[string]string
}

func newCollector(dataDir string) *collector {
	c := &collector{
		dataDir:          dataDir,
		settings:         defaultSettings(),
		kickCh:           make(chan struct{}, 1),
		lastHistory:      map[string]historyPoint{},
		lastHistoryWrite: map[string]time.Time{},
		prevHealth:       map[string]string{},
		prevTemp:         map[string]string{},
	}
	c.maybeReloadSettings()
	c.seedHistory()
	return c
}

func (c *collector) cachePath() string { return filepath.Join(c.dataDir, "cache.json") }
func (c *collector) kickPath() string  { return filepath.Join(c.dataDir, "rescan") }

func (c *collector) kick() {
	select {
	case c.kickCh <- struct{}{}:
	default:
	}
}

func (c *collector) interval() time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	return time.Duration(c.settings.IntervalSeconds) * time.Second
}

func (c *collector) run() {
	c.collect()
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	last := time.Now()
	for range tick.C {
		c.maybeReloadSettings()
		c.processRequests()
		kicked := os.Remove(c.kickPath()) == nil
		if kicked || time.Since(last) >= c.interval() {
			c.collect()
			last = time.Now()
		}
	}
}

func (c *collector) maybeReloadSettings() {
	st, err := os.Stat(settingsPath(c.dataDir))
	if err != nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if st.ModTime().Equal(c.setMtime) {
		return
	}
	c.settings = loadSettings(c.dataDir)
	c.setMtime = st.ModTime()
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
		excluded := map[string]bool{}
		for _, id := range c.settings.ExcludeDisks {
			excluded[id] = true
		}
		prevByDevice := map[string]Disk{}
		for _, d := range c.disks {
			prevByDevice[d.Device+"|"+d.smartType] = d
		}
		for _, dev := range devices {
			if prev, ok := prevByDevice[dev.Path+"|"+dev.Type]; ok && excluded[prev.ID] {
				prev.Stale = true
				resp.Disks = append(resp.Disks, prev)
				continue
			}
			raw, j, err := readSmart(c.smartctl, dev)
			if err != nil {
				id := "dev-" + sanitize(strings.TrimPrefix(dev.Path, "/dev/"))
				if dev.Type != "" {
					id += "-" + sanitize(dev.Type)
				}
				resp.Disks = append(resp.Disks, Disk{
					ID:     id,
					Device: dev.Path,
					Health: "unknown",
					Error:  err.Error(),
				})
				continue
			}
			disk := normalize(dev, j)
			disk.evaluateHealth(c.settings.forDisk(disk.ID, disk.NVMe != nil))
			c.writeRaw(disk.ID, raw)
			now := time.Now()
			c.appendHistory(&disk, now)
			c.updateAlarms(&disk, now)
			resp.Disks = append(resp.Disks, disk)
		}
	}
	c.disks = resp.Disks

	out, err := json.Marshal(resp)
	if err != nil {
		log.Printf("cache marshal: %v", err)
		return
	}
	tmp := c.cachePath() + ".tmp"
	if err := os.WriteFile(tmp, out, 0644); err != nil {
		log.Printf("cache write: %v", err)
		return
	}
	if err := os.Rename(tmp, c.cachePath()); err != nil {
		log.Printf("cache rename: %v", err)
	}
}

func (c *collector) writeRaw(id string, raw []byte) {
	if len(raw) == 0 || id == "" {
		return
	}
	dir := filepath.Join(c.dataDir, "raw")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(dir, id+".json"), raw, 0644)
}

func readCache(path string) (disksResponse, error) {
	var resp disksResponse
	raw, err := os.ReadFile(path)
	if err != nil {
		return resp, err
	}
	return resp, json.Unmarshal(raw, &resp)
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
	_ = flags.Duration("interval", 10*time.Second, "accepted for compatibility; use settings.json")
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
		newCollector(*data).run()
	case "serve":
		log.Printf("cdifnos %s serving on %s", version, addr)
		if err := http.ListenAndServe(addr, newHandler(*data)); err != nil {
			log.Fatal(err)
		}
	case "all":
		go newCollector(*data).run()
		log.Printf("cdifnos %s listening on %s (single process)", version, addr)
		if err := http.ListenAndServe(addr, newHandler(*data)); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatalf("unknown mode %q (want collect|serve)", mode)
	}
}
