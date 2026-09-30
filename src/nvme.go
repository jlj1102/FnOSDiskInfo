package main

// NVMe Identify Controller parsing shared by the platform-specific ioctl
// implementations. The two bits used by the feature row are ported from
// CrystalDiskInfo (AtaSmart.cpp:4001-4009): ONCS byte 520 bit 2 = Dataset
// Management (TRIM), byte 525 bit 0 = Volatile Write Cache.

func nvmeIdentifyFeatures(buf []byte) (trim, vwc bool) {
	if len(buf) <= 525 {
		return false, false
	}
	return buf[520]&0x04 != 0, buf[525]&0x01 != 0
}

// nvmeFeatures mirrors CDI's NVMe feature list order (S.M.A.R.T., TRIM,
// VolatileWriteCache — DiskInfoDlgUpdate.cpp).
func nvmeFeatures(smart, trim, vwc bool) []string {
	f := []string{}
	if smart {
		f = append(f, "S.M.A.R.T.")
	}
	if trim {
		f = append(f, "TRIM")
	}
	if vwc {
		f = append(f, "VolatileWriteCache")
	}
	return f
}

// nvmeControllerDev maps /dev/nvme0 or /dev/nvme0n1 to the controller node.
func nvmeControllerDev(devPath string) string {
	ctrl := nvmeController(devPath)
	if ctrl == "" {
		return ""
	}
	return "/dev/" + ctrl
}
