package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type historyPoint struct {
	T          int64  `json:"t"`
	Status     string `json:"status,omitempty"`
	Temp       *int   `json:"temperature,omitempty"`
	Life       *int   `json:"life,omitempty"`
	Hours      *int   `json:"power_on_hours,omitempty"`
	Count      *int   `json:"power_on_count,omitempty"`
	R05        *int64 `json:"reallocated,omitempty"`
	RC4        *int64 `json:"realloc_events,omitempty"`
	RC5        *int64 `json:"pending,omitempty"`
	RC6        *int64 `json:"uncorrectable,omitempty"`
	HostReads  *int64 `json:"host_reads,omitempty"`
	HostWrites *int64 `json:"host_writes,omitempty"`
}

// historyMetrics maps API metric names to point accessors.
var historyMetrics = map[string]func(historyPoint) (int64, bool){
	"temperature":    func(p historyPoint) (int64, bool) { return ptr(p.Temp) },
	"life":           func(p historyPoint) (int64, bool) { return ptr(p.Life) },
	"power_on_hours": func(p historyPoint) (int64, bool) { return ptr(p.Hours) },
	"power_on_count": func(p historyPoint) (int64, bool) { return ptr(p.Count) },
	"reallocated":    func(p historyPoint) (int64, bool) { return ptr64(p.R05) },
	"realloc_events": func(p historyPoint) (int64, bool) { return ptr64(p.RC4) },
	"pending":        func(p historyPoint) (int64, bool) { return ptr64(p.RC5) },
	"uncorrectable":  func(p historyPoint) (int64, bool) { return ptr64(p.RC6) },
	"host_reads":     func(p historyPoint) (int64, bool) { return ptr64(p.HostReads) },
	"host_writes":    func(p historyPoint) (int64, bool) { return ptr64(p.HostWrites) },
}

func ptr(v *int) (int64, bool) {
	if v == nil {
		return 0, false
	}
	return int64(*v), true
}

func ptr64(v *int64) (int64, bool) {
	if v == nil {
		return 0, false
	}
	return *v, true
}

func historyPath(dataDir, id string) string {
	return filepath.Join(dataDir, "history", id+".jsonl")
}

func historyDir(dataDir string) string { return filepath.Join(dataDir, "history") }

func alarmsPath(dataDir string) string { return filepath.Join(dataDir, "alarms.jsonl") }

func historyPointFrom(d *Disk) historyPoint {
	p := historyPoint{
		T:         time.Now().Unix(),
		Status:    d.Health,
		Temp:      d.Temperature,
		Life:      d.Life,
		Hours:     d.PowerOnHours,
		Count:     d.PowerOnCount,
		HostReads: nil,
	}
	for i := range d.Attributes {
		a := &d.Attributes[i]
		v := a.RawValue & 0xFFFF
		switch a.ID {
		case 0x05:
			vv := v
			p.R05 = &vv
		case 0xC4:
			vv := v
			p.RC4 = &vv
		case 0xC5:
			vv := v
			p.RC5 = &vv
		case 0xC6:
			vv := v
			p.RC6 = &vv
		}
	}
	if d.NVMe != nil {
		// data units are 1000 x 512-byte sectors; convert to GB for graphing
		hr := d.NVMe.DataUnitsRead * 512000 / 1e9
		hw := d.NVMe.DataUnitsWritten * 512000 / 1e9
		p.HostReads, p.HostWrites = &hr, &hw
	}
	return p
}

// appendHistory writes a point only when something changed, throttled to one
// write per 60s unless the health status changed.
func (c *collector) appendHistory(d *Disk, now time.Time) {
	if d.ID == "" {
		return
	}
	p := historyPointFrom(d)
	prev, ok := c.lastHistory[d.ID]
	same := ok && sameHistoryPoint(prev, p)
	if same {
		return
	}
	statusChanged := !ok || prev.Status != p.Status
	if ok && !statusChanged && now.Sub(c.lastHistoryWrite[d.ID]) < 60*time.Second {
		return
	}
	dir := historyDir(c.dataDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return
	}
	f, err := os.OpenFile(historyPath(c.dataDir, d.ID), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	raw, err := json.Marshal(p)
	if err == nil {
		_, _ = f.Write(append(raw, '\n'))
	}
	c.lastHistory[d.ID] = p
	c.lastHistoryWrite[d.ID] = now
}

func sameHistoryPoint(a, b historyPoint) bool {
	// status and t differ by design; compare the sampled values only
	return eqInt(a.Temp, b.Temp) &&
		eqInt(a.Life, b.Life) &&
		eqInt(a.Hours, b.Hours) &&
		eqInt(a.Count, b.Count) &&
		eqInt64(a.R05, b.R05) &&
		eqInt64(a.RC4, b.RC4) &&
		eqInt64(a.RC5, b.RC5) &&
		eqInt64(a.RC6, b.RC6) &&
		eqInt64(a.HostReads, b.HostReads) &&
		eqInt64(a.HostWrites, b.HostWrites)
}

func eqInt(a, b *int) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func eqInt64(a, b *int64) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

type alarmEvent struct {
	T       int64  `json:"t"`
	Disk    string `json:"disk"`
	Model   string `json:"model"`
	Kind    string `json:"kind"` // health | temperature
	From    string `json:"from"`
	To      string `json:"to"`
	Message string `json:"message"`
}

// updateAlarms logs class transitions for health and temperature.
func (c *collector) updateAlarms(d *Disk, now time.Time) {
	if d.ID == "" {
		return
	}
	prevHealth, seen := c.prevHealth[d.ID]
	if d.Health != prevHealth {
		c.appendAlarm(alarmEvent{
			T: now.Unix(), Disk: d.ID, Model: d.Model, Kind: "health",
			From: prevHealth, To: d.Health,
			Message: fmt.Sprintf("%s: %s -> %s", d.Model, labelOr(prevHealth), d.Health),
		})
		c.prevHealth[d.ID] = d.Health
	}
	if !seen && d.Health == "good" {
		c.prevHealth[d.ID] = d.Health
	}

	tempClass := "unknown"
	if d.Temperature != nil {
		if *d.Temperature >= d.AlarmTemp {
			tempClass = "bad"
		} else {
			tempClass = "good"
		}
	}
	if prev, ok := c.prevTemp[d.ID]; ok && tempClass != prev {
		c.appendAlarm(alarmEvent{
			T: now.Unix(), Disk: d.ID, Model: d.Model, Kind: "temperature",
			From: prev, To: tempClass,
			Message: fmt.Sprintf("%s: temperature %d°C (%s -> %s)", d.Model, derefInt(d.Temperature), prev, tempClass),
		})
	}
	c.prevTemp[d.ID] = tempClass
}

func labelOr(s string) string {
	if s == "" {
		return "unknown"
	}
	return s
}

func derefInt(v *int) int {
	if v == nil {
		return 0
	}
	return *v
}

func (c *collector) appendAlarm(ev alarmEvent) {
	f, err := os.OpenFile(alarmsPath(c.dataDir), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	raw, err := json.Marshal(ev)
	if err == nil {
		_, _ = f.Write(append(raw, '\n'))
	}
}

func readAlarms(dataDir string, limit int) []alarmEvent {
	raw, err := os.ReadFile(alarmsPath(dataDir))
	if err != nil {
		return []alarmEvent{}
	}
	lines := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	if limit > 0 && len(lines) > limit {
		lines = lines[len(lines)-limit:]
	}
	out := make([]alarmEvent, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			continue
		}
		var ev alarmEvent
		if json.Unmarshal([]byte(line), &ev) == nil {
			out = append(out, ev)
		}
	}
	return out
}

func readHistory(dataDir, id, metric string, points int) ([][2]int64, bool) {
	fn, ok := historyMetrics[metric]
	if !ok {
		return nil, false
	}
	raw, err := os.ReadFile(historyPath(dataDir, id))
	if err != nil {
		return [][2]int64{}, true
	}
	lines := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	if points > 0 && len(lines) > points {
		lines = lines[len(lines)-points:]
	}
	out := make([][2]int64, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			continue
		}
		var p historyPoint
		if json.Unmarshal([]byte(line), &p) != nil {
			continue
		}
		if v, ok := fn(p); ok {
			out = append(out, [2]int64{p.T, v})
		}
	}
	return out, true
}

// seedHistory loads the last written point per disk so restarts don't produce
// duplicate points. Pruning keeps files bounded.
func (c *collector) seedHistory() {
	entries, err := os.ReadDir(historyDir(c.dataDir))
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		path := filepath.Join(historyDir(c.dataDir), e.Name())
		pruneJSONL(path, 10000)
		if line := lastLine(path); line != "" {
			var p historyPoint
			if json.Unmarshal([]byte(line), &p) == nil {
				id := strings.TrimSuffix(e.Name(), ".jsonl")
				c.lastHistory[id] = p
			}
		}
	}
	pruneJSONL(alarmsPath(c.dataDir), 2000)
}

func lastLine(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	s := strings.TrimRight(string(raw), "\n")
	if i := strings.LastIndexByte(s, '\n'); i >= 0 {
		return s[i+1:]
	}
	return s
}

// pruneJSONL rewrites the file keeping only the last maxLines lines.
func pruneJSONL(path string, maxLines int) {
	raw, err := os.ReadFile(path)
	if err != nil || len(raw) < 4*1024*1024 {
		return
	}
	lines := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	if len(lines) <= maxLines {
		return
	}
	keep := strings.Join(lines[len(lines)-maxLines:], "\n") + "\n"
	_ = os.WriteFile(path, []byte(keep), 0644)
}
