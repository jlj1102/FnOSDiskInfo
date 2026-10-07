package main

import (
	"encoding/json"
	"strings"
	"testing"
)

const scsiFixture = `{
  "device": {"name": "/dev/sda", "type": "scsi", "protocol": "SCSI"},
  "model_name": "SEAGATE ST6000NM0034",
  "scsi_revision": "E005",
  "scsi_version": "SPC-4",
  "scsi_transport_protocol": {"name": "SAS (SPL-4)", "value": 6},
  "serial_number": "ZAD0F3DL0000C715DXFS",
  "user_capacity": {"bytes": 6001175126016},
  "rotation_rate": 7200,
  "temperature": {"current": 39, "drive_trip": 60},
  "power_on_time": {"hours": 52563},
  "scsi_grown_defect_list": 2,
  "scsi_pending_defects": {"count": 0},
  "scsi_error_counter_log": {
    "read": {"total_errors_corrected": 123456, "total_uncorrected_errors": 3},
    "write": {"total_errors_corrected": 0, "total_uncorrected_errors": 0},
    "verify": {"total_errors_corrected": 0, "total_uncorrected_errors": 0}
  },
  "scsi_start_stop_cycle_counter": {"Accumulated start-stop cycles": 42, "Accumulated load-unload cycles": 812},
  "smart_support": {"available": true, "enabled": true}
}`

// SATA members behind MegaRAID scan as "-d sat /dev/bus/N"; the comment's
// member index must be folded back into a replayable "-d megaraid,N".
func TestParseScanMegaraid(t *testing.T) {
	out := strings.Join([]string{
		"/dev/bus/0 -d sat # /dev/bus/0 [megaraid_disk_00] [SAT], ATA device",
		"/dev/bus/0 -d megaraid,3 # /dev/bus/0 [megaraid_disk_03], SCSI device",
		"/dev/sda -d scsi # /dev/sda, SCSI device",
		"# /dev/sdb -d scsi # /dev/sdb, SCSI device open failed: no such device",
	}, "\n")
	devs := parseScanOutput(out)
	want := []Device{
		{Path: "/dev/bus/0", Type: "megaraid,0"},
		{Path: "/dev/bus/0", Type: "megaraid,3"},
		{Path: "/dev/sda", Type: "scsi"},
	}
	if len(devs) != len(want) {
		t.Fatalf("devices = %+v", devs)
	}
	for i := range want {
		if devs[i] != want[i] {
			t.Errorf("device %d = %+v, want %+v", i, devs[i], want[i])
		}
	}
}

func TestNormalizeSCSI(t *testing.T) {
	var j smartJSON
	if err := json.Unmarshal([]byte(scsiFixture), &j); err != nil {
		t.Fatal(err)
	}
	d := normalize(Device{Path: "/dev/sda", Type: "scsi"}, &j)
	d.evaluateHealth(defaultSettings().forDisk(d.ID, false))

	if d.Protocol != "SAS" {
		t.Errorf("protocol = %q", d.Protocol)
	}
	if d.Firmware != "E005" {
		t.Errorf("firmware = %q", d.Firmware)
	}
	if d.Standard != "SPC-4" {
		t.Errorf("standard = %q", d.Standard)
	}
	if d.PowerOnCount == nil || *d.PowerOnCount != 42 {
		t.Errorf("power on count = %v", d.PowerOnCount)
	}
	if d.PowerOnHours == nil || *d.PowerOnHours != 52563 {
		t.Errorf("power on hours = %v", d.PowerOnHours)
	}
	if d.SmartKey != "SmartScsi" {
		t.Errorf("smart key = %q", d.SmartKey)
	}
	// No smart_status -> health stays unknown (decision), but the counter
	// rows are still populated.
	if d.Health != "unknown" {
		t.Errorf("health = %q (%v)", d.Health, d.StatusReasons)
	}
	if len(d.Attributes) != 6 {
		t.Fatalf("attributes = %d", len(d.Attributes))
	}
	byID := map[int]Attribute{}
	for _, a := range d.Attributes {
		byID[a.ID] = a
	}
	if a := byID[0x02]; a.Raw != "2" || a.Status != "caution" {
		t.Errorf("grown defect row = %+v", a)
	}
	if a := byID[0x03]; a.Raw != "0" || a.Status != "good" {
		t.Errorf("pending defect row = %+v", a)
	}
	if a := byID[0x04]; !strings.Contains(a.Raw, "corrected 123456 / uncorrected 3") || a.Status != "caution" {
		t.Errorf("read error row = %+v", a)
	}
	if a := byID[0x05]; a.Status != "good" {
		t.Errorf("write error row = %+v", a)
	}
	if a := byID[0x07]; a.Raw != "812" {
		t.Errorf("load/unload row = %+v", a)
	}

	// With smart_status the health class follows it and row 01 appears.
	var jp smartJSON
	passed := strings.Replace(scsiFixture, `"smart_support"`, `"smart_status": {"passed": true}, "smart_support"`, 1)
	if err := json.Unmarshal([]byte(passed), &jp); err != nil {
		t.Fatal(err)
	}
	dp := normalize(Device{Path: "/dev/sda", Type: "scsi"}, &jp)
	dp.evaluateHealth(defaultSettings().forDisk(dp.ID, false))
	if dp.Health != "good" {
		t.Errorf("health with status = %q (%v)", dp.Health, dp.StatusReasons)
	}
	if len(dp.Attributes) != 7 || dp.Attributes[0].ID != 0x01 || dp.Attributes[0].Raw != "OK" {
		t.Errorf("status row = %+v", dp.Attributes[0])
	}

	// Failed status -> bad, and the row LED follows.
	jf := strings.Replace(scsiFixture, `"smart_support"`, `"smart_status": {"passed": false}, "smart_support"`, 1)
	var jbad smartJSON
	if err := json.Unmarshal([]byte(jf), &jbad); err != nil {
		t.Fatal(err)
	}
	db := normalize(Device{Path: "/dev/sda", Type: "scsi"}, &jbad)
	db.evaluateHealth(defaultSettings().forDisk(db.ID, false))
	if db.Health != "bad" || db.Attributes[0].Status != "bad" {
		t.Errorf("failed status: health=%q row=%+v", db.Health, db.Attributes[0])
	}
}

func TestStableIDTypeSuffix(t *testing.T) {
	var j smartJSON
	if got := stableID("/dev/bus/0", "megaraid,7", &j); got != "dev-bus0-megaraid7" {
		t.Errorf("stable id = %q", got)
	}
}
