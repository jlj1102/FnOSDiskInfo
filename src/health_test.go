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
		d := &Disk{}
		if got := attributeStatus(d, &c.a, s); got != c.want {
			t.Errorf("id %02X: got %s, want %s", c.a.ID, got, c.want)
		}
	}
}

// CDI's NVMe LED rules (UpdateListCtrl NVMe branch).
func TestNVMeAttributeLED(t *testing.T) {
	s := defaultSettings().forDisk("nvme0", true)
	newDisk := func(attrs ...Attribute) *Disk {
		return &Disk{NVMe: &NVMeInfo{}, Attributes: attrs}
	}
	spare := func(cur, th int) []Attribute {
		return []Attribute{{ID: 0x03, Current: cur, Status: "good"}, {ID: 0x04, Current: th, Status: "good"}}
	}
	cases := []struct {
		name string
		d    *Disk
		a    Attribute
		want string
	}{
		{"critical warning", newDisk(), Attribute{ID: 0x01, Current: 1}, "bad"},
		{"critical warning ok", newDisk(), Attribute{ID: 0x01, Current: 0}, "good"},
		{"temp below alarm", newDisk(), Attribute{ID: 0x02, Current: 33}, "good"},
		{"temp at alarm", newDisk(), Attribute{ID: 0x02, Current: s.AlarmTemp}, "bad"},
		{"spare above threshold", newDisk(spare(100, 50)...), Attribute{ID: 0x03, Current: 100}, "good"},
		{"spare at threshold", newDisk(spare(50, 50)...), Attribute{ID: 0x03, Current: 50}, "caution"},
		{"spare below threshold", newDisk(spare(49, 50)...), Attribute{ID: 0x03, Current: 49}, "bad"},
		{"spare threshold unsupported", newDisk(spare(0, 0)...), Attribute{ID: 0x03, Current: 0}, "good"},
		{"used low", newDisk(), Attribute{ID: 0x05, Current: 7}, "good"},
		{"used near wear limit", newDisk(), Attribute{ID: 0x05, Current: 95}, "caution"},
	}
	for _, c := range cases {
		if got := attributeStatus(c.d, &c.a, s); got != c.want {
			t.Errorf("%s: got %s, want %s", c.name, got, c.want)
		}
	}
}

// Kingston SA400 (real sample): E7 current 7 / raw 93 -> life 93 via the
// family rules, plus host reads/writes and NAND writes from the attributes.
func TestSSDFamilyValues(t *testing.T) {
	d := &Disk{
		Model:    "KINGSTON SA400S37120G",
		Firmware: "SBFK62C3",
		IsSSD:    true,
		Attributes: []Attribute{
			{ID: 0x01, Current: 0, RawValue: 0},
			{ID: 0x09, Current: 100, RawValue: 9418},
			{ID: 0x0C, Current: 100, RawValue: 1327},
			{ID: 0xA9, Current: 100, RawValue: 13},
			{ID: 0xAD, Current: 100, RawValue: 103},
			{ID: 0xC2, Current: 70, RawValue: 30},
			{ID: 0xE7, Current: 7, RawValue: 93, Raw: "93"},
			{ID: 0xE9, Current: 100, RawValue: 6836},
			{ID: 0xF1, Current: 100, RawValue: 4643},
			{ID: 0xF2, Current: 100, RawValue: 4164},
		},
	}
	fam, ok := matchSSDFamily(d.Model, d.Firmware, d)
	if !ok || fam.key != "SmartKingstonSA400" || !fam.lifeRaw {
		t.Fatalf("family = %+v ok=%v", fam, ok)
	}
	applySSDValues(d, fam)
	if d.Life == nil || *d.Life != 93 {
		t.Fatalf("life = %v, want 93", d.Life)
	}
	if d.HostWrites == nil || *d.HostWrites != 4643 {
		t.Errorf("host writes = %v", d.HostWrites)
	}
	if d.HostReads == nil || *d.HostReads != 4164 {
		t.Errorf("host reads = %v", d.HostReads)
	}
	if d.NandWrites == nil || *d.NandWrites != 6836 {
		t.Errorf("nand writes = %v", d.NandWrites)
	}
}
