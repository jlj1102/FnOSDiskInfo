package main

import (
	"fmt"
	"strings"
	"time"
)

func fmtCapacity(bytes int64) string {
	if bytes <= 0 {
		return "-"
	}
	gb := float64(bytes) / 1e9
	if gb >= 1000 {
		return fmt.Sprintf("%.2f TB", gb/1000)
	}
	return fmt.Sprintf("%.1f GB", gb)
}

// buildReport renders a CrystalDiskInfo-style text report.
func buildReport(resp disksResponse) string {
	var b strings.Builder
	fmt.Fprintf(&b, "DiskInfo for fnOS %s\r\n", version)
	fmt.Fprintf(&b, "Date : %s\r\n\r\n", time.Now().Format("2006/01/02 15:04:05"))

	b.WriteString("-- Disk List ---------------------------------------------------------------\r\n")
	for i, d := range resp.Disks {
		fmt.Fprintf(&b, " (%02d) %s : %s [%s]\r\n", i+1, d.Model, fmtCapacity(d.CapacityBytes), d.Device)
	}

	for i, d := range resp.Disks {
		label := d.Model
		if label == "" {
			label = d.Device
		}
		b.WriteString("\r\n============================================================================\r\n")
		fmt.Fprintf(&b, " (%02d) %s : %s\r\n", i+1, label, fmtCapacity(d.CapacityBytes))
		b.WriteString(strings.Repeat("-", 76) + "\r\n")
		row := func(k, v string) { fmt.Fprintf(&b, " %-12s: %s\r\n", k, v) }
		row("Health", d.Health)
		if d.Life != nil {
			row("Life", fmt.Sprintf("%d %%", *d.Life))
		}
		if d.Temperature != nil {
			row("Temperature", fmt.Sprintf("%d C (alarm %d C)", *d.Temperature, d.AlarmTemp))
		}
		row("Firmware", d.Firmware)
		row("Serial", d.Serial)
		row("Interface", d.Protocol)
		if d.PowerOnHours != nil {
			row("Power On", fmt.Sprintf("%d hours", *d.PowerOnHours))
		}
		if d.PowerOnCount != nil {
			row("Power On Count", fmt.Sprintf("%d", *d.PowerOnCount))
		}
		if d.RotationRate > 0 {
			row("Rotation", fmt.Sprintf("%d RPM", d.RotationRate))
		}
		row("Device", d.Device)
		if d.Error != "" {
			row("Error", d.Error)
		}
		for _, r := range d.StatusReasons {
			row("Reason", r)
		}
		if d.SelfTest != nil {
			row("Self-Test", fmt.Sprintf("%s: %s", d.SelfTest.Type, d.SelfTest.Status))
		}

		if len(d.Attributes) > 0 {
			b.WriteString("\r\n-- S.M.A.R.T. --------------------------------------------------------------\r\n")
			b.WriteString(" ID Cur Wor Thr Raw               Attribute\r\n")
			for _, a := range d.Attributes {
				fmt.Fprintf(&b, " %02X %3d %3d %3d %-17s %s\r\n", a.ID, a.Current, a.Worst, a.Threshold, a.Raw, a.Name)
			}
		}
	}
	return b.String()
}
