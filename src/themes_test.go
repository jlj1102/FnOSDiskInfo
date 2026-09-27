package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"unicode/utf16"
)

func makeZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func importZip(t *testing.T, dir string, z []byte) ([]themeInfo, []string) {
	t.Helper()
	infos, skipped, err := importThemes(dir, bytes.NewReader(z), "")
	if err != nil {
		t.Fatal(err)
	}
	return infos, skipped
}

func readManifest(t *testing.T, dir, id string) themeManifest {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(themesDir(dir), id, "theme.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m themeManifest
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestImportThemeNative(t *testing.T) {
	dir := t.TempDir()
	z := makeZip(t, map[string]string{
		"theme.json": `{"id":"mypack","name":"My Pack","vars":{"--cdi-bg":"#123456"}}`,
		"bg.png":     "x",
	})
	infos, _ := importZip(t, dir, z)
	if len(infos) != 1 || infos[0].ID != "mypack" || infos[0].Name != "My Pack" {
		t.Fatalf("info = %+v", infos)
	}
}

func TestImportThemeZipSlip(t *testing.T) {
	dir := t.TempDir()
	z := makeZip(t, map[string]string{"../evil.png": "x"})
	if _, _, err := importThemes(dir, bytes.NewReader(z), ""); err == nil {
		t.Fatal("expected path traversal rejection")
	}
	if _, err := os.Stat(filepath.Join(dir, "evil.png")); err == nil {
		t.Fatal("file escaped the themes dir")
	}
}

func TestImportThemeRejectsExecutable(t *testing.T) {
	dir := t.TempDir()
	z := makeZip(t, map[string]string{"theme.json": "{}", "run.sh": "echo pwned"})
	if _, _, err := importThemes(dir, bytes.NewReader(z), ""); err == nil {
		t.Fatal("expected extension rejection")
	}
}

func TestImportThemeRejectsForeignIni(t *testing.T) {
	dir := t.TempDir()
	z := makeZip(t, map[string]string{"DiskInfo.ini": "[Setting]\nTheme=x"})
	if _, _, err := importThemes(dir, bytes.NewReader(z), ""); err == nil {
		t.Fatal("expected non-theme ini rejection")
	}
}

// Real CDI theme shape: theme.ini + <asset>-<zoom>.png.
func TestImportCDIZoomPack(t *testing.T) {
	dir := t.TempDir()
	ini := "[Info]\r\nAuthor=hiyohiyo\r\n\r\n[Color];RGB\r\n" +
		"LabelText=0x000000;\r\nListBk1=0xFFFFFF;\r\nListBk2=0xF8F8F8;\r\nListLine1=0xE0E0E0;\r\n" +
		"Glass=0xFFFFFF;\r\n\r\n[Alpha]\r\nGlassAlpha=128;\r\n"
	z := makeZip(t, map[string]string{
		"ShizukuOffice/theme.ini":               ini,
		"ShizukuOffice/ShizukuBackground-100.png": "small",
		"ShizukuOffice/ShizukuBackground-300.png": "big",
		"ShizukuOffice/diskGood-100.png":          "g100",
		"ShizukuOffice/diskGood-300.png":          "g300",
		"ShizukuOffice/diskGoodMini-100.png":      "gm",
		"ShizukuOffice/selectSound-100.png":       "ignored",
	})
	infos, _ := importZip(t, dir, z)
	if len(infos) != 1 || infos[0].ID != "shizukuoffice" {
		t.Fatalf("infos = %+v", infos)
	}
	m := readManifest(t, dir, "shizukuoffice")
	if m.Vars["--cdi-panel"] != "#FFFFFF" || m.Vars["--cdi-bg"] != "#F8F8F8" || m.Vars["--cdi-border"] != "#E0E0E0" {
		t.Fatalf("vars = %v", m.Vars)
	}
	if m.Images["background"] != "ShizukuBackground-300.png" {
		t.Errorf("background = %q", m.Images["background"])
	}
	if m.Images["disk_good"] != "diskGood-100.png" {
		t.Errorf("disk_good = %q", m.Images["disk_good"])
	}
	if m.Images["disk_good_mini"] != "diskGoodMini-100.png" {
		t.Errorf("disk_good_mini = %q", m.Images["disk_good_mini"])
	}
	// unused zoom variants are trimmed
	if _, err := os.Stat(filepath.Join(themesDir(dir), "shizukuoffice", "diskGood-300.png")); err == nil {
		t.Error("unused zoom variant kept")
	}
	// non-image assets are ignored, not copied into the manifest
	if _, ok := m.Images["selectSound"]; ok {
		t.Error("sound asset leaked into manifest")
	}
}

func TestImportMultiThemeZip(t *testing.T) {
	dir := t.TempDir()
	z := makeZip(t, map[string]string{
		"A/theme.ini":  "[Color]\r\nListBk1=0xFFFFFF;\r\n",
		"A/good-100.png": "a",
		"B/bad-100.png":  "b",
	})
	infos, _ := importZip(t, dir, z)
	if len(infos) != 2 {
		t.Fatalf("infos = %+v", infos)
	}
	ids := []string{infos[0].ID, infos[1].ID}
	if ids[0] != "a" || ids[1] != "b" {
		t.Fatalf("ids = %v", ids)
	}
}

func TestImportThemeOverwrite(t *testing.T) {
	dir := t.TempDir()
	z1 := makeZip(t, map[string]string{"X/good-100.png": "v1"})
	importZip(t, dir, z1)
	z2 := makeZip(t, map[string]string{"X/bad-100.png": "v2"})
	infos, _ := importZip(t, dir, z2)
	if len(infos) != 1 || infos[0].ID != "x" {
		t.Fatalf("infos = %+v", infos)
	}
	if _, err := os.Stat(filepath.Join(themesDir(dir), "x", "good-100.png")); err == nil {
		t.Error("old asset survived the overwrite")
	}
	if _, err := os.Stat(filepath.Join(themesDir(dir), "x", "bad-100.png")); err != nil {
		t.Error("new asset missing after overwrite")
	}
}

func TestParseColor(t *testing.T) {
	cases := map[string]string{
		"0x000000;": "#000000",
		"0xFFFFFF":  "#FFFFFF",
		"0x0000FF;": "#0000FF", // COLORREF 0xBBGGRR prints as #RRGGBB
		"16711680":  "#FF0000",
		"#00FF00":   "#00FF00",
		"garbage":   "",
	}
	for in, want := range cases {
		if got := parseColor(in); got != want {
			t.Errorf("parseColor(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseINIUTF16(t *testing.T) {
	src := "[Color]\r\nListBk1=0xFFFFFF;\r\n"
	u16 := utf16.Encode([]rune(src))
	raw := []byte{0xFF, 0xFE}
	for _, v := range u16 {
		raw = append(raw, byte(v), byte(v>>8))
	}
	ini := parseINI(raw)
	if ini.get("color", "listbk1") != "0xFFFFFF;" {
		t.Fatalf("utf16 ini = %v", ini)
	}
}

func TestThemeDeleteGuard(t *testing.T) {
	if !isBuiltinTheme("classic") {
		t.Fatal("classic must be builtin")
	}
	if sanitizeThemeID("classic") != "" {
		t.Fatal("builtin ids must be rejected")
	}
	if got := sanitizeThemeID("ShizukuDark~nijihashi_sola"); got != "shizukudarknijihashi_sola" {
		t.Fatalf("sanitize = %q", got)
	}
}

func TestSlotForBase(t *testing.T) {
	cases := map[string]string{
		"diskGood":           "disk_good",
		"diskStatusGoodMini": "status_good_mini",
		"SDdiskStatusBad":    "sd_bad",
		"temperatureCaution": "temp_caution",
		"noDiskMini":         "nodisk",
		"nextDisk":           "next",
		"ShizukuBackground":  "background",
		"logo":               "logo",
		"good":               "status_good",
		"playSound":          "",
		"ShizukuCopyright":   "",
	}
	for in, want := range cases {
		if got, _ := slotForBase(in); got != want {
			t.Errorf("slotForBase(%q) = %q, want %q", in, got, want)
		}
	}
	if _, zoom := splitZoom("diskGood-300.png"); zoom != 300 {
		t.Error("splitZoom failed")
	}
}

// One CDI pack can carry several art families at once; each must land in its
// own slot deterministically (previously diskGood could overwrite status_good).
func TestSlotPriority(t *testing.T) {
	dir := t.TempDir()
	z := makeZip(t, map[string]string{
		"T/diskGood-100.png":         "btn",
		"T/diskStatusGood-100.png":   "health",
		"T/SDdiskStatusGood-100.png": "life",
		"T/temperatureGood-100.png":  "temp",
		"T/good-100.png":             "generic",
	})
	importZip(t, dir, z)
	m := readManifest(t, dir, "t")
	for slot, want := range map[string]string{
		"disk_good":   "diskGood-100.png",
		"status_good": "diskStatusGood-100.png",
		"sd_good":     "SDdiskStatusGood-100.png",
		"temp_good":   "temperatureGood-100.png",
	} {
		if m.Images[slot] != want {
			t.Errorf("%s = %q, want %q", slot, m.Images[slot], want)
		}
	}
}

func makeStripPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		c := color.NRGBA{R: uint8(255 * y / h), G: 20, B: 40, A: 255}
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// CDI button art is a vertical sprite strip; the importer must split it into
// per-frame files (disk buttons 4 frames, pager 2) like CDI's ButtonFx does.
func TestSplitFrames(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "diskGood-100.png")
	if err := os.WriteFile(src, makeStripPNG(t, 84, 192), 0644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out")
	if err := os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	names := splitPNGFrames(src, out, "disk_good", 4)
	if len(names) != 4 {
		t.Fatalf("names = %v", names)
	}
	for _, n := range names {
		f, err := os.Open(filepath.Join(out, n))
		if err != nil {
			t.Fatal(err)
		}
		img, _, err := image.Decode(f)
		f.Close()
		if err != nil {
			t.Fatal(err)
		}
		if img.Bounds().Dx() != 84 || img.Bounds().Dy() != 48 {
			t.Errorf("%s size = %v", n, img.Bounds())
		}
	}
	if names := splitPNGFrames(src, out, "pre", 5); names != nil {
		t.Errorf("non-divisible strip must stay single, got %v", names)
	}
	if names := splitPNGFrames(src, out, "logo", 1); names != nil {
		t.Errorf("single-frame slot must stay single, got %v", names)
	}
}

func TestImportCDIFrameSplit(t *testing.T) {
	dir := t.TempDir()
	z := makeZip(t, map[string]string{
		"T/diskGood-100.png":         string(makeStripPNG(t, 84, 192)),
		"T/preDisk-100.png":          string(makeStripPNG(t, 24, 48)),
		"T/SDdiskStatusGood-100.png": string(makeStripPNG(t, 128, 192)),
	})
	importZip(t, dir, z)
	m := readManifest(t, dir, "t")
	if m.FrameCount["disk_good"] != 4 || m.FrameCount["pre"] != 2 {
		t.Fatalf("frame_count = %v", m.FrameCount)
	}
	if m.Images["disk_good"] != "disk_good.0.png" || m.Images["pre"] != "pre.0.png" {
		t.Fatalf("images = %v", m.Images)
	}
	if m.Images["sd_good"] != "SDdiskStatusGood-100.png" {
		t.Errorf("sd_good = %q", m.Images["sd_good"])
	}
	if _, err := os.Stat(filepath.Join(themesDir(dir), "t", "disk_good.3.png")); err != nil {
		t.Error("missing selected frame")
	}
	if _, err := os.Stat(filepath.Join(themesDir(dir), "t", "diskGood-100.png")); err == nil {
		t.Error("original strip kept after splitting")
	}
}
