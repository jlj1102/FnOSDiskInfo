package main

import (
	"os"
	"testing"
)

func TestSettingsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s := defaultSettings()
	s.IntervalSeconds = 30
	v := 2
	s.Disks = map[string]diskOverride{"wwn-0x1": {Threshold05: &v}}
	if err := saveSettings(dir, s); err != nil {
		t.Fatal(err)
	}
	got := loadSettings(dir)
	if got.IntervalSeconds != 30 {
		t.Errorf("interval = %d", got.IntervalSeconds)
	}
	eff := got.forDisk("wwn-0x1", false)
	if eff.Threshold05 != 2 || eff.ThresholdC5 != 1 {
		t.Errorf("effective = %+v", eff)
	}
	if err := validateSettings(s); err != nil {
		t.Errorf("validate: %v", err)
	}
	bad := defaultSettings()
	bad.IntervalSeconds = 1
	if err := validateSettings(bad); err == nil {
		t.Error("expected interval validation error")
	}
}

func TestReadHistory(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(historyDir(dir), 0755); err != nil {
		t.Fatal(err)
	}
	lines := "{\"t\":100,\"temperature\":30,\"status\":\"good\"}\n{\"t\":200,\"temperature\":35,\"status\":\"good\"}\n"
	if err := os.WriteFile(historyPath(dir, "x"), []byte(lines), 0644); err != nil {
		t.Fatal(err)
	}
	pts, ok := readHistory(dir, "x", "temperature", 1)
	if !ok || len(pts) != 1 || pts[0][0] != 200 || pts[0][1] != 35 {
		t.Fatalf("points = %v ok=%v", pts, ok)
	}
	if _, ok := readHistory(dir, "x", "bogus", 0); ok {
		t.Error("expected unknown metric to be rejected")
	}
}
