package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Privileged operations are requested by the unprivileged web process through
// files in $TRIM_PKGVAR/requests; the root collector executes them.

type request struct {
	ID       string `json:"id"`
	Action   string `json:"action"` // self-test | abort-test | aam | apm
	Disk     string `json:"disk"`
	TestType string `json:"test_type,omitempty"`
	Value    string `json:"value,omitempty"` // aam/apm: "off" or 1..254
	Created  int64  `json:"created"`
}

type requestResult struct {
	ID     string `json:"id"`
	State  string `json:"state"` // done
	Output string `json:"output,omitempty"`
	Error  string `json:"error,omitempty"`
	At     int64  `json:"at"`
}

var requestValuePattern = regexp.MustCompile(`^(off|[1-9][0-9]{0,2})$`)

var selftestTypes = map[string]bool{"short": true, "long": true, "conveyance": true}

func requestsDir(dataDir string) string { return filepath.Join(dataDir, "requests") }

func submitRequest(dataDir string, r request) error {
	dir := requestsDir(dataDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	raw, err := json.Marshal(r)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, r.ID+".json"), raw, 0644)
}

// readRequestState returns "done" | "pending" | "missing".
func readRequestState(dataDir, id string) (requestResult, string) {
	dir := requestsDir(dataDir)
	if id == "" || strings.ContainsAny(id, `/\..`) {
		return requestResult{}, "missing"
	}
	if raw, err := os.ReadFile(filepath.Join(dir, id+".result")); err == nil {
		var res requestResult
		if json.Unmarshal(raw, &res) == nil {
			return res, "done"
		}
	}
	if _, err := os.Stat(filepath.Join(dir, id+".json")); err == nil {
		return requestResult{}, "pending"
	}
	return requestResult{}, "missing"
}

// processRequests is called by the collector loop every second.
func (c *collector) processRequests() {
	dir := requestsDir(c.dataDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		path := filepath.Join(dir, name)
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var r request
		if json.Unmarshal(raw, &r) != nil {
			_ = os.Remove(path)
			continue
		}
		res := c.execRequest(r)
		out, _ := json.Marshal(res)
		_ = os.WriteFile(filepath.Join(dir, r.ID+".result"), out, 0644)
		_ = os.Remove(path)
		c.kick()
	}
}

func (c *collector) execRequest(r request) requestResult {
	res := requestResult{ID: r.ID, State: "done", At: time.Now().Unix()}

	c.mu.Lock()
	var disk *Disk
	for i := range c.disks {
		if c.disks[i].ID == r.Disk {
			d := c.disks[i]
			disk = &d
			break
		}
	}
	c.mu.Unlock()
	if disk == nil {
		res.Error = "unknown disk id"
		return res
	}

	var args []string
	switch r.Action {
	case "self-test":
		if !selftestTypes[r.TestType] {
			res.Error = "invalid test type"
			return res
		}
		args = append(args, "-t", r.TestType)
	case "abort-test":
		args = append(args, "-X")
	case "aam-apm-get":
		args = append(args, "-j", "-g", "aam", "-g", "apm")
	case "aam", "apm":
		if !requestValuePattern.MatchString(r.Value) {
			res.Error = "invalid value"
			return res
		}
		args = append(args, "-g", r.Action+","+r.Value)
	default:
		res.Error = "invalid action"
		return res
	}
	if disk.smartType != "" {
		args = append(args, "-d", disk.smartType)
	}
	args = append(args, disk.Device)

	out, err := runSmartctlCombined(c.smartctl, 60*time.Second, args...)
	res.Output = string(out)
	if err != nil && len(out) == 0 {
		res.Error = err.Error()
	}
	return res
}

func newRequestID() string {
	return strconv.FormatInt(time.Now().UnixNano(), 10)
}
