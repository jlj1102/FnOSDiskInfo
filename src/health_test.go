package main

import "testing"

// CDI's SMART LED (DiskInfoDlgUpdate.cpp UpdateListCtrl): temperature is
// always good, when_failed "past" is ignored, 05/C5/C6 caution follows the
// configured raw limits, and IDs outside CDI's ranges never go bad.
func TestAttributeLED(t *testing.T) {
	s := defaultSettings().forDisk("test", false)
	cases := []struct {
		a    Attribute
		want string
	}{
		{Attribute{ID: 0xBE, Current: 55, Worst: 40, Threshold: 40, RawValue: 45}, "good"}, // Airflow Temp, when_failed=past in real data
		{Attribute{ID: 0x05, Current: 100, Threshold: 10, RawValue: 2}, "caution"},
		{Attribute{ID: 0x05, Current: 5, Threshold: 10, RawValue: 0}, "bad"},
		{Attribute{ID: 0x05, Current: 100, Threshold: 10, RawValue: 0}, "good"},
		{Attribute{ID: 0xC2, Current: 20, Threshold: 60}, "good"},
		{Attribute{ID: 0xB8, Current: 5, Threshold: 10}, "good"},
		{Attribute{ID: 0xE9, Current: 5, Threshold: 10}, "good"},
		{Attribute{ID: 0xBB, Current: 5, Threshold: 10}, "bad"},
	}
	for _, c := range cases {
		if got := attributeStatus(&c.a, false, s); got != c.want {
			t.Errorf("id %02X: got %s, want %s", c.a.ID, got, c.want)
		}
	}
}
