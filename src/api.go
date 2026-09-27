package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type server struct {
	dataDir string
}

func newHandler(dataDir string) http.Handler {
	s := &server{dataDir: dataDir}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/disks", s.handleDisks)
	mux.HandleFunc("GET /api/disks/{id}", s.handleDisk)
	mux.HandleFunc("GET /api/disks/{id}/smart", s.handleSmart)
	mux.HandleFunc("GET /api/disks/{id}/history", s.handleHistory)
	mux.HandleFunc("GET /api/disks/{id}/raw", s.handleRaw)
	mux.HandleFunc("GET /api/disks/{id}/report.txt", s.handleReport)
	mux.HandleFunc("POST /api/disks/rescan", s.handleRescan)
	mux.HandleFunc("POST /api/disks/{id}/self-test", s.handleSelfTest)
	mux.HandleFunc("POST /api/disks/{id}/abort-test", s.handleAbortTest)
	mux.HandleFunc("POST /api/disks/{id}/aam-apm", s.handleAamApm)
	mux.HandleFunc("POST /api/disks/{id}/aam-apm-get", s.handleAamApmGet)
	mux.HandleFunc("GET /api/requests/{id}", s.handleRequest)
	mux.HandleFunc("GET /api/settings", s.handleGetSettings)
	mux.HandleFunc("PUT /api/settings", s.handlePutSettings)
	mux.HandleFunc("GET /api/alarms", s.handleAlarms)
	mux.HandleFunc("GET /api/themes", s.handleThemes)
	mux.HandleFunc("POST /api/themes/import", s.handleThemeImport)
	mux.HandleFunc("DELETE /api/themes/{id}", s.handleThemeDelete)

	sub, err := fs.Sub(webFiles, "web")
	if err != nil {
		log.Fatal(err)
	}
	embedded := http.FileServerFS(sub)
	mux.Handle("/themes/", s.serveThemes(embedded))
	mux.Handle("/", embedded)
	return mux
}

// serveThemes serves imported theme files from $TRIM_PKGVAR/themes and falls
// back to the embedded built-in themes.
func (s *server) serveThemes(embedded http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rest := strings.TrimPrefix(r.URL.Path, "/themes/")
		parts := strings.SplitN(rest, "/", 2)
		if len(parts) == 2 && themeIDPattern.MatchString(parts[0]) && !isBuiltinTheme(parts[0]) {
			name := path.Clean("/" + parts[1])
			name = strings.TrimPrefix(name, "/")
			if name != "" && !strings.Contains(name, "..") {
				p := filepath.Join(themesDir(s.dataDir), parts[0], filepath.FromSlash(name))
				if st, err := os.Stat(p); err == nil && !st.IsDir() {
					http.ServeFile(w, r, p)
					return
				}
			}
		}
		embedded.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// sameOrigin rejects cross-site POST/PUT from other origins.
func sameOrigin(r *http.Request) bool {
	o := r.Header.Get("Origin")
	if o == "" {
		return true
	}
	return strings.HasSuffix(o, "://"+r.Host)
}

func (s *server) findDisk(id string) (Disk, bool) {
	resp, err := readCache(filepath.Join(s.dataDir, "cache.json"))
	if err != nil {
		return Disk{}, false
	}
	for _, d := range resp.Disks {
		if d.ID == id {
			return d, true
		}
	}
	return Disk{}, false
}

func (s *server) handleDisks(w http.ResponseWriter, r *http.Request) {
	resp, err := readCache(filepath.Join(s.dataDir, "cache.json"))
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, disksResponse{
			Version: version,
			Error:   "collector has not produced a cache yet",
		})
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *server) handleDisk(w http.ResponseWriter, r *http.Request) {
	d, ok := s.findDisk(r.PathValue("id"))
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "unknown disk id"})
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func (s *server) handleSmart(w http.ResponseWriter, r *http.Request) {
	d, ok := s.findDisk(r.PathValue("id"))
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "unknown disk id"})
		return
	}
	attrs := d.Attributes
	if attrs == nil {
		attrs = []Attribute{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": d.ID, "device": d.Device, "attributes": attrs})
}

func (s *server) handleHistory(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, ok := s.findDisk(id); !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "unknown disk id"})
		return
	}
	metric := r.URL.Query().Get("metric")
	if metric == "" {
		metric = "temperature"
	}
	points := 500
	if p := r.URL.Query().Get("points"); p != "" {
		if p == "all" {
			points = 0
		} else if n, err := strconv.Atoi(p); err == nil && n > 0 {
			points = n
		}
	}
	series, ok := readHistory(s.dataDir, id, metric, points)
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unknown metric"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "metric": metric, "points": series})
}

func (s *server) handleRaw(w http.ResponseWriter, r *http.Request) {
	d, ok := s.findDisk(r.PathValue("id"))
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "unknown disk id"})
		return
	}
	path := filepath.Join(s.dataDir, "raw", d.ID+".json")
	if _, err := os.Stat(path); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no raw data yet"})
		return
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"cdifnos-%s.json\"", d.ID))
	http.ServeFile(w, r, path)
}

func (s *server) handleReport(w http.ResponseWriter, r *http.Request) {
	resp, err := readCache(filepath.Join(s.dataDir, "cache.json"))
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "no data"})
		return
	}
	id := r.PathValue("id")
	if _, ok := s.findDisk(id); !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "unknown disk id"})
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"cdifnos-report.txt\""))
	_, _ = w.Write([]byte(buildReport(resp)))
}

func (s *server) handleRescan(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(r) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "cross-origin request rejected"})
		return
	}
	if err := os.WriteFile(filepath.Join(s.dataDir, "rescan"), []byte("1"), 0644); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]bool{"ok": true})
}

func (s *server) submit(w http.ResponseWriter, r *http.Request, action string, extra func(req *request) bool) {
	if !sameOrigin(r) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "cross-origin request rejected"})
		return
	}
	diskID := r.PathValue("id")
	if _, ok := s.findDisk(diskID); !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "unknown disk id"})
		return
	}
	req := request{
		ID:      newRequestID(),
		Action:  action,
		Disk:    diskID,
		Created: time.Now().Unix(),
	}
	if extra != nil && !extra(&req) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	if err := submitRequest(s.dataDir, req); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"request": req.ID})
}

func (s *server) handleSelfTest(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Type string `json:"type"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	s.submit(w, r, "self-test", func(req *request) bool {
		if !selftestTypes[body.Type] {
			return false
		}
		req.TestType = body.Type
		return true
	})
}

func (s *server) handleAbortTest(w http.ResponseWriter, r *http.Request) {
	s.submit(w, r, "abort-test", nil)
}

func (s *server) handleAamApmGet(w http.ResponseWriter, r *http.Request) {
	s.submit(w, r, "aam-apm-get", nil)
}

func (s *server) handleAamApm(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Kind  string `json:"kind"`
		Value string `json:"value"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	s.submit(w, r, body.Kind, func(req *request) bool {
		if body.Kind != "aam" && body.Kind != "apm" {
			return false
		}
		if !requestValuePattern.MatchString(body.Value) {
			return false
		}
		req.Value = body.Value
		return true
	})
}

func (s *server) handleRequest(w http.ResponseWriter, r *http.Request) {
	res, state := readRequestState(s.dataDir, r.PathValue("id"))
	switch state {
	case "done":
		writeJSON(w, http.StatusOK, res)
	case "pending":
		writeJSON(w, http.StatusOK, map[string]string{"state": "pending"})
	default:
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "unknown request"})
	}
}

type settingsResponse struct {
	settings
	BuiltInDefaults diskSettings            `json:"built_in_defaults"`
	Effective       map[string]diskSettings `json:"effective,omitempty"`
}

func (s *server) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	st := loadSettings(s.dataDir)
	resp := settingsResponse{settings: st, BuiltInDefaults: defaultSettings().Default}
	resp.Disks = st.Disks
	if resp.Disks == nil {
		resp.Disks = map[string]diskOverride{}
	}
	if cache, err := readCache(filepath.Join(s.dataDir, "cache.json")); err == nil {
		resp.Effective = map[string]diskSettings{}
		for _, d := range cache.Disks {
			resp.Effective[d.ID] = st.forDisk(d.ID, d.NVMe != nil)
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *server) handlePutSettings(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(r) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "cross-origin request rejected"})
		return
	}
	st := defaultSettings()
	if err := json.NewDecoder(r.Body).Decode(&st); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if err := validateSettings(st); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := saveSettings(s.dataDir, st); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, st)
}

func validateSettings(s settings) error {
	if s.IntervalSeconds < 2 || s.IntervalSeconds > 3600 {
		return errors.New("interval_seconds must be 2..3600")
	}
	check := func(name string, v, min, max int) error {
		if v < min || v > max {
			return fmt.Errorf("%s must be %d..%d", name, min, max)
		}
		return nil
	}
	if err := check("default.alarm_temp", s.Default.AlarmTemp, 1, 100); err != nil {
		return err
	}
	if err := check("default.threshold_05", s.Default.Threshold05, 0, 100000); err != nil {
		return err
	}
	if err := check("default.threshold_c5", s.Default.ThresholdC5, 0, 100000); err != nil {
		return err
	}
	if err := check("default.threshold_c6", s.Default.ThresholdC6, 0, 100000); err != nil {
		return err
	}
	if err := check("default.threshold_ff", s.Default.ThresholdFF, 0, 100); err != nil {
		return err
	}
	for id, o := range s.Disks {
		if o.AlarmTemp != nil {
			if err := check(id+".alarm_temp", *o.AlarmTemp, 1, 100); err != nil {
				return err
			}
		}
		if o.Threshold05 != nil {
			if err := check(id+".threshold_05", *o.Threshold05, 0, 100000); err != nil {
				return err
			}
		}
		if o.ThresholdC5 != nil {
			if err := check(id+".threshold_c5", *o.ThresholdC5, 0, 100000); err != nil {
				return err
			}
		}
		if o.ThresholdC6 != nil {
			if err := check(id+".threshold_c6", *o.ThresholdC6, 0, 100000); err != nil {
				return err
			}
		}
		if o.ThresholdFF != nil {
			if err := check(id+".threshold_ff", *o.ThresholdFF, 0, 100); err != nil {
				return err
			}
		}
	}
	for _, id := range s.ExcludeDisks {
		if !validDiskID(id) {
			return fmt.Errorf("invalid exclude_disks id: %s", id)
		}
	}
	return nil
}

func (s *server) handleAlarms(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"alarms": readAlarms(s.dataDir, 200)})
}

func (s *server) handleThemes(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"themes": listThemes(s.dataDir)})
}

func (s *server) handleThemeImport(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(r) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "cross-origin request rejected"})
		return
	}
	nameHint := r.URL.Query().Get("name")
	infos, skipped, err := importThemes(s.dataDir, r.Body, nameHint)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error(), "skipped": skipped})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"themes": infos, "skipped": skipped})
}

func (s *server) handleThemeDelete(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(r) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "cross-origin request rejected"})
		return
	}
	id := r.PathValue("id")
	if !themeIDPattern.MatchString(id) || isBuiltinTheme(id) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "cannot delete this theme"})
		return
	}
	target := filepath.Join(themesDir(s.dataDir), id)
	if _, err := os.Stat(target); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "theme not found"})
		return
	}
	if err := os.RemoveAll(target); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
