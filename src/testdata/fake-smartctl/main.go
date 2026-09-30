// fake smartctl for local end-to-end testing (never packaged).
// Devices: sda HDD (sat), sdb NVMe, sdc SATA SSD (Kingston SA400).
package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

const hddFixture = `{
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
    {"id": 190, "name": "Airflow_Temperature_Cel", "value": 55, "worst": 40, "thresh": 40, "when_failed": "past", "raw": {"value": 45, "string": "45 (Min/Max 43/48)"}},
    {"id": 194, "name": "Temperature_Celsius", "value": 30, "worst": 40, "thresh": 0, "when_failed": "", "raw": {"value": %d, "string": "%d"}}
  ]}
}`

const nvmeFixture = `{
  "json_format_version": [1, 0],
  "smartctl": {"version": [7, 3], "exit_status": 0, "messages": []},
  "device": {"name": "/dev/sdb", "info_name": "/dev/sdb", "type": "nvme", "protocol": "NVMe"},
  "model_name": "FAKE-BC501 NVMe SK hynix 256GB",
  "serial_number": "FAKE-NVME-001",
  "firmware_version": "80002C00",
  "nvme_version": {"string": "1.2", "value": 66048},
  "nvme_number_of_namespaces": 1,
  "user_capacity": {"blocks": 500118192, "bytes": 256060514304},
  "logical_block_size": 512,
  "smart_support": {"available": true, "enabled": true},
  "smart_status": {"passed": true, "nvme": {"value": 0}},
  "nvme_optional_nvm_commands": {"value": 45, "dataset_management": true},
  "nvme_smart_health_information_log": {
    "critical_warning": 0,
    "temperature": %d,
    "available_spare": 100,
    "available_spare_threshold": 50,
    "percentage_used": 7,
    "data_units_read": 104531623,
    "data_units_written": 57968870,
    "host_reads": 1944991175,
    "host_writes": 871572678,
    "controller_busy_time": 6811,
    "power_cycles": 3983,
    "power_on_hours": %d,
    "unsafe_shutdowns": 1336,
    "media_errors": 0,
    "num_err_log_entries": 0,
    "warning_temp_time": 22,
    "critical_comp_time": 1,
    "temperature_sensors": [33, 34]
  },
  "temperature": {"current": %d},
  "power_cycle_count": 3983,
  "power_on_time": {"hours": %d}
}`

const ssdFixture = `{
  "json_format_version": [1, 0],
  "smartctl": {"version": [7, 3], "exit_status": 0, "messages": []},
  "device": {"name": "/dev/sdc", "info_name": "/dev/sdc [SAT]", "type": "sat", "protocol": "ATA"},
  "model_name": "KINGSTON SA400S37120G",
  "serial_number": "FAKE-SSD-001",
  "firmware_version": "SBFKB1E1",
  "user_capacity": {"blocks": 234441648, "bytes": 120034123776},
  "rotation_rate": 0,
  "interface_speed": {
    "max": {"string": "6.0 Gb/s", "units_per_second": 60},
    "current": {"string": "6.0 Gb/s", "units_per_second": 60}
  },
  "ata_version": {"string": "ACS-3 T13/2161-D revision 4", "major_value": 2040, "minor_value": 283},
  "sata_version": {"string": "SATA 3.2", "value": 255},
  "trim": {"supported": true, "deterministic": false, "zeroed": false},
  "smart_support": {"available": true, "enabled": true},
  "ata_apm": {"enabled": true, "level": 254, "string": "maximum performance"},
  "ata_log_directory": {"table": [{"address": 16, "name": "NCQ Command Error log", "read": true}]},
  "ata_smart_data": {"capabilities": {"gp_logging_supported": true}},
  "smart_status": {"passed": true},
  "temperature": {"current": %d},
  "power_on_time": {"hours": %d},
  "power_cycle_count": 1327,
  "ata_smart_attributes": {"revision": 16, "table": [
    {"id": 1, "name": "Raw_Read_Error_Rate", "value": 0, "worst": 100, "thresh": 0, "when_failed": "", "raw": {"value": 0, "string": "0"}},
    {"id": 9, "name": "Power_On_Hours", "value": 100, "worst": 100, "thresh": 0, "when_failed": "", "raw": {"value": %d, "string": "%d"}},
    {"id": 12, "name": "Power_Cycle_Count", "value": 100, "worst": 100, "thresh": 0, "when_failed": "", "raw": {"value": 1327, "string": "1327"}},
    {"id": 148, "name": "Unknown_Attribute", "value": 100, "worst": 100, "thresh": 0, "when_failed": "", "raw": {"value": 0, "string": "0"}},
    {"id": 149, "name": "Unknown_Attribute", "value": 100, "worst": 100, "thresh": 0, "when_failed": "", "raw": {"value": 0, "string": "0"}},
    {"id": 167, "name": "Write_Protect_Mode", "value": 100, "worst": 100, "thresh": 0, "when_failed": "", "raw": {"value": 0, "string": "0"}},
    {"id": 168, "name": "SATA_Phy_Error_Count", "value": 100, "worst": 100, "thresh": 0, "when_failed": "", "raw": {"value": 0, "string": "0"}},
    {"id": 169, "name": "Bad_Block_Rate", "value": 100, "worst": 100, "thresh": 0, "when_failed": "", "raw": {"value": 13, "string": "13"}},
    {"id": 170, "name": "Bad_Blk_Ct_Lat/Erl", "value": 100, "worst": 100, "thresh": 0, "when_failed": "", "raw": {"value": 9, "string": "0/9"}},
    {"id": 172, "name": "Erase_Fail_Count", "value": 100, "worst": 100, "thresh": 0, "when_failed": "", "raw": {"value": 0, "string": "0"}},
    {"id": 173, "name": "MaxAvgErase_Ct", "value": 100, "worst": 100, "thresh": 0, "when_failed": "", "raw": {"value": 4587623, "string": "103 (Average 70)"}},
    {"id": 181, "name": "Program_Fail_Count", "value": 100, "worst": 100, "thresh": 0, "when_failed": "", "raw": {"value": 0, "string": "0"}},
    {"id": 182, "name": "Erase_Fail_Count", "value": 100, "worst": 100, "thresh": 0, "when_failed": "", "raw": {"value": 0, "string": "0"}},
    {"id": 187, "name": "Reported_Uncorrect", "value": 100, "worst": 100, "thresh": 0, "when_failed": "", "raw": {"value": 0, "string": "0"}},
    {"id": 192, "name": "Unsafe_Shutdown_Count", "value": 100, "worst": 100, "thresh": 0, "when_failed": "", "raw": {"value": 34, "string": "34"}},
    {"id": 194, "name": "Temperature_Celsius", "value": 70, "worst": 59, "thresh": 0, "when_failed": "", "raw": {"value": 176094511134, "string": "30 (Min/Max 13/41)"}},
    {"id": 196, "name": "Reallocated_Event_Count", "value": 100, "worst": 100, "thresh": 0, "when_failed": "", "raw": {"value": 0, "string": "0"}},
    {"id": 199, "name": "SATA_CRC_Error_Count", "value": 100, "worst": 100, "thresh": 0, "when_failed": "", "raw": {"value": 0, "string": "0"}},
    {"id": 218, "name": "CRC_Error_Count", "value": 100, "worst": 100, "thresh": 0, "when_failed": "", "raw": {"value": 0, "string": "0"}},
    {"id": 231, "name": "SSD_Life_Left", "value": 7, "worst": 7, "thresh": 0, "when_failed": "", "raw": {"value": 93, "string": "93"}},
    {"id": 233, "name": "Flash_Writes_GiB", "value": 100, "worst": 100, "thresh": 0, "when_failed": "", "raw": {"value": 6836, "string": "6836"}},
    {"id": 241, "name": "Lifetime_Writes_GiB", "value": 100, "worst": 100, "thresh": 0, "when_failed": "", "raw": {"value": 4643, "string": "4643"}},
    {"id": 242, "name": "Lifetime_Reads_GiB", "value": 100, "worst": 100, "thresh": 0, "when_failed": "", "raw": {"value": 4164, "string": "4164"}},
    {"id": 244, "name": "Average_Erase_Count", "value": 100, "worst": 100, "thresh": 0, "when_failed": "", "raw": {"value": 70, "string": "70"}},
    {"id": 245, "name": "Max_Erase_Count", "value": 100, "worst": 100, "thresh": 0, "when_failed": "", "raw": {"value": 103, "string": "103"}},
    {"id": 246, "name": "Total_Erase_Count", "value": 100, "worst": 100, "thresh": 0, "when_failed": "", "raw": {"value": 274184, "string": "274184"}}
  ]}
}`

func main() {
	args := os.Args[1:]
	only := os.Getenv("FAKE_DEV") // "nvme" / "ssd" exposes a single /dev/sda device
	for _, a := range args {
		if a == "--version" || a == "-V" {
			fmt.Println("smartctl 7.3 2022-02-28 r5338 [x86_64-linux] (fake)")
			return
		}
		if a == "--scan-open" {
			switch only {
			case "nvme":
				fmt.Println("/dev/sda -d nvme # /dev/sda, NVMe device")
			case "ssd":
				fmt.Println("/dev/sda -d sat # /dev/sda [SAT], ATA device")
			default:
				fmt.Println("/dev/sda -d sat # /dev/sda [SAT], ATA device")
				fmt.Println("/dev/sdb -d nvme # /dev/sdb, NVMe device")
				fmt.Println("/dev/sdc -d sat # /dev/sdc [SAT], ATA device")
			}
			return
		}
	}
	temp := 30 + int(time.Now().Unix()/10)%5
	hours := 18234 + int(time.Now().Unix()%50)
	nvTemp := 33 + int(time.Now().Unix()/10)%3
	nvHours := 3014 + int(time.Now().Unix()%20)
	joined := strings.Join(args, " ")
	if only == "nvme" {
		joined += " /dev/sda"
	}
	switch {
	case strings.Contains(joined, "-t "):
		fmt.Println("Testing has begun.")
	case strings.Contains(joined, " -X"):
		fmt.Println("Self-test aborted.")
	case strings.Contains(joined, "-g "):
		fmt.Println("ATA feature set executed.")
	case strings.Contains(joined, "-j") && (only == "nvme" || strings.Contains(joined, "/dev/sdb")):
		fmt.Printf(nvmeFixture, nvTemp, nvHours, nvTemp, nvHours)
	case strings.Contains(joined, "-j") && (only == "ssd" || strings.Contains(joined, "/dev/sdc")):
		fmt.Printf(ssdFixture, temp, hours, hours, hours)
	case strings.Contains(joined, "-j"):
		fmt.Printf(hddFixture, temp, hours, hours, hours, temp, temp)
	default:
		os.Exit(1)
	}
}
