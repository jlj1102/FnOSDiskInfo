package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Theme packs: built-ins are embedded under web/themes/<id>/, imported packs
// live in $TRIM_PKGVAR/themes/<id>/ and are served read-only over /themes/.

type themeInfo struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Builtin bool   `json:"builtin"`
}

// themeManifest is our native format. CDI-style packs without theme.json get a
// best-effort manifest generated from their asset file names.
type themeManifest struct {
	ID     string            `json:"id,omitempty"`
	Name   string            `json:"name,omitempty"`
	Vars   map[string]string `json:"vars,omitempty"`
	CSS    string            `json:"css,omitempty"`
	Images map[string]string `json:"images,omitempty"`
}

var builtinThemes = []themeInfo{
	{ID: "classic", Name: "CrystalDiskInfo Classic", Builtin: true},
	{ID: "dark", Name: "Dark", Builtin: true},
	{ID: "follow", Name: "Follow fnOS", Builtin: true},
}

var themeIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,31}$`)

var allowedThemeExt = map[string]bool{
	".json": true, ".css": true,
	".png": true, ".jpg": true, ".jpeg": true, ".webp": true, ".gif": true, ".svg": true,
}

var themeImageExt = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".webp": true, ".gif": true,
}

const (
	maxThemeZip        = 20 << 20
	maxThemeUnpacked   = 100 << 20
	maxThemeFiles      = 300
	maxThemeFileSize   = 20 << 20
	themeImportTimeout = 30 * time.Second
)

func themesDir(dataDir string) string { return filepath.Join(dataDir, "themes") }

func isBuiltinTheme(id string) bool {
	for _, t := range builtinThemes {
		if t.ID == id {
			return true
		}
	}
	return false
}

func listThemes(dataDir string) []themeInfo {
	out := append([]themeInfo{}, builtinThemes...)
	entries, err := os.ReadDir(themesDir(dataDir))
	if err != nil {
		return out
	}
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		id := e.Name()
		name := id
		if raw, err := os.ReadFile(filepath.Join(themesDir(dataDir), id, "theme.json")); err == nil {
			var m themeManifest
			if json.Unmarshal(raw, &m) == nil && m.Name != "" {
				name = m.Name
			}
		}
		out = append(out, themeInfo{ID: id, Name: name})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func importTheme(dataDir string, body io.Reader) (themeInfo, error) {
	raw, err := io.ReadAll(io.LimitReader(body, maxThemeZip+1))
	if err != nil {
		return themeInfo{}, err
	}
	if len(raw) > maxThemeZip {
		return themeInfo{}, errors.New("theme package exceeds 20 MB")
	}
	zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return themeInfo{}, errors.New("not a valid zip file")
	}
	if len(zr.File) > maxThemeFiles {
		return themeInfo{}, errors.New("too many files in theme package")
	}

	if err := os.MkdirAll(themesDir(dataDir), 0755); err != nil {
		return themeInfo{}, err
	}
	tmp, err := os.MkdirTemp(themesDir(dataDir), ".import-")
	if err != nil {
		return themeInfo{}, err
	}
	defer os.RemoveAll(tmp)

	var total uint64
	for _, f := range zr.File {
		name := f.Name
		if strings.Contains(name, "\\") {
			return themeInfo{}, errors.New("invalid entry name")
		}
		clean := path.Clean(name)
		if clean == "." || clean == "/" {
			continue
		}
		if strings.HasPrefix(clean, "..") || path.IsAbs(clean) || strings.Contains(clean, "../") {
			return themeInfo{}, fmt.Errorf("path traversal in %q", name)
		}
		if f.Mode()&os.ModeSymlink != 0 {
			return themeInfo{}, errors.New("symlinks are not allowed")
		}
		if f.FileInfo().IsDir() {
			continue
		}
		ext := strings.ToLower(path.Ext(clean))
		if !allowedThemeExt[ext] {
			return themeInfo{}, fmt.Errorf("file type not allowed: %s", name)
		}
		total += f.UncompressedSize64
		if total > maxThemeUnpacked {
			return themeInfo{}, errors.New("theme package too large when unpacked")
		}
		if f.UncompressedSize64 > maxThemeFileSize {
			return themeInfo{}, fmt.Errorf("file too large: %s", name)
		}

		dst := filepath.Join(tmp, filepath.FromSlash(clean))
		if !strings.HasPrefix(dst, tmp+string(os.PathSeparator)) {
			return themeInfo{}, fmt.Errorf("path traversal in %q", name)
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return themeInfo{}, err
		}
		rc, err := f.Open()
		if err != nil {
			return themeInfo{}, err
		}
		out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
		if err != nil {
			rc.Close()
			return themeInfo{}, err
		}
		_, err = io.Copy(out, io.LimitReader(rc, maxThemeFileSize+1))
		out.Close()
		rc.Close()
		if err != nil {
			return themeInfo{}, err
		}
	}

	root, manifest, err := prepareThemeDir(tmp)
	if err != nil {
		return themeInfo{}, err
	}

	id := themeIDPattern.FindString(strings.ToLower(manifest.ID))
	if id == "" {
		base := filepath.Base(root)
		if strings.HasPrefix(base, ".") {
			base = "theme-" + strconv.FormatInt(time.Now().Unix(), 10)
		}
		id = sanitizeThemeID(base)
	}
	if id == "" {
		return themeInfo{}, errors.New("cannot determine theme id")
	}
	if isBuiltinTheme(id) {
		return themeInfo{}, errors.New("theme id conflicts with a built-in theme")
	}
	target := filepath.Join(themesDir(dataDir), id)
	if _, err := os.Stat(target); err == nil {
		return themeInfo{}, errors.New("theme already exists: " + id)
	}
	if err := os.Rename(root, target); err != nil {
		return themeInfo{}, err
	}
	name := manifest.Name
	if name == "" {
		name = id
	}
	return themeInfo{ID: id, Name: name}, nil
}

func sanitizeThemeID(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		}
	}
	out := strings.Trim(b.String(), "-_")
	if len(out) > 32 {
		out = out[:32]
	}
	if out == "" || isBuiltinTheme(out) {
		return ""
	}
	return out
}

// prepareThemeDir finds the directory that actually contains the theme files
// (zip may wrap everything in a single folder), parses theme.json, or
// generates a best-effort manifest for CDI-style asset packs.
func prepareThemeDir(tmp string) (string, themeManifest, error) {
	root := findThemeRoot(tmp)
	var m themeManifest
	if raw, err := os.ReadFile(filepath.Join(root, "theme.json")); err == nil {
		if err := json.Unmarshal(raw, &m); err != nil {
			return "", m, errors.New("invalid theme.json")
		}
		return root, m, nil
	}
	m = generateCDIManifest(root)
	if m.Name == "" {
		m.Name = filepath.Base(root)
	}
	for _, v := range m.Vars {
		if strings.Contains(v, "url(") {
			return "", m, errors.New("invalid theme var")
		}
	}
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return "", m, err
	}
	if err := os.WriteFile(filepath.Join(root, "theme.json"), raw, 0644); err != nil {
		return "", m, err
	}
	return root, m, nil
}

func findThemeRoot(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return dir
	}
	subdirs := 0
	root := dir
	for _, e := range entries {
		if e.IsDir() {
			subdirs++
			root = filepath.Join(dir, e.Name())
		}
	}
	if subdirs == 1 {
		if _, err := os.Stat(filepath.Join(root, "theme.json")); err == nil {
			return root
		}
		if subs, err := os.ReadDir(root); err == nil && len(subs) > 0 {
			return root
		}
	}
	return dir
}

// generateCDIManifest maps CrystalDiskInfo-style asset names onto our slots.
// Best effort (AGENTS section 15, L1/L2).
func generateCDIManifest(dir string) themeManifest {
	m := themeManifest{Images: map[string]string{}}
	slot := func(base string) string {
		base = strings.ToLower(base)
		base = strings.NewReplacer(" ", "", "_", "", "-", "").Replace(base)
		switch {
		case strings.Contains(base, "good"):
			return "status_good"
		case strings.Contains(base, "caution"):
			return "status_caution"
		case strings.Contains(base, "bad"):
			return "status_bad"
		case strings.Contains(base, "unknown"):
			return "status_unknown"
		}
		return ""
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return m
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if !themeImageExt[ext] {
			continue
		}
		base := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		low := strings.ToLower(base)
		switch {
		case strings.HasPrefix(low, "background") || low == "bg":
			m.Images["background"] = e.Name()
		case strings.HasPrefix(low, "logo"):
			m.Images["logo"] = e.Name()
		case strings.Contains(low, "character") || strings.HasPrefix(low, "shizuku") || strings.Contains(low, "kurei"):
			if _, ok := m.Images["character"]; !ok {
				m.Images["character"] = e.Name()
			}
		case strings.HasPrefix(low, "temperature") || strings.HasPrefix(low, "temp"):
			if s := slot(low); s != "" {
				m.Images["temp_"+strings.TrimPrefix(s, "status_")] = e.Name()
			}
		case strings.HasPrefix(low, "diskstatus") || strings.HasPrefix(low, "sd"):
			if s := slot(low); s != "" {
				m.Images[s] = e.Name()
			}
		case low == "good" || low == "caution" || low == "bad" || low == "unknown":
			if s := slot(low); s != "" {
				m.Images[s] = e.Name()
			}
		}
	}
	if len(m.Images) == 0 {
		m.Images = nil
	}
	return m
}
