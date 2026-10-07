package main

import (
	"fmt"
	"strconv"
)

// SCSI/SAS disks expose no ATA-style SMART attributes, and CrystalDiskInfo
// has no SCSI log-sense support at all. smartctl's SCSI JSON carries error
// counters, defect lists and cycle counters instead; they are mapped to
// pseudo attributes (IDs 01-07) shown with the Current/Worst/Threshold
// columns hidden, like CDI's NVMe pseudo attributes.

type scsiErrorCounters struct {
	TotalErrorsCorrected   *int64 `json:"total_errors_corrected"`
	TotalUncorrectedErrors *int64 `json:"total_uncorrected_errors"`
}

type scsiStartStop struct {
	Accumulated     *int64 `json:"accumulated_start_stop_cycles"`  // smartctl >= 7.4
	Raw             *int64 `json:"Accumulated start-stop cycles"`  // smartctl 7.3 raw label
	LoadAccumulated *int64 `json:"accumulated_load_unload_cycles"` // smartctl >= 7.4
	LoadRaw         *int64 `json:"Accumulated load-unload cycles"` // smartctl 7.3 raw label
}

// cycles returns the accumulated start-stop count (power-on count).
func (s scsiStartStop) cycles() *int64 {
	if s.Accumulated != nil {
		return s.Accumulated
	}
	return s.Raw
}

// loadUnload returns the accumulated load/unload count.
func (s scsiStartStop) loadUnload() *int64 {
	if s.LoadAccumulated != nil {
		return s.LoadAccumulated
	}
	return s.LoadRaw
}

// scsiAttributes renders the SCSI health data smartctl reports as pseudo
// attributes; unsupported/absent fields produce no row.
func scsiAttributes(j *smartJSON) []Attribute {
	attrs := []Attribute{}
	add := func(id int, name, raw string, value int64) {
		// Current carries the count so history (attr-XX) and the report work;
		// the column itself is hidden for SCSI like NVMe.
		attrs = append(attrs, Attribute{ID: id, Name: name, Current: int(value), Raw: raw, RawValue: value, Status: "good"})
	}
	if j.SmartStatus != nil {
		raw, v := "OK", int64(0)
		if !j.SmartStatus.Passed {
			raw, v = "Failed", 1
		}
		add(0x01, "SMART Health Status", raw, v)
	}
	if j.ScsiGrownDefectList != nil {
		add(0x02, "Grown Defect List", strconv.Itoa(*j.ScsiGrownDefectList), int64(*j.ScsiGrownDefectList))
	}
	if j.ScsiPendingDefects.Count != nil {
		add(0x03, "Pending Defects", strconv.Itoa(*j.ScsiPendingDefects.Count), int64(*j.ScsiPendingDefects.Count))
	}
	errRow := func(id int, name string, c scsiErrorCounters) {
		if c.TotalErrorsCorrected == nil && c.TotalUncorrectedErrors == nil {
			return
		}
		corr, unc := int64(0), int64(0)
		if c.TotalErrorsCorrected != nil {
			corr = *c.TotalErrorsCorrected
		}
		if c.TotalUncorrectedErrors != nil {
			unc = *c.TotalUncorrectedErrors
		}
		add(id, name, fmt.Sprintf("corrected %d / uncorrected %d", corr, unc), unc)
	}
	errRow(0x04, "Read Errors", j.ScsiErrorCounterLog.Read)
	errRow(0x05, "Write Errors", j.ScsiErrorCounterLog.Write)
	errRow(0x06, "Verify Errors", j.ScsiErrorCounterLog.Verify)
	if c := j.ScsiStartStop.loadUnload(); c != nil {
		add(0x07, "Load/Unload Cycles", strconv.FormatInt(*c, 10), *c)
	}
	return attrs
}

// scsiAttributeStatus mirrors the NVMe pseudo-attribute LED rules: error and
// defect counts only go caution, cycle counts stay good. The disk health
// class itself still follows smart_status only.
func scsiAttributeStatus(a *Attribute) string {
	switch a.ID {
	case 0x01:
		if a.Raw == "Failed" {
			return "bad"
		}
	case 0x02, 0x03, 0x04, 0x05, 0x06:
		if a.RawValue > 0 {
			return "caution"
		}
	}
	return "good"
}
