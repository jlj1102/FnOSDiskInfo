package main

import "fmt"

// evaluateHealth mirrors the essentials of CrystalDiskInfo's CheckDiskStatus:
// - ATA: attribute below its threshold -> bad; 05/C5/C6 raw above the
//   configurable caution threshold -> caution; SSD life below FF -> caution.
// - NVMe: critical warning bits -> bad; available spare vs threshold; life.
func (d *Disk) evaluateHealth(s diskSettings) {
	d.AlarmTemp = s.AlarmTemp
	d.Life = computeLife(d)
	d.StatusReasons = nil

	if d.Error != "" || d.smartPassed == nil {
		d.Health = "unknown"
		return
	}

	var reasons []string

	if d.NVMe != nil {
		status := ""
		n := d.NVMe
		if n.CriticalWarning != 0 {
			status = "bad"
			reasons = append(reasons, nvmeWarningReasons(n.CriticalWarning)...)
		}
		if n.SpareThreshold > 0 && n.SpareThreshold <= 100 && n.AvailableSpare > 0 {
			switch {
			case n.AvailableSpare < n.SpareThreshold:
				status = "bad"
				reasons = append(reasons, fmt.Sprintf("Available spare %d%% below threshold %d%%", n.AvailableSpare, n.SpareThreshold))
			case n.AvailableSpare == n.SpareThreshold && status != "bad":
				if status == "" {
					status = "caution"
				}
				reasons = append(reasons, fmt.Sprintf("Available spare %d%% reached threshold", n.AvailableSpare))
			}
		}
		if d.Life != nil && *d.Life <= s.ThresholdFF && status != "bad" {
			status = "caution"
			reasons = append(reasons, fmt.Sprintf("Life %d%% <= %d%%", *d.Life, s.ThresholdFF))
		}
		if status == "" {
			if *d.smartPassed {
				status = "good"
			} else {
				status = "bad"
				reasons = append(reasons, "SMART status failed")
			}
		}
		d.Health = status
		d.StatusReasons = reasons
		return
	}

	errors, caution, thresholds := 0, false, 0
	for i := range d.Attributes {
		a := &d.Attributes[i]
		a.Status = attributeStatus(a, d.IsSSD, s)
		if a.Threshold > 0 {
			thresholds++
			if a.Current < a.Threshold {
				errors++
				reasons = append(reasons, fmt.Sprintf("%s: %d < threshold %d", a.Name, a.Current, a.Threshold))
			}
		}
		if d.IsSSD {
			continue
		}
		var limit int
		switch a.ID {
		case 0x05:
			limit = s.Threshold05
		case 0xC5:
			limit = s.ThresholdC5
		case 0xC6:
			limit = s.ThresholdC6
		}
		raw := a.RawValue & 0xFFFF
		if limit > 0 && raw != 0xFFFF && raw >= int64(limit) {
			caution = true
			reasons = append(reasons, fmt.Sprintf("%s: raw %d >= caution threshold %d", a.Name, raw, limit))
		}
	}
	if d.IsSSD && d.Life != nil && *d.Life <= s.ThresholdFF {
		caution = true
		reasons = append(reasons, fmt.Sprintf("Life %d%% <= %d%%", *d.Life, s.ThresholdFF))
	}

	switch {
	case errors > 0:
		d.Health = "bad"
	case !*d.smartPassed:
		d.Health = "bad"
		reasons = append(reasons, "SMART status failed")
	case caution:
		d.Health = "caution"
	case len(d.Attributes) == 0 && thresholds == 0:
		d.Health = "unknown"
	default:
		d.Health = "good"
	}
	d.StatusReasons = reasons
}

// attributeStatus is CDI's per-attribute LED (DiskInfoDlgUpdate.cpp
// UpdateListCtrl): temperature is always good; 05/C5/C6 go bad below the ATA
// threshold and caution above the configured raw limit; the standard ATA
// attribute ranges only go bad below a non-zero threshold; everything else
// stays good. when_failed ("In_the_past" etc.) is intentionally ignored.
func attributeStatus(a *Attribute, isSSD bool, s diskSettings) string {
	if a.ID == 0xC2 {
		return "good"
	}
	if !isSSD && (a.ID == 0x05 || a.ID == 0xC5 || a.ID == 0xC6) {
		if a.Threshold > 0 && a.Current < a.Threshold {
			return "bad"
		}
		var limit int
		switch a.ID {
		case 0x05:
			limit = s.Threshold05
		case 0xC5:
			limit = s.ThresholdC5
		case 0xC6:
			limit = s.ThresholdC6
		}
		if limit > 0 && a.RawValue&0xFFFF != 0xFFFF && a.RawValue&0xFFFF >= int64(limit) {
			return "caution"
		}
		return "good"
	}
	if a.Threshold > 0 && a.Current < a.Threshold && cdiAttrRange(a.ID) {
		return "bad"
	}
	return "good"
}

// cdiAttrRange lists the attribute IDs CDI checks against the ATA threshold;
// IDs outside these ranges are forced good (source: DiskInfoDlgUpdate.cpp).
func cdiAttrRange(id int) bool {
	switch {
	case id >= 0x01 && id <= 0x0D,
		id == 0x16,
		id >= 0xBB && id <= 0xC1,
		id >= 0xC3 && id <= 0xD1,
		id >= 0xD3 && id <= 0xD4,
		id >= 0xDC && id <= 0xE4,
		id >= 0xE6 && id <= 0xE7,
		id == 0xF0, id == 0xFA, id == 0xFE:
		return true
	}
	return false
}

// computeLife mirrors CDI's Life percentage: NVMe = 100 - percentage_used;
// ATA SSD = a known wear attribute, checked vendor-blind.
// ponytail: vendor-blind order, upgrade to CDI's per-vendor table if a disk
// reports a too optimistic/pessimistic value.
func computeLife(d *Disk) *int {
	if d.NVMe != nil {
		v := 100 - d.NVMe.PercentageUsed
		if v < 0 {
			v = 0
		}
		if v > 100 {
			v = 100
		}
		return &v
	}
	if !d.IsSSD {
		return nil
	}
	wearIDs := []int{0xE7, 0xAD, 0xCA, 0xE9, 0xB1, 0xA9, 0xE8, 0xF1}
	for _, id := range wearIDs {
		for i := range d.Attributes {
			a := &d.Attributes[i]
			if a.ID == id && a.Current >= 1 && a.Current <= 100 {
				v := a.Current
				return &v
			}
		}
	}
	return nil
}

var nvmeWarningBits = []struct {
	bit int
	msg string
}{
	{0x01, "Available spare below threshold"},
	{0x02, "Temperature above/below threshold"},
	{0x04, "NVM subsystem reliability degraded"},
	{0x08, "Media in read-only mode"},
	{0x10, "Volatile memory backup failed"},
	{0x20, "Persistent memory read-only"},
}

func nvmeWarningReasons(cw int) []string {
	var out []string
	for _, w := range nvmeWarningBits {
		if cw&w.bit != 0 {
			out = append(out, w.msg)
		}
	}
	return out
}
