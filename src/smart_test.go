package main

import (
	"encoding/json"
	"os"
	"testing"
)

func TestNormalizeATA(t *testing.T) {
	raw, err := os.ReadFile("testdata/ata.json")
	if err != nil {
		t.Fatal(err)
	}
	var j smartJSON
	if err := json.Unmarshal(raw, &j); err != nil {
		t.Fatal(err)
	}
	d := normalize(Device{Path: "/dev/sda", Type: "sat"}, &j)

	if d.ID != "wwn-0x5000c500601a0097" {
		t.Errorf("id = %q", d.ID)
	}
	if d.Model != "ST3000DM008-2DM166" || d.Serial != "Z9A1B2C3" || d.Firmware != "CC26" {
		t.Errorf("model/serial/fw = %q/%q/%q", d.Model, d.Serial, d.Firmware)
	}
	if d.Health != "good" {
		t.Errorf("health = %q", d.Health)
	}
	if d.Temperature == nil || *d.Temperature != 31 {
		t.Errorf("temperature = %v", d.Temperature)
	}
	if d.PowerOnHours == nil || *d.PowerOnHours != 18234 {
		t.Errorf("power on hours = %v", d.PowerOnHours)
	}
	if len(d.Attributes) != 2 || d.Attributes[0].Raw != "0" || d.Attributes[1].Name != "Power_On_Hours" {
		t.Errorf("attributes = %+v", d.Attributes)
	}
}

func TestParseScanOutput(t *testing.T) {
	out := "/dev/sda -d sat # /dev/sda [SAT], ATA device\n/dev/sdb -d sat # /dev/sdb [SAT], ATA device\n"
	devs := parseScanOutput(out)
	if len(devs) != 2 || devs[0].Path != "/dev/sda" || devs[0].Type != "sat" || devs[1].Path != "/dev/sdb" {
		t.Errorf("devices = %+v", devs)
	}
}
