package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type diskSettings struct {
	AlarmTemp   int `json:"alarm_temp"`
	Threshold05 int `json:"threshold_05"`
	ThresholdC5 int `json:"threshold_c5"`
	ThresholdC6 int `json:"threshold_c6"`
	ThresholdFF int `json:"threshold_ff"`
}

type diskOverride struct {
	AlarmTemp   *int `json:"alarm_temp,omitempty"`
	Threshold05 *int `json:"threshold_05,omitempty"`
	ThresholdC5 *int `json:"threshold_c5,omitempty"`
	ThresholdC6 *int `json:"threshold_c6,omitempty"`
	ThresholdFF *int `json:"threshold_ff,omitempty"`
}

type settings struct {
	IntervalSeconds int                     `json:"interval_seconds"`
	Default         diskSettings            `json:"default"`
	Disks           map[string]diskOverride `json:"disks,omitempty"`
	ExcludeDisks    []string                `json:"exclude_disks,omitempty"`
}

func defaultSettings() settings {
	return settings{
		IntervalSeconds: 10,
		Default: diskSettings{
			AlarmTemp:   50,
			Threshold05: 1,
			ThresholdC5: 1,
			ThresholdC6: 1,
			ThresholdFF: 10,
		},
	}
}

func settingsPath(dataDir string) string { return filepath.Join(dataDir, "settings.json") }

// loadSettings returns defaults for missing/invalid files or fields.
func loadSettings(dataDir string) settings {
	s := defaultSettings()
	raw, err := os.ReadFile(settingsPath(dataDir))
	if err == nil {
		_ = json.Unmarshal(raw, &s)
	}
	if s.IntervalSeconds < 2 || s.IntervalSeconds > 3600 {
		s.IntervalSeconds = 10
	}
	if s.Default.AlarmTemp <= 0 || s.Default.AlarmTemp > 100 {
		s.Default.AlarmTemp = 50
	}
	s.Default.Threshold05 = clampThreshold(s.Default.Threshold05, 1)
	s.Default.ThresholdC5 = clampThreshold(s.Default.ThresholdC5, 1)
	s.Default.ThresholdC6 = clampThreshold(s.Default.ThresholdC6, 1)
	s.Default.ThresholdFF = clampThreshold(s.Default.ThresholdFF, 10)
	out := s.ExcludeDisks[:0]
	for _, id := range s.ExcludeDisks {
		if validDiskID(id) {
			out = append(out, id)
		}
	}
	s.ExcludeDisks = append([]string{}, out...)
	return s
}

func validDiskID(id string) bool {
	if id == "" || len(id) > 128 {
		return false
	}
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == '.', r == ':':
		default:
			return false
		}
	}
	return true
}

func clampThreshold(v, def int) int {
	if v < 0 || v > 100000 {
		return def
	}
	return v
}

func saveSettings(dataDir string, s settings) error {
	raw, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := settingsPath(dataDir) + ".tmp"
	if err := os.WriteFile(tmp, raw, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, settingsPath(dataDir))
}

// forDisk merges the global default with the per-disk override.
// NVMe disks default to a 60C alarm, other disks to 50C (matches CDI).
func (s settings) forDisk(id string, nvme bool) diskSettings {
	d := s.Default
	o := s.Disks[id]
	if o.AlarmTemp != nil {
		d.AlarmTemp = *o.AlarmTemp
	}
	if o.Threshold05 != nil {
		d.Threshold05 = *o.Threshold05
	}
	if o.ThresholdC5 != nil {
		d.ThresholdC5 = *o.ThresholdC5
	}
	if o.ThresholdC6 != nil {
		d.ThresholdC6 = *o.ThresholdC6
	}
	if o.ThresholdFF != nil {
		d.ThresholdFF = *o.ThresholdFF
	}
	if nvme && o.AlarmTemp == nil && s.Default.AlarmTemp == 50 {
		d.AlarmTemp = 60
	}
	return d
}
