package main

import (
	"strings"
	"testing"
)

// Identify Controller bits from CDI (AtaSmart.cpp): ONCS byte 520 bit 2 =
// Dataset Management (TRIM), byte 525 bit 0 = Volatile Write Cache.
func TestNVMeIdentifyFeatures(t *testing.T) {
	buf := make([]byte, 4096)
	buf[520] = 0x04
	buf[525] = 0x01
	if trim, vwc := nvmeIdentifyFeatures(buf); !trim || !vwc {
		t.Fatalf("trim=%v vwc=%v", trim, vwc)
	}
	buf[520], buf[525] = 0, 0
	if trim, vwc := nvmeIdentifyFeatures(buf); trim || vwc {
		t.Fatalf("clear buffer: trim=%v vwc=%v", trim, vwc)
	}
	if trim, vwc := nvmeIdentifyFeatures(make([]byte, 10)); trim || vwc {
		t.Fatal("short buffer must report false")
	}
}

func TestNVMeFeatures(t *testing.T) {
	if got := strings.Join(nvmeFeatures(true, true, true), ","); got != "S.M.A.R.T.,TRIM,VolatileWriteCache" {
		t.Fatalf("features = %q", got)
	}
	if got := strings.Join(nvmeFeatures(true, false, false), ","); got != "S.M.A.R.T." {
		t.Fatalf("features = %q", got)
	}
	if got := nvmeControllerDev("/dev/nvme0n1"); got != "/dev/nvme0" {
		t.Fatalf("controller dev = %q", got)
	}
}
