package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func loadATA(t *testing.T) smartJSON {
	t.Helper()
	raw, err := os.ReadFile("testdata/ata.json")
	if err != nil {
		t.Fatal(err)
	}
	var j smartJSON
	if err := json.Unmarshal(raw, &j); err != nil {
		t.Fatal(err)
	}
	return j
}

func TestNormalizeATA(t *testing.T) {
	j := loadATA(t)
	d := normalize(Device{Path: "/dev/sda", Type: "sat"}, &j)
	d.evaluateHealth(defaultSettings().forDisk(d.ID, false))

	if d.ID != "wwn-0x5000c500601a0097" {
		t.Errorf("id = %q", d.ID)
	}
	if d.Model != "ST3000DM008-2DM166" || d.Serial != "Z9A1B2C3" || d.Firmware != "CC26" {
		t.Errorf("model/serial/fw = %q/%q/%q", d.Model, d.Serial, d.Firmware)
	}
	if d.Health != "good" {
		t.Errorf("health = %q (%v)", d.Health, d.StatusReasons)
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
	if d.RotationRate != 7200 {
		t.Errorf("rotation rate = %d", d.RotationRate)
	}
	if d.TransferMode != "SATA/600 | SATA/600" {
		t.Errorf("transfer mode = %q", d.TransferMode)
	}
	if d.Standard != "ACS-3 | SATA 3.1" {
		t.Errorf("standard = %q", d.Standard)
	}
	if got := strings.Join(d.Features, ","); got != "S.M.A.R.T.,NCQ,GPL" {
		t.Errorf("features = %q", got)
	}
}

func TestATACaution(t *testing.T) {
	j := loadATA(t)
	j.AtaAttrTable.Table[0].Raw.Value = 2 // attribute 05 above the default caution threshold 1
	j.AtaAttrTable.Table[0].Raw.String = "2"
	d := normalize(Device{Path: "/dev/sda", Type: "sat"}, &j)
	d.evaluateHealth(defaultSettings().forDisk(d.ID, false))

	if d.Health != "caution" {
		t.Fatalf("health = %q (%v)", d.Health, d.StatusReasons)
	}
	if len(d.StatusReasons) == 0 {
		t.Fatal("expected status reasons")
	}
}

func TestATAThresholdError(t *testing.T) {
	j := loadATA(t)
	j.AtaAttrTable.Table[0].Thresh = 50
	j.AtaAttrTable.Table[0].Value = 40
	d := normalize(Device{Path: "/dev/sda", Type: "sat"}, &j)
	d.evaluateHealth(defaultSettings().forDisk(d.ID, false))

	if d.Health != "bad" {
		t.Fatalf("health = %q (%v)", d.Health, d.StatusReasons)
	}
}

const nvmeFixture = `{
  "device": {"name": "/dev/nvme0", "type": "nvme", "protocol": "NVMe", "model_name": "NVMe SSD", "serial_number": "NV1"},
  "nvme_version": {"string": "1.2", "value": 66048},
  "smart_status": {"passed": true},
  "nvme_smart_health_information_log": {
    "critical_warning": 0,
    "temperature": 40,
    "available_spare": 100,
    "available_spare_threshold": 10,
    "percentage_used": 5,
    "data_units_read": 104531623,
    "data_units_written": 57968870,
    "host_reads": 1944991175,
    "host_writes": 871572678,
    "power_on_hours": 100,
    "power_cycles": 7
  },
  "nvme_self_test_log": {
    "table": [
      {"self_test_code": {"string": "Short"}, "self_test_result": {"string": "Completed without error"}, "power_on_hours": 99}
    ]
  }
}`

func TestNormalizeNVMe(t *testing.T) {
	var j smartJSON
	if err := json.Unmarshal([]byte(nvmeFixture), &j); err != nil {
		t.Fatal(err)
	}
	d := normalize(Device{Path: "/dev/nvme0", Type: "nvme"}, &j)
	d.evaluateHealth(defaultSettings().forDisk(d.ID, true))

	if d.Health != "good" {
		t.Fatalf("health = %q (%v)", d.Health, d.StatusReasons)
	}
	if d.Life == nil || *d.Life != 95 {
		t.Fatalf("life = %v", d.Life)
	}
	if d.AlarmTemp != 60 {
		t.Errorf("nvme alarm temp = %d", d.AlarmTemp)
	}
	if d.SelfTest == nil || d.SelfTest.Type != "Short" {
		t.Errorf("self test = %+v", d.SelfTest)
	}
	if d.NVMe == nil || d.NVMe.AvailableSpare != 100 {
		t.Errorf("nvme info = %+v", d.NVMe)
	}
	if d.Protocol != "NVM Express" || d.SmartKey != "SmartNVMe" {
		t.Errorf("protocol/smart key = %q/%q", d.Protocol, d.SmartKey)
	}
	if d.Standard != "NVM Express 1.2" {
		t.Errorf("standard = %q", d.Standard)
	}
	if len(d.Attributes) != 15 {
		t.Fatalf("attributes = %d", len(d.Attributes))
	}
	if d.Attributes[5].ID != 0x06 || !strings.Contains(d.Attributes[5].Raw, "[") {
		t.Errorf("data units read row = %+v", d.Attributes[5])
	}
	if d.HostReads == nil || *d.HostReads != 49844 {
		t.Errorf("host reads = %v", d.HostReads)
	}
	if d.HostWrites == nil || *d.HostWrites != 27641 {
		t.Errorf("host writes = %v", *d.HostWrites)
	}
	// LED rules: critical warning 0 -> good, percentage used 5 -> good
	if d.Attributes[0].Status != "good" || d.Attributes[4].Status != "good" {
		t.Errorf("led status = %q/%q", d.Attributes[0].Status, d.Attributes[4].Status)
	}
}

func TestNVMeCriticalWarning(t *testing.T) {
	var j smartJSON
	if err := json.Unmarshal([]byte(nvmeFixture), &j); err != nil {
		t.Fatal(err)
	}
	j.NVMeLog.CriticalWarning = 0x04
	d := normalize(Device{Path: "/dev/nvme0", Type: "nvme"}, &j)
	d.evaluateHealth(defaultSettings().forDisk(d.ID, true))

	if d.Health != "bad" {
		t.Fatalf("health = %q (%v)", d.Health, d.StatusReasons)
	}
	if len(d.StatusReasons) != 1 {
		t.Fatalf("reasons = %v", d.StatusReasons)
	}
}

// nvmeLinkMode reads the PCIe link info from sysfs (CDI's
// GetTransferModePCIe equivalent) and formats it as "PCIe 3.0 x4".
func TestNVMELinkMode(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "class", "nvme", "nvme0", "device")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	write := func(name, val string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(val), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("current_link_speed", "8.0 GT/s PCIe\n")
	write("current_link_width", "4\n")
	write("max_link_speed", "8.0 GT/s PCIe\n")
	write("max_link_width", "4\n")
	if got := nvmeLinkMode(root, "/dev/nvme0"); got != "PCIe 3.0 x4 | PCIe 3.0 x4" {
		t.Errorf("nvme0 = %q", got)
	}
	if got := nvmeLinkMode(root, "/dev/nvme0n1"); got != "PCIe 3.0 x4 | PCIe 3.0 x4" {
		t.Errorf("nvme0n1 = %q", got)
	}
	write("current_link_width", "0")
	if got := nvmeLinkMode(root, "/dev/nvme0"); got != "---- | PCIe 3.0 x4" {
		t.Errorf("zero width = %q", got)
	}
	if got := nvmeLinkMode(root, "/dev/nvme9"); got != "" {
		t.Errorf("missing controller = %q", got)
	}
}

func TestParseScanOutput(t *testing.T) {
	out := "/dev/sda -d sat # /dev/sda [SAT], ATA device\n/dev/sdb -d sat # /dev/sdb [SAT], ATA device\n"
	devs := parseScanOutput(out)
	if len(devs) != 2 || devs[0].Path != "/dev/sda" || devs[0].Type != "sat" || devs[1].Path != "/dev/sdb" {
		t.Errorf("devices = %+v", devs)
	}
}
