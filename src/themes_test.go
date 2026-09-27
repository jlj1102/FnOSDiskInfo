package main

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
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

func TestImportThemeNative(t *testing.T) {
	dir := t.TempDir()
	z := makeZip(t, map[string]string{
		"theme.json": `{"id":"mypack","name":"My Pack","vars":{"--cdi-bg":"#123456"}}`,
		"bg.png":     "notreallyapng",
	})
	info, err := importTheme(dir, bytes.NewReader(z))
	if err != nil {
		t.Fatal(err)
	}
	if info.ID != "mypack" || info.Name != "My Pack" {
		t.Fatalf("info = %+v", info)
	}
	if _, err := os.Stat(filepath.Join(themesDir(dir), "mypack", "theme.json")); err != nil {
		t.Fatal(err)
	}
}

func TestImportThemeZipSlip(t *testing.T) {
	dir := t.TempDir()
	z := makeZip(t, map[string]string{"../evil.png": "x"})
	if _, err := importTheme(dir, bytes.NewReader(z)); err == nil {
		t.Fatal("expected path traversal rejection")
	}
	if _, err := os.Stat(filepath.Join(dir, "evil.png")); err == nil {
		t.Fatal("file escaped the themes dir")
	}
}

func TestImportThemeRejectsExecutable(t *testing.T) {
	dir := t.TempDir()
	z := makeZip(t, map[string]string{"theme.json": "{}", "run.sh": "echo pwned"})
	if _, err := importTheme(dir, bytes.NewReader(z)); err == nil {
		t.Fatal("expected extension rejection")
	}
}

func TestImportCDIPack(t *testing.T) {
	dir := t.TempDir()
	z := makeZip(t, map[string]string{
		"MyTheme/good.png":       "a",
		"MyTheme/bad.png":        "b",
		"MyTheme/background.png": "c",
	})
	info, err := importTheme(dir, bytes.NewReader(z))
	if err != nil {
		t.Fatal(err)
	}
	if info.ID != "mytheme" {
		t.Fatalf("id = %q", info.ID)
	}
	raw, err := os.ReadFile(filepath.Join(themesDir(dir), "mytheme", "theme.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "status_good") || !strings.Contains(string(raw), "background") {
		t.Fatalf("generated manifest = %s", raw)
	}
}

func TestThemeDeleteGuard(t *testing.T) {
	if !isBuiltinTheme("classic") {
		t.Fatal("classic must be builtin")
	}
	if sanitizeThemeID("classic") != "" {
		t.Fatal("builtin ids must be rejected")
	}
}
