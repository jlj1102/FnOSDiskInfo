package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// SSD family detection and value rules ported from CrystalDiskInfo
// (AtaSmart.cpp: CheckSsdSupport + the IsSsd* predicates and the
// Life/Host Reads/Host Writes/NAND Writes switch). A family decides the
// Smart* language section used for attribute names, the host/nand units and
// the per-attribute value rules.

type rwUnit int

const (
	rwUnknown rwUnit = iota
	rw512b
	rw1mb
	rw16mb
	rw32mb
	rwGB
)

type nandUnit int

const (
	nandGB nandUnit = iota
	nand1MB
)

type ssdFamily struct {
	key    string // Smart* language section
	vendor string
	units  rwUnit
	nand   nandUnit
	// CDI's FlagLife* / FlagLifeSanDisk* flags
	lifeRaw      bool
	lifeRawIncr  bool
	lifeNoReport bool
	sd0_1        bool
	sd1          bool
	sdLenovo     bool
	sdCloud      bool
	sdUsb        bool
	samsungEnt   bool
}

func up(s string) string { return strings.ToUpper(s) }

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

func prefixAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.HasPrefix(s, sub) {
			return true
		}
	}
	return false
}

// hasSig reports whether the attribute list starts with the given ID sequence
// (CDI's Attribute[0..n].Id checks).
func hasSig(ids []int, sig ...int) bool {
	if len(ids) < len(sig) {
		return false
	}
	for i, v := range sig {
		if ids[i] != v {
			return false
		}
	}
	return true
}

func attrIDs(d *Disk) []int {
	out := make([]int, 0, len(d.Attributes))
	for i := range d.Attributes {
		out = append(out, d.Attributes[i].ID)
	}
	return out
}

var sigSamsungSM951 = []int{0x05, 0x09, 0x0C, 0xAA, 0xAB, 0xAC, 0xAD, 0xAE, 0xB2, 0xB4}
var sigSamsungA = []int{0x09, 0x0C, 0xB2, 0xB3, 0xB4}
var sigSamsungB = []int{0x09, 0x0C, 0xB1, 0xB2, 0xB3, 0xB4, 0xB7}
var sigSamsungC = []int{0x09, 0x0C, 0xAF, 0xB0, 0xB1, 0xB2, 0xB3, 0xB4}
var sigSamsungD = []int{0x05, 0x09, 0x0C, 0xB1, 0xB3, 0xB5, 0xB6}

var sigIntelA = []int{0x03, 0x04, 0x05, 0x09, 0x0C}
var sigMicronA = []int{0x01, 0x05, 0x09, 0x0C, 0xAA, 0xAB, 0xAC, 0xAD, 0xAE, 0xB5, 0xB7}

var sigSandForceA = []int{0x01, 0x05, 0x09, 0x0C, 0x0D, 0x64, 0xAA}
var sigSandForceB = []int{0x01, 0x05, 0x09, 0x0C, 0xAB, 0xAC}
var sigSandForceC = []int{0x01, 0x02, 0x03, 0x05, 0x07, 0x08, 0x09, 0x0A, 0x0C, 0xA7, 0xA8, 0xA9, 0xAA, 0xAD, 0xAF, 0xB1}

var sigOczA = []int{0x01, 0x03, 0x04, 0x05, 0x09, 0x0C, 0xE8, 0xE9}
var sigOczVectorA = []int{0x05, 0x09, 0x0C, 0xAB, 0xAE, 0xC3, 0xC4, 0xC5, 0xC6}

var sigSiliconMotionA = []int{0x01, 0x05, 0x09, 0x0C, 0xA0, 0xA1, 0xA3, 0xA4, 0xA5, 0xA6, 0xA7, 0xA8, 0xA9, 0xAF, 0xB0, 0xB1, 0xB2, 0xB5, 0xB6, 0xC0}
var sigSiliconMotionB = []int{0x01, 0x05, 0x09, 0x0C, 0xA0, 0xA1, 0xA3, 0xA4, 0xA5, 0xA6, 0xA7, 0x94, 0x95, 0x96, 0x97, 0xA9, 0xB1, 0xB5, 0xB6, 0xBB}
var sigSiliconMotionC = []int{0x01, 0x05, 0x09, 0x0C, 0x94, 0x95, 0x96, 0x97, 0x9F, 0xA0, 0xA1}
var sigSiliconMotionD = []int{0x01, 0x05, 0x09, 0x0C, 0xA0, 0xA1, 0xA3, 0xA4, 0xA5, 0xA6, 0xA7}
var sigSiliconMotionE = []int{0x01, 0x05, 0x09, 0x0C, 0xA0, 0xA1, 0xA3, 0x94, 0x95, 0x96, 0x97}

var sigPhisonA = []int{0x01, 0x09, 0x0C, 0xA8, 0xAA, 0xAD, 0xC0, 0xC2, 0xDA, 0xE7, 0xF1}
var sigPhisonB = []int{0x01, 0x09, 0x0C, 0xA8, 0xAA, 0xAD, 0xC0, 0xDA, 0xE7, 0xF1}

var sigSeagateIronWolf = []int{0x01, 0x05, 0x09, 0x0C, 0x64, 0x66, 0x67, 0xAA, 0xAB, 0xAC, 0xAD, 0xAE, 0xB1, 0xB7, 0xBB}
var sigSeagateSsd = []int{0x01, 0x09, 0x0C, 0x10, 0x11, 0xA8, 0xAA, 0xAD, 0xAE, 0xB1, 0xC0, 0xC2, 0xDA, 0xE7, 0xE8, 0xE9, 0xEB, 0xF1, 0xF2}

var sigMarvellA = []int{0x05, 0x09, 0x0C, 0xA1, 0xA4, 0xA5, 0xA6, 0xA7}
var sigMarvellB = []int{0x05, 0x09, 0x0C, 0xA4, 0xA5, 0xA6, 0xA7}

var sigMaxiotekA = []int{0x05, 0x09, 0x0C, 0xA4, 0xA5, 0xA6, 0xA7}
var sigMaxiotekB = []int{0x05, 0x09, 0x0C, 0xA7, 0xA8, 0xA9}

var sigRealtekA = []int{0x01, 0x05, 0x09, 0x0C, 0xA1, 0xA2, 0xA3, 0xA4, 0xA6, 0xA7}

var sigJMicron66x = []int{0x01, 0x02, 0x03, 0x05, 0x07, 0x08, 0x09, 0x0A, 0x0C, 0xA7, 0xA8, 0xA9, 0xAA, 0xAD, 0xAF}
var sigJMicron61x = []int{0x01, 0x02, 0x03, 0x05, 0x07, 0x08, 0x09, 0x0A, 0x0C, 0xA8, 0xAF, 0xC0, 0xC2}
var sigJMicron60x = []int{0x0C, 0x09, 0xC2, 0xE5, 0xE8, 0xE9}
var sigIndilinx = []int{0x01, 0x09, 0x0C, 0xB8, 0xC3, 0xC4}
var sigPlextor = []int{0x01, 0x05, 0x09, 0x0C, 0xB1, 0xB2, 0xB5, 0xB6}

func samsungEnterprise(model string) bool {
	return containsAny(model,
		"SM863", "PM863", "PM863A", "SM883", "PM883", "SM843T", "PM853T",
		"MZ7KM", "MZ7KH", "MZ7LM", "MZ7LH")
}

// matchSSDFamily mirrors CDI's CheckSsdSupport order. isSSD comes from the
// rotation-rate heuristic; isSsdOld (CDI's model list) may upgrade it.
func matchSSDFamily(model, firmware string, d *Disk) (ssdFamily, bool) {
	m := up(model)
	f := up(firmware)
	ids := attrIDs(d)

	switch {
	case prefixAny(m, "ADATA_IM2S", "ADATA_IMSS", "ADATA_ISSS", "IM2S", "IMSS", "ISSS"):
		return ssdFamily{key: "SmartAdataIndustrial", vendor: "adata_industrial", units: rw512b}, true

	case containsAny(m, "SANDISK", "SD ULTRA", "SDLF1"):
		fam := ssdFamily{key: "SmartSanDisk", vendor: "sandisk", units: rwGB, sd1: true}
		switch {
		case containsAny(m, "X600") && strings.Contains(m, "2280"), containsAny(m, "X400", "X300", "X110", "SD5"):
			fam.units = rw512b
			fam.sd1 = true
			if ids[2] == 0xAF || ids[3] == 0xAF {
				fam.key = "SmartSanDiskDell"
			} else {
				fam.key = "SmartSanDiskGb"
			}
		case containsAny(m, "Z400"):
			fam.key = "SmartSanDiskDell"
		case strings.Contains(m, "1006"):
			fam.units = rw16mb
			if strings.Contains(m, "8U") {
				fam.key, fam.vendor = "SmartSanDiskHpVenus", "sandisk_hp_venus"
			} else {
				fam.key, fam.vendor = "SmartSanDiskHp", "sandisk_hp"
			}
		case strings.Contains(m, "G1001"):
			fam.sdLenovo = true
			if strings.Contains(m, "6S") || strings.Contains(m, "7S") || strings.Contains(m, "8U") {
				fam.key, fam.vendor = "SmartSanDiskLenovoHelenVenus", "sandisk_lenovo_helen_venus"
			} else if strings.Contains(m, "SD9SB") {
				fam.key, fam.nand = "SmartSanDiskGb", nand1MB
			} else {
				fam.key, fam.vendor = "SmartSanDiskLenovo", "sandisk_lenovo"
			}
		case strings.Contains(m, "G1012"), strings.Contains(m, "Z400s 2.5"):
			fam.key, fam.vendor = "SmartSanDiskDell", "sandisk_dell"
		case strings.Contains(m, "SSD P4"):
			fam.units, fam.sdUsb = rw512b, true
		case strings.Contains(m, "iSSD P4"):
			fam.key, fam.units = "SmartSanDiskGb", rw512b
		case containsAny(m, "SDSSDP", "SDSSDRC"):
			fam.units, fam.sd1, fam.sd0_1 = rw512b, false, true
		case containsAny(m, "SSD U100", "SSD U110", "SSD I100", "SSD I110", "PSSD"):
			fam.units, fam.sd1, fam.sdUsb = rw512b, false, true
		case containsAny(m, "SDLF1CRR-", "SDLF1DAR-", "SDLF1CRM-", "SDLF1DAM-"):
			fam.key, fam.vendor, fam.units = "SmartSanDiskCloud", "sandisk_cloud", rwGB
			fam.sd1, fam.sdCloud = false, true
		}
		return fam, true

	case prefixAny(m, "WDC ", "WD "):
		units := rwGB
		if strings.Contains(m, "SA530") {
			units = rw16mb
		}
		return ssdFamily{key: "SmartWdc", vendor: "wdc", units: units}, true

	case hasSig(ids, sigSeagateIronWolf...):
		return ssdFamily{key: "SmartSeagateIronWolf", vendor: "seagate", units: rwGB}, true
	case hasSig(ids, sigSeagateSsd...):
		return ssdFamily{key: "SmartSeagate", vendor: "seagate", units: rwGB, lifeRaw: true}, true
	case prefixAny(m, "SEAGATE"), (strings.HasPrefix(m, "ST") && !strings.HasPrefix(m, "STT")), strings.HasPrefix(m, "ZA"):
		fam := ssdFamily{key: "SmartSeagate", vendor: "seagate", units: rwGB, lifeRaw: true}
		if strings.Contains(m, "BARRACUDA") {
			fam.key = "SmartSeagateBarraCuda"
		} else if strings.Contains(m, "HM") || strings.Contains(m, "FP") {
			fam.lifeRaw = false
		}
		return fam, true

	case (len(ids) > 0 && ids[0] == 0xBB && len(ids) == 1), prefixAny(m, "MTRON"):
		return ssdFamily{key: "SmartMtron", vendor: "mtron"}, true

	case strings.Contains(m, "TOSHIBA") && d.IsSSD:
		units := rwGB
		if containsAny(m, "THNSNC", "THNSNJ", "THNSNK", "KSG60", "TL100", "TR150", "TR200") {
			units = rw32mb
		}
		return ssdFamily{key: "SmartToshiba", vendor: "toshiba", units: units}, true

	case hasSig(ids, sigJMicron66x...), prefixAny(m, "ADATA SU700"):
		return ssdFamily{key: "SmartJMicron66x", vendor: "jmicron"}, true
	case hasSig(ids, sigJMicron61x...):
		return ssdFamily{key: "SmartJMicron61x", vendor: "jmicron"}, true
	case hasSig(ids, sigJMicron60x...):
		return ssdFamily{key: "SmartJMicron60x", vendor: "jmicron"}, true
	case hasSig(ids, sigIndilinx...):
		return ssdFamily{key: "SmartIndilinx", vendor: "indilinx"}, true

	case strings.Contains(m, "INTEL SSDSCKHB"):
		return ssdFamily{key: "SmartIntelDc", vendor: "intel_dc"}, true
	case hasSig(ids, sigIntelA...) &&
		((len(ids) > 7 && ids[5] == 0xC0 && ids[6] == 0xE8 && ids[7] == 0xE9) ||
			(len(ids) > 6 && ids[5] == 0xC0 && ids[6] == 0xE1) ||
			(len(ids) > 7 && ids[5] == 0xAA && ids[6] == 0xAB && ids[7] == 0xAC)),
		containsAny(m, "INTEL", "SOLIDIGM"):
		return ssdFamily{key: "SmartIntel", vendor: "intel"}, true

	case hasSig(ids, sigSamsungSM951...):
		return ssdFamily{key: "SmartSamsung", vendor: "samsung", units: rwGB}, true
	case hasSig(ids, sigSamsungA...), hasSig(ids, sigSamsungB...),
		hasSig(ids, sigSamsungC...), hasSig(ids, sigSamsungD...):
		return ssdFamily{key: "SmartSamsung", vendor: "samsung"}, true
	case (containsAny(m, "SAMSUNG", "MZ-") && d.IsSSD):
		return ssdFamily{key: "SmartSamsung", vendor: "samsung", samsungEnt: samsungEnterprise(m)}, true

	case prefixAny(m, "MICRON_M600", "MICRON M600", "MICRON_M550", "MICRON M550",
		"MICRON_M510", "MICRON M510", "MICRON_M500", "MICRON M500",
		"MICRON_1300", "MICRON 1300", "MICRON_1100", "MICRON 1100", "MTFDDA"):
		return ssdFamily{key: "SmartMicronMU03", vendor: "micron_mu03", units: rw512b}, true
	case (containsAny(m, "M500SSD", "MX500SSD", "BX500SSD", "MX300SSD", "BX300SSD", "MX200SSD", "BX200SSD", "MX100SSD", "BX100SSD") || prefixAny(m, "MTFD")) && !strings.Contains(f, "MU01"):
		return ssdFamily{key: "SmartMicronMU03", vendor: "micron_mu03", units: rw32mb}, true

	case hasSig(ids, sigMicronA...),
		prefixAny(m, "P600", "C600", "M6-", "M600", "P500", "M5-", "M500", "P400", "C400", "M4-", "M400", "P300", "C300", "M3-", "M300", "CRUCIAL", "MICRON", "MTFD"),
		(strings.HasPrefix(m, "C500") && !strings.HasPrefix(f, "H")),
		(strings.HasPrefix(m, "CT") && strings.Contains(m, "SSD")):
		return ssdFamily{key: "SmartMicron", vendor: "micron"}, true

	case hasSig(ids, sigSandForceA...), hasSig(ids, sigSandForceB...), hasSig(ids, sigSandForceC...),
		strings.Contains(m, "SANDFORCE"):
		return ssdFamily{key: "SmartSandForce", vendor: "sandforce"}, true

	case strings.HasPrefix(m, "OCZ-TRION"), hasSig(ids, sigOczA...):
		if strings.HasPrefix(m, "OCZ") {
			return ssdFamily{key: "SmartOcz", vendor: "ocz"}, true
		}
		return ssdFamily{}, false

	case strings.HasPrefix(m, "RADEON R7"), hasSig(ids, sigOczVectorA...), strings.HasPrefix(m, "PANASONIC RP-SSB"):
		return ssdFamily{key: "SmartOczVector", vendor: "ocz_vector"}, true
	case strings.HasPrefix(m, "OCZ"):
		return ssdFamily{key: "SmartOczVector", vendor: "ocz_vector"}, true

	case containsAny(m, "CV8-", "CVB-", "ER2-"):
		return ssdFamily{key: "SmartSsstc", vendor: "ssstc"}, true

	case hasSig(ids, sigPlextor...),
		prefixAny(m, "PLEXTOR", "LITEON", "CV6-CQ", "CSSD-S6T128NM3PQ", "CSSD-S6T256NM3PQ"):
		return ssdFamily{key: "SmartPlextor", vendor: "plextor"}, true

	case strings.Contains(m, "KINGSTON"):
		switch {
		case containsAny(m, "SM2280", "SEDC400", "SKC310", "SHSS", "SUV300", "SKC400"):
			return ssdFamily{key: "SmartKingston", vendor: "kingston", units: rwGB}, true
		case strings.Contains(m, "SA400"):
			fam := ssdFamily{key: "SmartKingstonSA400", vendor: "kingston", units: rwGB, lifeRaw: true}
			if strings.HasPrefix(f, "03070009") {
				fam.lifeRaw = false
			}
			return fam, true
		case strings.Contains(m, "KC600"):
			return ssdFamily{key: "SmartKingstonKC600", vendor: "kingston", units: rw32mb}, true
		case strings.Contains(m, "DC500"):
			return ssdFamily{key: "SmartKingstonDC500", vendor: "kingston", units: rwGB}, true
		case containsAny(m, "SUV400", "SUV500"):
			return ssdFamily{key: "SmartKingstonSuv", vendor: "kingston", units: rwGB}, true
		}

	case strings.HasPrefix(m, "CORSAIR"):
		units := rwUnknown
		if strings.Contains(m, "VOYAGER GTX") {
			units = rw1mb
		}
		return ssdFamily{key: "SmartCorsair", vendor: "corsair", units: units}, true

	case hasSig(ids, sigRealtekA...):
		return ssdFamily{key: "SmartRealtek", vendor: "realtek", units: rwGB}, true

	case containsAny(m, "SK HYNIX"), prefixAny(m, "HFS", "SHG"):
		fam := ssdFamily{key: "SmartSKhynix", vendor: "skhynix", units: rwGB}
		switch {
		case strings.Contains(m, "HFS") && (strings.Contains(m, "TND") || strings.Contains(m, "MND")):
			fam.lifeRawIncr = true
		case strings.Contains(m, "HFS") && strings.Contains(m, "TNF"):
			fam.lifeRaw = true
		case containsAny(m, "SC311", "SC401"):
			fam.units, fam.lifeRaw = rw512b, true
		}
		return fam, true

	case strings.Contains(m, "KIOXIA"):
		return ssdFamily{key: "SmartKioxia", vendor: "kioxia", units: rw32mb}, true
	case strings.Contains(m, "CVC-"):
		return ssdFamily{key: "SmartSiliconMotionCVC", vendor: "siliconmotion_cvc", units: rwGB}, true

	case hasSig(ids, sigSiliconMotionA...), hasSig(ids, sigSiliconMotionB...),
		hasSig(ids, sigSiliconMotionC...), hasSig(ids, sigSiliconMotionD...),
		hasSig(ids, sigSiliconMotionE...), prefixAny(m, "ADATA SX950"),
		strings.HasPrefix(m, "TS"):
		fam := ssdFamily{key: "SmartSiliconMotion", vendor: "siliconmotion", lifeRaw: true}
		switch {
		case strings.HasPrefix(m, "SSD") && strings.HasPrefix(f, "FW"), // Goldenfir: disabled in CDI
			prefixAny(m, "WT200", "WT100", "WT "), strings.HasPrefix(m, "TECMIYO"),
			strings.HasPrefix(m, "ADATA SU650") && strings.HasPrefix(f, "XD0R3C0A"):
			fam.lifeRaw = false
		}
		return fam, true

	case hasSig(ids, sigPhisonA...), hasSig(ids, sigPhisonB...):
		fam := ssdFamily{key: "SmartPhison", vendor: "phison", units: rwGB, lifeRaw: true}
		if strings.HasPrefix(m, "AIGO PSSD P6") {
			fam.lifeRaw = false
		}
		if strings.HasPrefix(f, "S9") {
			fam.units = rw1mb
		}
		return fam, true

	case hasSig(ids, sigMarvellA...), hasSig(ids, sigMarvellB...):
		if strings.HasPrefix(m, "HANYE-Q55") {
			return ssdFamily{}, false
		}
		units := rwGB
		if strings.HasPrefix(m, "LEXAR") && (strings.HasPrefix(f, "SN0") || strings.HasPrefix(f, "V6")) {
			units = rw32mb
		}
		return ssdFamily{key: "SmartMarvell", vendor: "marvell", units: units}, true

	case prefixAny(m, "MAXIO"):
		return ssdFamily{key: "SmartMaxiotek", vendor: "maxiotek", units: rwGB}, true
	case prefixAny(m, "CUSO C5S-EVO"):
		return ssdFamily{key: "SmartMaxiotek", vendor: "maxiotek", units: rw512b}, true
	case strings.HasPrefix(m, "HANYE-Q55") && hasSig(ids, sigMaxiotekA...),
		hasSig(ids, sigMaxiotekB...):
		return ssdFamily{key: "SmartMaxiotek", vendor: "maxiotek"}, true

	case prefixAny(m, "APACER", "ZADAK"), prefixAny(f, "AP", "SF", "PN"):
		return ssdFamily{key: "SmartApacer", vendor: "apacer", units: rw512b, lifeRaw: true}, true
	case strings.Contains(m, "ZHITAI"):
		return ssdFamily{key: "SmartYmtc", vendor: "ymtc", units: rw512b}, true
	case strings.HasPrefix(m, "SCY"):
		return ssdFamily{key: "SmartScy", vendor: "scy", units: rw32mb}, true
	case strings.HasPrefix(m, "RECADATA"):
		return ssdFamily{key: "SmartRecadata", vendor: "recadata", units: rwGB}, true
	}

	return ssdFamily{}, false
}

// isSsdOld is CDI's IsSsdOld model list: marks old drives as SSD even without
// a usable rotation rate.
func isSsdOld(model string) bool {
	m := up(model)
	return prefixAny(m, "OCZ", "SPCC", "PATRIOT", "PHOTOFAST", "STT_FTM", "SUPER TALENT",
		"SANDFORCE", "HANYE", "INTEL", "TOSHIBA THNS", "CSSD") ||
		containsAny(m, "SOLID", "SSD", "SILICONHARDDISK")
}

func unitsToGB(u rwUnit, raw uint64) int {
	switch u {
	case rw512b:
		return int(raw / 2 / 1024 / 1024)
	case rw1mb:
		return int(raw / 1024)
	case rw16mb:
		return int(raw / 64)
	case rw32mb:
		return int(raw / 32)
	case rwGB:
		return int(uint32(raw))
	}
	return -1
}

func raw64(a *Attribute) uint64 { return uint64(a.RawValue) }
func raw32(a *Attribute) int    { return int(uint32(a.RawValue)) }

// applySSDValues ports CDI's Life/Host Reads/Host Writes/NAND switch.
func applySSDValues(d *Disk, fam ssdFamily) {
	setLife := func(v int) {
		if v <= 0 || v > 100 {
			v = -1
		}
		d.Life = &v
	}
	byID := func(id int) *Attribute {
		for i := range d.Attributes {
			if d.Attributes[i].ID == id {
				return &d.Attributes[i]
			}
		}
		return nil
	}

	for i := range d.Attributes {
		a := &d.Attributes[i]
		switch a.ID {
		case 0xBB:
			if fam.vendor == "mtron" {
				setLife(a.Current)
			}
		case 0xCA:
			if containsAny(fam.vendor, "micron", "micron_mu03", "intel_dc", "siliconmotion_cvc") {
				setLife(a.Current)
			}
		case 0xD1:
			if fam.vendor == "indilinx" {
				setLife(a.Current)
			}
		case 0xC9:
			if containsAny(fam.vendor, "sandisk_hp", "sandisk_hp_venus") {
				setLife(a.Current)
			}
		case 0xE6:
			if fam.vendor == "wdc" || fam.vendor == "sandisk" {
				life := -1
				switch {
				case fam.sdUsb:
				case fam.sd0_1:
					life = 100 - (int((raw64(a)>>8)&0xFF)*256+int(raw64(a)&0xFF))/100
				case fam.sdLenovo:
					life = a.Current
				default:
					life = 100 - int((raw64(a)>>8)&0xFF)
				}
				if life <= 0 {
					life = -1
				}
				d.Life = &life
			} else if fam.vendor == "sandisk_lenovo" || fam.vendor == "sandisk_dell" {
				setLife(a.Current)
			}
		case 0xE8:
			if fam.vendor == "plextor" {
				setLife(a.Current)
			} else if fam.vendor == "ocz" {
				d.HostWrites = intPtr(int(raw64(a) / 2 / 1024 / 1024))
			}
		case 0xE9:
			switch {
			case containsAny(fam.vendor, "intel", "ocz", "ocz_vector", "skhynix"):
				if fam.lifeRaw {
					setLife(raw32(a))
				} else {
					setLife(a.Current)
				}
			case fam.vendor == "sandisk_lenovo_helen_venus":
				setLife(a.Current)
			case fam.vendor == "samsung" && fam.samsungEnt:
				setLife(a.Current)
			case containsAny(fam.vendor, "sandisk", "sandisk_lenovo", "sandisk_cloud") && fam.units == rwGB:
				if fam.nand == nand1MB {
					d.NandWrites = intPtr(int(raw64(a) / 1024))
				} else {
					d.NandWrites = intPtr(raw32(a))
				}
			case containsAny(fam.vendor, "plextor", "kingston", "wdc", "ssstc", "seagate", "siliconmotion_cvc"):
				d.NandWrites = intPtr(raw32(a))
			case containsAny(fam.vendor, "jmicron", "adata_industrial"):
				d.NandWrites = intPtr(int(raw64(a) / 2 / 1024 / 1024))
			case fam.vendor == "maxiotek":
				if fam.units == rw512b {
					d.NandWrites = intPtr(int(raw64(a) / 2 / 1024 / 1024))
				} else {
					d.NandWrites = intPtr(raw32(a))
				}
			}
		case 0xE1:
			if fam.vendor == "intel" {
				d.HostWrites = intPtr(int(raw64(a) / 32))
			}
		case 0xEA:
			if containsAny(fam.vendor, "kingston", "seagate") || (fam.vendor == "skhynix" && fam.units == rwGB) {
				d.NandWrites = intPtr(raw32(a))
			}
		case 0xEB:
			if fam.vendor == "intel_dc" {
				d.HostWrites = intPtr(int(raw64(a) / 32))
			}
		case 0xF1:
			switch {
			case fam.vendor == "" || fam.key == "SmartSsd":
				d.HostWrites = intPtr(unitsToGB(fam.units, raw64(a)))
			case fam.vendor == "toshiba" && fam.units == rwGB,
				fam.vendor == "siliconmotion_cvc" && fam.units == rwGB:
				d.HostWrites = intPtr(raw32(a))
			case containsAny(fam.vendor, "intel", "toshiba", "kioxia", "siliconmotion"):
				d.HostWrites = intPtr(int(raw64(a) / 32))
			case containsAny(fam.vendor, "sandforce", "ocz_vector", "corsair", "kingston", "realtek", "wdc", "ssstc", "skhynix", "phison", "seagate", "marvell", "maxiotek", "ymtc", "scy", "recadata", "micron_mu03", "sandisk_hp", "sandisk_hp_venus", "sandisk_lenovo", "sandisk_lenovo_helen_venus", "sandisk_dell", "adata_industrial"):
				if fam.units == rwGB {
					d.HostWrites = intPtr(raw32(a))
				} else {
					d.HostWrites = intPtr(unitsToGB(fam.units, raw64(a)))
				}
			case fam.vendor == "samsung" && fam.units == rwGB:
				d.HostWrites = intPtr(raw32(a))
			case containsAny(fam.vendor, "samsung", "apacer", "jmicron"):
				d.HostWrites = intPtr(int(raw64(a) / 2 / 1024 / 1024))
			case fam.vendor == "plextor":
				d.HostWrites = intPtr(int(raw64(a) / 32))
			case fam.vendor == "sandisk" && fam.units == rwGB:
				d.HostWrites = intPtr(raw32(a))
			case fam.vendor == "sandisk":
				d.HostWrites = intPtr(int(raw64(a) / 2 / 1024 / 1024))
			case containsAny(fam.vendor, "micron", "micron_mu03"):
				setLife(a.Current)
			}
		case 0xF2:
			switch {
			case fam.vendor == "" || fam.key == "SmartSsd":
				d.HostReads = intPtr(unitsToGB(fam.units, raw64(a)))
			case fam.vendor == "toshiba" && fam.units == rwGB,
				fam.vendor == "siliconmotion_cvc" && fam.units == rwGB:
				d.HostReads = intPtr(raw32(a))
			case containsAny(fam.vendor, "intel", "toshiba", "siliconmotion"):
				d.HostReads = intPtr(int(raw64(a) / 32))
			case containsAny(fam.vendor, "sandforce", "ocz_vector", "corsair", "kingston", "realtek", "wdc", "ssstc", "skhynix", "seagate", "marvell", "maxiotek", "ymtc", "scy", "recadata", "micron_mu03", "sandisk_hp", "sandisk_hp_venus", "sandisk_lenovo", "sandisk_lenovo_helen_venus", "sandisk_dell", "adata_industrial"):
				if fam.units == rwGB {
					d.HostReads = intPtr(raw32(a))
				} else {
					d.HostReads = intPtr(unitsToGB(fam.units, raw64(a)))
				}
			case fam.vendor == "samsung" && fam.units == rwGB:
				d.HostReads = intPtr(raw32(a))
			case containsAny(fam.vendor, "samsung", "jmicron"):
				d.HostReads = intPtr(int(raw64(a) / 2 / 1024 / 1024))
			case fam.vendor == "plextor":
				d.HostReads = intPtr(int(raw64(a) / 32))
			case fam.vendor == "sandisk" && fam.units == rwGB:
				d.HostReads = intPtr(raw32(a))
			case fam.vendor == "sandisk":
				d.HostReads = intPtr(int(raw64(a) / 2 / 1024 / 1024))
			}
		case 0xF3:
			if fam.vendor == "intel" {
				d.NandWrites = intPtr(int(raw64(a) / 32))
			}
		case 0xF9:
			switch {
			case containsAny(fam.vendor, "intel", "realtek", "wdc"),
				fam.vendor == "sandisk" && fam.units == rwGB,
				containsAny(fam.vendor, "sandisk_hp", "sandisk_hp_venus", "sandisk_lenovo_helen_venus"):
				d.NandWrites = intPtr(raw32(a))
			case fam.vendor == "ocz_vector":
				d.NandWrites = intPtr(int(raw64(a) * 16 / 1024 / 1024))
			}
		case 0xFA:
			if fam.vendor == "realtek" {
				d.NandWrites = intPtr(raw32(a))
			}
		case 0x64:
			if fam.vendor == "sandforce" {
				// GBytesErased → the rotation row label in CDI; not shown here
				_ = a
			}
		case 0xAD:
			if containsAny(fam.vendor, "toshiba", "kioxia") {
				setLife(a.Current - 100)
			}
		case 0xB1:
			if fam.vendor == "samsung" {
				setLife(a.Current)
			}
		case 0xE7:
			switch fam.vendor {
			case "sandforce", "corsair", "kingston", "skhynix", "realtek", "sandisk",
				"ssstc", "apacer", "jmicron", "phison", "seagate", "maxiotek", "ymtc",
				"scy", "recadata", "adata_industrial":
				switch {
				case fam.lifeNoReport:
					v := -1
					d.Life = &v
				case fam.lifeRawIncr:
					setLife(100 - raw32(a))
				case fam.lifeRaw:
					setLife(raw32(a))
				default:
					setLife(a.Current)
				}
			}
		case 0xA9:
			if fam.vendor == "realtek" || (fam.vendor == "kingston" && fam.units == rw32mb) || fam.vendor == "siliconmotion" {
				switch {
				case fam.lifeRawIncr:
					setLife(100 - raw32(a))
				case fam.lifeRaw:
					setLife(raw32(a))
				default:
					setLife(a.Current)
				}
			}
		case 0xC6:
			if fam.vendor == "ocz_vector" {
				d.HostReads = intPtr(raw32(a))
			}
		case 0xC7:
			if fam.vendor == "ocz_vector" {
				d.HostWrites = intPtr(raw32(a))
			}
		case 0xF5:
			switch fam.vendor {
			case "sandisk_cloud":
				setLife(a.Current)
			case "micron":
				d.NandWrites = intPtr(int(raw64(a) * 8 / 1024 / 1024))
			case "micron_mu03":
				d.NandWrites = intPtr(int(raw64(a) / 32))
			case "kingston":
				if fam.units == rw32mb {
					d.NandWrites = intPtr(int(raw64(a) / 32))
				}
			case "siliconmotion", "scy":
				d.NandWrites = intPtr(int(raw64(a) / 32))
			case "recadata":
				d.NandWrites = intPtr(raw32(a))
			}
		case 0xF6:
			if containsAny(fam.vendor, "micron", "micron_mu03") {
				d.HostWrites = intPtr(int(raw64(a) / 2 / 1024 / 1024))
			}
		}
	}
	_ = byID
}

func intPtr(v int) *int { return &v }

// nvmeLinkMode mirrors CDI's GetTransferModePCIe/SlotSpeedToString
// (SlotSpeedGetter.cpp) using the PCIe link info smartctl's JSON does not
// carry: /sys/class/nvme/<ctrl>/device/{current,max}_link_{speed,width}.
func nvmeLinkMode(sysRoot, devPath string) string {
	ctrl := nvmeController(devPath)
	if ctrl == "" {
		return ""
	}
	base := filepath.Join(sysRoot, "class", "nvme", ctrl, "device")
	cur := pcieLink(filepath.Join(base, "current_link_speed"), filepath.Join(base, "current_link_width"))
	max := pcieLink(filepath.Join(base, "max_link_speed"), filepath.Join(base, "max_link_width"))
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

var nvmeCtrlRe = regexp.MustCompile(`^(nvme\d+)`)

// nvmeController maps /dev/nvme0 or /dev/nvme0n1 to the sysfs controller name.
func nvmeController(devPath string) string {
	return nvmeCtrlRe.FindString(filepath.Base(devPath))
}

func pcieLink(speedPath, widthPath string) string {
	speed, err := os.ReadFile(speedPath)
	if err != nil {
		return ""
	}
	width, err := os.ReadFile(widthPath)
	if err != nil {
		return ""
	}
	spec := pcieSpec(strings.TrimSpace(string(speed)))
	w, err := strconv.Atoi(strings.TrimSpace(string(width)))
	if spec == 0 || err != nil || w <= 0 {
		return "----"
	}
	return "PCIe " + strconv.Itoa(spec) + ".0 x" + strconv.Itoa(w)
}

// pcieSpec maps the sysfs link speed to the PCIe generation (CDI style).
func pcieSpec(speed string) int {
	switch {
	case strings.HasPrefix(speed, "2.5"):
		return 1
	case strings.HasPrefix(speed, "5.0"):
		return 2
	case strings.HasPrefix(speed, "8.0"):
		return 3
	case strings.HasPrefix(speed, "16.0"):
		return 4
	case strings.HasPrefix(speed, "32.0"):
		return 5
	case strings.HasPrefix(speed, "64.0"):
		return 6
	}
	return 0
}
