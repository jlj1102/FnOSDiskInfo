// fake smartctl for local end-to-end testing (never packaged).
package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

const fixture = `{
  "json_format_version": [1, 0],
  "smartctl": {"version": [7, 3], "exit_status": 0, "messages": []},
  "device": {"name": "/dev/sda", "type": "sat", "protocol": "ATA",
    "model_name": "FAKE-ST3000DM008", "serial_number": "FAKE1234",
    "firmware_version": "CC26"},
  "model_name": "FAKE-ST3000DM008",
  "user_capacity": {"bytes": 3000592982016},
  "rotation_rate": 7200,
  "interface_speed": {
    "max": {"string": "6.0 Gb/s", "units_per_second": 60},
    "current": {"string": "6.0 Gb/s", "units_per_second": 60}
  },
  "ata_version": {"string": "ACS-3 T13/2161-D revision 5", "major_value": 2032, "minor_value": 109},
  "sata_version": {"string": "SATA 3.1", "value": 127},
  "trim": {"supported": false},
  "smart_support": {"available": true, "enabled": true},
  "ata_log_directory": {"table": [{"address": 16, "name": "NCQ Command Error log", "read": true}]},
  "ata_smart_data": {"capabilities": {"gp_logging_supported": true}},
  "smart_status": {"passed": true},
  "temperature": {"current": %d},
  "power_on_time": {"hours": %d},
  "power_cycle_count": 42,
  "wwn": {"na": 0, "oui": 3152, "id": 1612316823},
  "ata_smart_self_test_log": {"standard": {"table": [
    {"type": {"string": "Short offline"}, "status": {"string": "Completed without error", "passed": true}, "lifetime_hours": 100}
  ]}},
  "ata_smart_attributes": {"revision": 16, "table": [
    {"id": 5, "name": "Reallocated_Sector_Ct", "value": 100, "worst": 100, "thresh": 10, "when_failed": "", "raw": {"value": 0, "string": "0"}},
    {"id": 9, "name": "Power_On_Hours", "value": 80, "worst": 80, "thresh": 0, "when_failed": "", "raw": {"value": %d, "string": "%d"}},
    {"id": 194, "name": "Temperature_Celsius", "value": 30, "worst": 40, "thresh": 0, "when_failed": "", "raw": {"value": %d, "string": "%d"}}
  ]}
}`

func main() {
	args := os.Args[1:]
	for _, a := range args {
		if a == "--version" || a == "-V" {
			fmt.Println("smartctl 7.3 2022-02-28 r5338 [x86_64-linux] (fake)")
			return
		}
		if a == "--scan-open" {
			fmt.Println("/dev/sda -d sat # /dev/sda [SAT], ATA device")
			return
		}
	}
	joined := strings.Join(args, " ")
	switch {
	case strings.Contains(joined, "-t "):
		fmt.Println("Testing has begun.")
	case strings.Contains(joined, " -X"):
		fmt.Println("Self-test aborted.")
	case strings.Contains(joined, "-g "):
		fmt.Println("ATA feature set executed.")
	case strings.Contains(joined, "-j"):
		temp := 30 + int(time.Now().Unix()/10)%5
		hours := 18234 + int(time.Now().Unix()%50)
		fmt.Printf(fixture, temp, hours, hours, hours, temp, temp)
	default:
		os.Exit(1)
	}
}
