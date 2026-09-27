package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
	Status    string `json:"status"` // ok | bad
}

// Disk is the normalized snapshot served to the frontend.
type Disk struct {
	ID            string      `json:"id"`
	Device        string      `json:"device"`
	Model         string      `json:"model"`
	Serial        string      `json:"serial"`
	Firmware      string      `json:"firmware"`
	Protocol      string      `json:"protocol"`
	CapacityBytes int64       `json:"capacity_bytes"`
	Temperature   *int        `json:"temperature"`
	PowerOnHours  *int        `json:"power_on_hours"`
	RotationRate  int         `json:"rotation_rate"`
	Health        string      `json:"health"` // good | bad | unknown
	Error         string      `json:"error,omitempty"`
	Attributes    []Attribute `json:"attributes,omitempty"`
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
		if err != nil || st.IsDir() || st.Mode()&0111 == 0 {
			continue
		}
		if out, err := exec.Command(p, "--version").Output(); err == nil && len(out) > 0 {
			return p
		}
	}
	return ""
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
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, smartctl, "--scan-open").Output()
	if len(out) == 0 && err != nil {
		return nil, err
	}
	devs := parseScanOutput(string(out))
	if len(devs) == 0 {
		return nil, errors.New("no devices found by smartctl --scan-open")
	}
	return devs, nil
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
	NVMe struct {
		Temperature  *int `json:"temperature"`
		PowerOnHours *int `json:"power_on_hours"`
	} `json:"nvme_smart_health_information_log"`
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
func readSmart(smartctl string, d Device) (*smartJSON, error) {
	args := []string{"-j", "-x"}
	if d.Type != "" {
		args = append(args, "-d", d.Type)
	}
	args = append(args, d.Path)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, smartctl, args...).Output()

	var j smartJSON
	if len(out) == 0 {
		if err != nil {
			return nil, err
		}
		return nil, errors.New("smartctl returned no output")
	}
	if e := json.Unmarshal(out, &j); e != nil {
		return nil, e
	}
	if j.Device.Name == "" {
		msg := "smartctl could not read device"
		for _, m := range j.Smartctl.Messages {
			if m.Severity == "error" {
				msg = m.String
				break
			}
		}
		return nil, errors.New(msg)
	}
	return &j, nil
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
		RotationRate:  j.Device.RotationRate,
		Health:        "unknown",
	}
	// ponytail: health is pass/fail only. Add a CDI-style percentage built
	// from worst attribute degradation if users ask for it.
	switch {
	case j.SmartStatus == nil:
	case j.SmartStatus.Passed:
		disk.Health = "good"
	default:
		disk.Health = "bad"
	}
	if disk.Temperature == nil && j.NVMe.Temperature != nil {
		disk.Temperature = j.NVMe.Temperature
	}
	if disk.PowerOnHours == nil && j.NVMe.PowerOnHours != nil {
		disk.PowerOnHours = j.NVMe.PowerOnHours
	}
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
			Threshold: a.Thresh, Raw: raw, Status: status,
		})
	}
	return disk
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
