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
	TransferMode  string      `json:"transfer_mode,omitempty"`
	Standard      string      `json:"standard,omitempty"`
	Features      []string    `json:"features,omitempty"`
	SmartKey      string      `json:"smart_key,omitempty"`
	HostReads     *int        `json:"host_reads_gb,omitempty"`
	HostWrites    *int        `json:"host_writes_gb,omitempty"`
	NandWrites    *int        `json:"nand_writes_gb,omitempty"`
	Life          *int        `json:"life,omitempty"`
	Health        string      `json:"health"` // good | caution | bad | unknown
	StatusReasons []string    `json:"status_reasons,omitempty"`
	SelfTest      *SelfTest   `json:"self_test,omitempty"`
	NVMe          *NVMeInfo   `json:"nvme,omitempty"`
	AAM           *int        `json:"aam,omitempty"`
	APM           *int        `json:"apm,omitempty"`
	Error         string      `json:"error,omitempty"`
	Attributes    []Attribute `json:"attributes,omitempty"`
	Stale         bool        `json:"stale,omitempty"`

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

// runSmartctlCombined also captures stderr, for control commands whose
// diagnostics (e.g. "AAM is not supported") only go there.
func runSmartctlCombined(smartctl string, timeout time.Duration, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return exec.CommandContext(ctx, smartctl, args...).CombinedOutput()
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

// sataSpeed is one entry of smartctl's interface_speed (SATA link rate).
type sataSpeed struct {
	String         string `json:"string"`
	UnitsPerSecond int    `json:"units_per_second"`
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
	// RotationRate is top-level in real smartctl output (rpm, 0 = SSD);
	// device.rotation_rate is a fallback for older fixtures.
	RotationRate *int `json:"rotation_rate"`
	SmartSupport struct {
		Available bool `json:"available"`
	} `json:"smart_support"`
	InterfaceSpeed struct {
		Max     sataSpeed `json:"max"`
		Current sataSpeed `json:"current"`
	} `json:"interface_speed"`
	AtaVersion struct {
		MajorValue int `json:"major_value"`
	} `json:"ata_version"`
	SataVersion struct {
		String string `json:"string"`
	} `json:"sata_version"`
	Trim struct {
		Supported bool `json:"supported"`
	} `json:"trim"`
	AtaLogDir struct {
		Table []struct {
			Address int `json:"address"`
		} `json:"table"`
	} `json:"ata_log_directory"`
	AtaSmartData struct {
		Capabilities struct {
			GpLoggingSupported bool `json:"gp_logging_supported"`
		} `json:"capabilities"`
	} `json:"ata_smart_data"`
	NvmeVersion struct {
		String string `json:"string"`
	} `json:"nvme_version"`
	NvmeOptionalNvmCommands struct {
		DatasetManagement bool `json:"dataset_management"`
	} `json:"nvme_optional_nvm_commands"`
	WWN *struct {
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

// ---------- ATA identity extras (transfer mode / standard / features) ----------

// rotationRate reads smartctl's top-level rotation_rate (0 = SSD); some older
// outputs only carry it inside device.
func rotationRate(j *smartJSON) int {
	if j.RotationRate != nil {
		return *j.RotationRate
	}
	return j.Device.RotationRate
}

// sataSpeedString renders one interface_speed entry CDI-style ("SATA/600")
// from the link rate unit; the raw smartctl string is the fallback.
func sataSpeedString(s sataSpeed) string {
	switch s.UnitsPerSecond {
	case 60:
		return "SATA/600"
	case 30:
		return "SATA/300"
	case 15:
		return "SATA/150"
	}
	return s.String
}

func transferMode(j *smartJSON) string {
	cur := sataSpeedString(j.InterfaceSpeed.Current)
	max := sataSpeedString(j.InterfaceSpeed.Max)
	switch {
	case cur == "" && max == "":
		return ""
	case cur == "":
		return max
	case max == "":
		return cur
	}
	return cur + " | " + max
}

// ataMajorName maps ata_version.major_value (bitmask) to the ATA standard
// name: highest set bit, same table as smartctl's get_ata_major_version.
func ataMajorName(v int) string {
	names := []string{"", "ATA-1", "ATA-2", "ATA-3", "ATA/ATAPI-4", "ATA/ATAPI-5",
		"ATA/ATAPI-6", "ATA/ATAPI-7", "ATA8-ACS", "ACS-2", "ACS-3", "ACS-4", "ACS-5", "ACS-6"}
	for bit := 15; bit >= 1; bit-- {
		if v&(1<<bit) == 0 {
			continue
		}
		if bit < len(names) {
			return names[bit]
		}
		return "ACS >6"
	}
	return ""
}

// ataStandard is "ACS-3 | SATA 3.1" (ATA major standard | SATA spec version);
// NVMe uses its own version string.
func ataStandard(j *smartJSON) string {
	if j.NvmeVersion.String != "" {
		return "NVM Express " + j.NvmeVersion.String
	}
	var parts []string
	if n := ataMajorName(j.AtaVersion.MajorValue); n != "" {
		parts = append(parts, n)
	}
	if j.SataVersion.String != "" {
		parts = append(parts, j.SataVersion.String)
	}
	return strings.Join(parts, " | ")
}

// ataFeatures mirrors CDI's feature row with what smartctl's JSON exposes.
// NCQ is inferred from the NCQ Command Error log (GP log 0x10, mandatory for
// NCQ capable devices); DevSleep/Streaming are not in smartctl output.
func ataFeatures(j *smartJSON) []string {
	if j.NVMeLog != nil {
		var f []string
		if j.SmartSupport.Available {
			f = append(f, "S.M.A.R.T.")
		}
		if j.NvmeOptionalNvmCommands.DatasetManagement {
			f = append(f, "TRIM")
		}
		return f
	}
	var f []string
	if j.SmartSupport.Available {
		f = append(f, "S.M.A.R.T.")
	}
	if j.AtaApm != nil {
		f = append(f, "APM")
	}
	if j.AtaAam != nil {
		f = append(f, "AAM")
	}
	for _, e := range j.AtaLogDir.Table {
		if e.Address == 0x10 {
			f = append(f, "NCQ")
			break
		}
	}
	if j.Trim.Supported {
		f = append(f, "TRIM")
	}
	if j.AtaSmartData.Capabilities.GpLoggingSupported {
		f = append(f, "GPL")
	}
	return f
}

// nvmeAttributes synthesizes CDI's 15 pseudo attributes (IDs 01-0F) from the
// NVMe SMART/Health log so the SMART table and history work for NVMe.
func nvmeAttributes(d *Disk) []Attribute {
	n := d.NVMe
	if n == nil {
		return []Attribute{}
	}
	add := func(id int, name string, cur int, raw string, rawv int64) Attribute {
		return Attribute{ID: id, Name: name, Current: cur, Raw: raw, RawValue: rawv, Status: "good"}
	}
	gb := func(units int64) string {
		return fmt.Sprintf("%d [%s]", units, humanGB(int((units*1000)>>21)))
	}
	pct := func(v int) string { return strconv.Itoa(v) + " %" }
	hours := 0
	if d.PowerOnHours != nil {
		hours = *d.PowerOnHours
	}
	temp := 0
	if d.Temperature != nil {
		temp = *d.Temperature
	}
	return []Attribute{
		add(0x01, "Critical Warning", n.CriticalWarning, fmt.Sprintf("0x%02X", n.CriticalWarning), int64(n.CriticalWarning)),
		add(0x02, "Composite Temperature", temp, fmt.Sprintf("%d °C", temp), int64(temp)),
		add(0x03, "Available Spare", n.AvailableSpare, pct(n.AvailableSpare), int64(n.AvailableSpare)),
		add(0x04, "Available Spare Threshold", n.SpareThreshold, pct(n.SpareThreshold), int64(n.SpareThreshold)),
		add(0x05, "Percentage Used", n.PercentageUsed, pct(n.PercentageUsed), int64(n.PercentageUsed)),
		add(0x06, "Data Units Read", int(n.DataUnitsRead), gb(n.DataUnitsRead), n.DataUnitsRead),
		add(0x07, "Data Units Written", int(n.DataUnitsWritten), gb(n.DataUnitsWritten), n.DataUnitsWritten),
		add(0x08, "Host Read Commands", int(n.HostReads), strconv.FormatInt(n.HostReads, 10), n.HostReads),
		add(0x09, "Host Write Commands", int(n.HostWrites), strconv.FormatInt(n.HostWrites, 10), n.HostWrites),
		add(0x0A, "Controller Busy Time", int(n.ControllerBusy), strconv.FormatInt(n.ControllerBusy, 10), n.ControllerBusy),
		add(0x0B, "Power Cycles", int(n.PowerCycles), strconv.FormatInt(n.PowerCycles, 10), n.PowerCycles),
		add(0x0C, "Power On Hours", hours, strconv.Itoa(hours), int64(hours)),
		add(0x0D, "Unsafe Shutdowns", int(n.UnsafeShutdowns), strconv.FormatInt(n.UnsafeShutdowns, 10), n.UnsafeShutdowns),
		add(0x0E, "Media and Data Integrity Errors", int(n.MediaErrors), strconv.FormatInt(n.MediaErrors, 10), n.MediaErrors),
		add(0x0F, "Number of Error Information Log Entries", int(n.ErrorLogEntries), strconv.FormatInt(n.ErrorLogEntries, 10), n.ErrorLogEntries),
	}
}

// humanGB renders CDI's tooltip-style size ("4.534 TB").
func humanGB(gb int) string {
	switch {
	case gb >= 1024*1024:
		return fmt.Sprintf("%.3f PB", float64(gb)/1024/1024)
	case gb >= 1024:
		return fmt.Sprintf("%.3f TB", float64(gb)/1024)
	default:
		return fmt.Sprintf("%d GB", gb)
	}
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
		RotationRate:  rotationRate(j),
		TransferMode:  transferMode(j),
		Standard:      ataStandard(j),
		Features:      ataFeatures(j),
		Health:        "unknown",
		Attributes:    []Attribute{},
		smartType:     d.Type,
	}
	if j.SmartStatus != nil {
		passed := j.SmartStatus.Passed
		disk.smartPassed = &passed
	}
	// CDI shows "NVM Express" as the interface for NVMe devices.
	if strings.EqualFold(disk.Protocol, "nvme") {
		disk.Protocol = "NVM Express"
	}
	// ponytail: rotation rate 0 is the SSD heuristic; CDI uses Identify data.
	disk.IsSSD = disk.RotationRate == 0 && !strings.Contains(strings.ToUpper(disk.Protocol), "SCSI")

	for _, a := range j.AtaAttrTable.Table {
		raw := a.Raw.String
		if raw == "" && a.Raw.Value != 0 {
			raw = strconv.FormatInt(a.Raw.Value, 10)
		}
		// Status stays "good" here; evaluateHealth derives the CDI LED
		// (good/caution/bad) from current/threshold — when_failed is not used.
		disk.Attributes = append(disk.Attributes, Attribute{
			ID: a.ID, Name: a.Name, Current: a.Value, Worst: a.Worst,
			Threshold: a.Thresh, Raw: raw, RawValue: a.Raw.Value, Status: "good",
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

	// CDI's CheckSsdSupport: pick the Smart* language section and apply the
	// family Host Reads/Writes/NAND/Life rules.
	switch {
	case disk.NVMe != nil:
		disk.SmartKey = "SmartNVMe"
		disk.TransferMode = nvmeLinkMode("/sys", d.Path)
		disk.Attributes = nvmeAttributes(&disk)
		if n := j.NVMeLog; n != nil {
			disk.HostReads = intPtr(int((n.DataUnitsRead * 1000) >> 21))
			disk.HostWrites = intPtr(int((n.DataUnitsWritten * 1000) >> 21))
		}
	case disk.IsSSD || isSsdOld(disk.Model):
		disk.IsSSD = true
		if fam, ok := matchSSDFamily(disk.Model, disk.Firmware, &disk); ok {
			disk.SmartKey = fam.key
			applySSDValues(&disk, fam)
		} else {
			disk.SmartKey = "SmartSsd"
		}
	default:
		disk.SmartKey = "Smart"
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
