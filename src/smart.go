package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Device is one physical disk as reported by `smartctl --scan-open`.
type Device struct {
	Path string
	Type string // value for -d, may be empty
}

// Attribute is one normalized SMART attribute row.
type Attribute struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Current   int    `json:"current"`
	Worst     int    `json:"worst"`
	Threshold int    `json:"threshold"`
	Raw       string `json:"raw"`
	RawValue  int64  `json:"raw_value"`
	Status    string `json:"status"` // ok | bad
}

type SelfTest struct {
	Type      string `json:"type,omitempty"`
	Status    string `json:"status,omitempty"`
	Passed    bool   `json:"passed,omitempty"`
	Hours     int    `json:"hours,omitempty"`
	Remaining int    `json:"remaining,omitempty"`
}

type NVMeInfo struct {
	CriticalWarning  int   `json:"critical_warning"`
	Temperature      int   `json:"temperature"`
	AvailableSpare   int   `json:"available_spare"`
	SpareThreshold   int   `json:"spare_threshold"`
	PercentageUsed   int   `json:"percentage_used"`
	DataUnitsRead    int64 `json:"data_units_read"`
	DataUnitsWritten int64 `json:"data_units_written"`
	HostReads        int64 `json:"host_reads"`
	HostWrites       int64 `json:"host_writes"`
	ControllerBusy   int64 `json:"controller_busy_time"`
	PowerCycles      int64 `json:"power_cycles"`
	UnsafeShutdowns  int64 `json:"unsafe_shutdowns"`
	MediaErrors      int64 `json:"media_errors"`
	ErrorLogEntries  int64 `json:"error_log_entries"`
	WarningTempTime  int   `json:"warning_temp_time"`
	CriticalTempTime int   `json:"critical_comp_time"`
}

// Disk is the normalized snapshot served to the frontend.
type Disk struct {
	ID            string      `json:"id"`
	Device        string      `json:"device"`
	Model         string      `json:"model"`
	Serial        string      `json:"serial"`
	Firmware      string      `json:"firmware"`
	Protocol      string      `json:"protocol"`
	IsSSD         bool        `json:"is_ssd,omitempty"`
	CapacityBytes int64       `json:"capacity_bytes"`
	Temperature   *int        `json:"temperature,omitempty"`
	AlarmTemp     int         `json:"alarm_temp,omitempty"`
	PowerOnHours  *int        `json:"power_on_hours,omitempty"`
	PowerOnCount  *int        `json:"power_on_count,omitempty"`
	RotationRate  int         `json:"rotation_rate,omitempty"`
	Life          *int        `json:"life,omitempty"`
	Health        string      `json:"health"` // good | caution | bad | unknown
	StatusReasons []string    `json:"status_reasons,omitempty"`
	SelfTest      *SelfTest   `json:"self_test,omitempty"`
	NVMe          *NVMeInfo   `json:"nvme,omitempty"`
	AAM           *int        `json:"aam,omitempty"`
	APM           *int        `json:"apm,omitempty"`
	Error         string      `json:"error,omitempty"`
	Attributes    []Attribute `json:"attributes,omitempty"`

	smartType   string
	smartPassed *bool
}

// findSmartctl returns the first usable smartctl: the capable copy shipped
// next to the app binary first, then well-known system paths.
func findSmartctl() string {
	candidates := []string{}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(filepath.Dir(exe)), "bin", "smartctl"))
	}
	candidates = append(candidates,
		"/usr/sbin/smartctl", "/usr/bin/smartctl",
		"/usr/local/sbin/smartctl", "/usr/local/bin/smartctl")
	if p, err := exec.LookPath("smartctl"); err == nil {
		candidates = append(candidates, p)
	}
	for _, p := range candidates {
		st, err := os.Stat(p)
		if err != nil || st.IsDir() {
			continue
		}
		// Windows has no execute bits; on Linux require them.
		if runtime.GOOS != "windows" && st.Mode()&0111 == 0 {
			continue
		}
		if out, err := exec.Command(p, "--version").Output(); err == nil && len(out) > 0 {
			return p
		}
	}
	return ""
}

// runSmartctl executes smartctl with the given timeout and returns output.
func runSmartctl(smartctl string, timeout time.Duration, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return exec.CommandContext(ctx, smartctl, args...).Output()
}

// parseScanOutput parses `smartctl --scan-open` lines such as
// `/dev/sda -d sat # /dev/sda [SAT], ATA device`.
func parseScanOutput(out string) []Device {
	var devs []Device
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.SplitN(line, "#", 2)[0]
		fields := strings.Fields(line)
		if len(fields) == 0 || !strings.HasPrefix(fields[0], "/dev/") {
			continue
		}
		d := Device{Path: fields[0]}
		for i := 1; i+1 < len(fields); i++ {
			if fields[i] == "-d" {
				d.Type = fields[i+1]
				break
			}
		}
		devs = append(devs, d)
	}
	return devs
}

func scanDevices(smartctl string) ([]Device, error) {
	out, err := runSmartctl(smartctl, 20*time.Second, "--scan-open")
	if len(out) == 0 && err != nil {
		return nil, err
	}
	devs := parseScanOutput(string(out))
	if len(devs) == 0 {
		return nil, errors.New("no devices found by smartctl --scan-open")
	}
	return devs, nil
}

type selfTestRow struct {
	Type struct {
		String string `json:"string"`
	} `json:"type"`
	Status struct {
		String string `json:"string"`
		Passed bool   `json:"passed"`
	} `json:"status"`
	Code struct {
		String string `json:"string"`
	} `json:"self_test_code"`
	Result struct {
		String string `json:"self_test_result"`
	} `json:"self_test_result"`
	LifetimeHours *int `json:"lifetime_hours"`
	PowerOnHours  *int `json:"power_on_hours"`
	Remaining     *int `json:"remaining_percent"`
}

type smartJSON struct {
	Device struct {
		Name         string `json:"name"`
		Type         string `json:"type"`
		Protocol     string `json:"protocol"`
		ModelName    string `json:"model_name"`
		SerialNumber string `json:"serial_number"`
		Firmware     string `json:"firmware_version"`
		RotationRate int    `json:"rotation_rate"`
	} `json:"device"`
	ModelName    string `json:"model_name"`
	SerialNumber string `json:"serial_number"`
	Firmware     string `json:"firmware_version"`
	UserCapacity struct {
		Bytes int64 `json:"bytes"`
	} `json:"user_capacity"`
	SmartStatus *struct {
		Passed bool `json:"passed"`
	} `json:"smart_status"`
	Temperature struct {
		Current *int `json:"current"`
	} `json:"temperature"`
	PowerOnTime struct {
		Hours *int `json:"hours"`
	} `json:"power_on_time"`
	PowerCycleCount *int `json:"power_cycle_count"`
	WWN             *struct {
		NA  int   `json:"na"`
		OUI int64 `json:"oui"`
		ID  int64 `json:"id"`
	} `json:"wwn"`
	AtaAttrTable struct {
		Table []struct {
			ID         int    `json:"id"`
			Name       string `json:"name"`
			Value      int    `json:"value"`
			Worst      int    `json:"worst"`
			Thresh     int    `json:"thresh"`
			WhenFailed string `json:"when_failed"`
			Raw        struct {
				Value  int64  `json:"value"`
				String string `json:"string"`
			} `json:"raw"`
		} `json:"table"`
	} `json:"ata_smart_attributes"`
	NVMeLog *struct {
		CriticalWarning  int   `json:"critical_warning"`
		Temperature      *int  `json:"temperature"`
		AvailableSpare   int   `json:"available_spare"`
		SpareThreshold   int   `json:"available_spare_threshold"`
		PercentageUsed   int   `json:"percentage_used"`
		DataUnitsRead    int64 `json:"data_units_read"`
		DataUnitsWritten int64 `json:"data_units_written"`
		HostReads        int64 `json:"host_reads"`
		HostWrites       int64 `json:"host_writes"`
		ControllerBusy   int64 `json:"controller_busy_time"`
		PowerCycles      int64 `json:"power_cycles"`
		PowerOnHours     *int  `json:"power_on_hours"`
		UnsafeShutdowns  int64 `json:"unsafe_shutdowns"`
		MediaErrors      int64 `json:"media_errors"`
		ErrorLogEntries  int64 `json:"num_err_log_entries"`
		WarningTempTime  int   `json:"warning_temp_time"`
		CriticalTempTime int   `json:"critical_comp_time"`
	} `json:"nvme_smart_health_information_log"`
	AtaSelfTest *struct {
		Standard struct {
			Table []selfTestRow `json:"table"`
		} `json:"standard"`
	} `json:"ata_smart_self_test_log"`
	NvmeSelfTest *struct {
		Table []selfTestRow `json:"table"`
	} `json:"nvme_self_test_log"`
	AtaAam *struct {
		Enabled bool `json:"enabled"`
		Level   int  `json:"level"`
	} `json:"ata_aam"`
	AtaApm *struct {
		Enabled bool `json:"enabled"`
		Level   int  `json:"level"`
	} `json:"ata_apm"`
	Smartctl struct {
		Messages []struct {
			String   string `json:"string"`
			Severity string `json:"severity"`
		} `json:"messages"`
	} `json:"smartctl"`
}

// readSmart runs `smartctl -j -x` and parses the JSON. smartctl's exit status
// is intentionally ignored: it sets bits for warning/dying disks while still
// producing valid JSON.
func readSmart(smartctl string, d Device) ([]byte, *smartJSON, error) {
	args := []string{"-j", "-x"}
	if d.Type != "" {
		args = append(args, "-d", d.Type)
	}
	args = append(args, d.Path)

	out, err := runSmartctl(smartctl, 30*time.Second, args...)
	if len(out) == 0 {
		if err != nil {
			return nil, nil, err
		}
		return nil, nil, errors.New("smartctl returned no output")
	}
	var j smartJSON
	if e := json.Unmarshal(out, &j); e != nil {
		return out, nil, e
	}
	if j.Device.Name == "" {
		msg := "smartctl could not read device"
		for _, m := range j.Smartctl.Messages {
			if m.Severity == "error" {
				msg = m.String
				break
			}
		}
		return out, nil, errors.New(msg)
	}
	return out, &j, nil
}

func normalize(d Device, j *smartJSON) Disk {
	disk := Disk{
		ID:            stableID(d.Path, j),
		Device:        d.Path,
		Model:         firstNonEmpty(j.Device.ModelName, j.ModelName),
		Serial:        firstNonEmpty(j.Device.SerialNumber, j.SerialNumber),
		Firmware:      firstNonEmpty(j.Device.Firmware, j.Firmware),
		Protocol:      firstNonEmpty(j.Device.Protocol, j.Device.Type),
		CapacityBytes: j.UserCapacity.Bytes,
		Temperature:   j.Temperature.Current,
		PowerOnHours:  j.PowerOnTime.Hours,
		PowerOnCount:  j.PowerCycleCount,
		RotationRate:  j.Device.RotationRate,
		Health:        "unknown",
		Attributes:    []Attribute{},
		smartType:     d.Type,
	}
	if j.SmartStatus != nil {
		passed := j.SmartStatus.Passed
		disk.smartPassed = &passed
	}
	// ponytail: rotation rate 0 is the SSD heuristic; CDI uses Identify data.
	disk.IsSSD = disk.RotationRate == 0 && !strings.Contains(strings.ToUpper(disk.Protocol), "SCSI")

	for _, a := range j.AtaAttrTable.Table {
		raw := a.Raw.String
		if raw == "" && a.Raw.Value != 0 {
			raw = strconv.FormatInt(a.Raw.Value, 10)
		}
		status := "ok"
		if a.WhenFailed != "" {
			status = "bad"
		}
		disk.Attributes = append(disk.Attributes, Attribute{
			ID: a.ID, Name: a.Name, Current: a.Value, Worst: a.Worst,
			Threshold: a.Thresh, Raw: raw, RawValue: a.Raw.Value, Status: status,
		})
	}

	if n := j.NVMeLog; n != nil {
		info := &NVMeInfo{
			CriticalWarning:  n.CriticalWarning,
			AvailableSpare:   n.AvailableSpare,
			SpareThreshold:   n.SpareThreshold,
			PercentageUsed:   n.PercentageUsed,
			DataUnitsRead:    n.DataUnitsRead,
			DataUnitsWritten: n.DataUnitsWritten,
			HostReads:        n.HostReads,
			HostWrites:       n.HostWrites,
			ControllerBusy:   n.ControllerBusy,
			PowerCycles:      n.PowerCycles,
			UnsafeShutdowns:  n.UnsafeShutdowns,
			MediaErrors:      n.MediaErrors,
			ErrorLogEntries:  n.ErrorLogEntries,
			WarningTempTime:  n.WarningTempTime,
			CriticalTempTime: n.CriticalTempTime,
		}
		if n.Temperature != nil {
			info.Temperature = *n.Temperature
			if disk.Temperature == nil {
				disk.Temperature = n.Temperature
			}
		}
		if disk.PowerOnHours == nil && n.PowerOnHours != nil {
			disk.PowerOnHours = n.PowerOnHours
		}
		if disk.PowerOnCount == nil && n.PowerCycles > 0 {
			v := int(n.PowerCycles)
			disk.PowerOnCount = &v
		}
		disk.NVMe = info
	}

	disk.SelfTest = selfTestFrom(j)

	if j.AtaAam != nil && j.AtaAam.Enabled {
		v := j.AtaAam.Level
		disk.AAM = &v
	}
	if j.AtaApm != nil && j.AtaApm.Enabled {
		v := j.AtaApm.Level
		disk.APM = &v
	}

	return disk
}

func selfTestFrom(j *smartJSON) *SelfTest {
	var row *selfTestRow
	if j.AtaSelfTest != nil && len(j.AtaSelfTest.Standard.Table) > 0 {
		row = &j.AtaSelfTest.Standard.Table[0]
	} else if j.NvmeSelfTest != nil && len(j.NvmeSelfTest.Table) > 0 {
		row = &j.NvmeSelfTest.Table[0]
	}
	if row == nil {
		return nil
	}
	st := &SelfTest{
		Type:   firstNonEmpty(row.Type.String, row.Code.String),
		Status: firstNonEmpty(row.Status.String, row.Result.String),
		Passed: row.Status.Passed,
	}
	if row.LifetimeHours != nil {
		st.Hours = *row.LifetimeHours
	} else if row.PowerOnHours != nil {
		st.Hours = *row.PowerOnHours
	}
	if row.Remaining != nil {
		st.Remaining = *row.Remaining
	}
	return st
}

// stableID prefers WWN (udev-compatible NAA-5 form), then serial, then device name.
func stableID(path string, j *smartJSON) string {
	if j.WWN != nil && j.WWN.NA == 0 {
		w := uint64(5)<<60 | (uint64(j.WWN.OUI)&0xFFFFFF)<<36 | uint64(j.WWN.ID)&0xFFFFFFFFF
		return fmt.Sprintf("wwn-0x%x", w)
	}
	if s := sanitize(j.SerialNumber); s != "" {
		return "serial-" + s
	}
	if s := sanitize(j.Device.SerialNumber); s != "" {
		return "serial-" + s
	}
	return "dev-" + sanitize(strings.TrimPrefix(path, "/dev/"))
}

func sanitize(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9', r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z', r == '-', r == '_', r == '.':
			b.WriteRune(r)
		}
	}
	return b.String()
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
