package main

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"
)

// Theme packs: built-ins are embedded under web/themes/<id>/, imported packs
// live in $TRIM_PKGVAR/themes/<id>/ and are served read-only over /themes/.
//
// Two pack formats are accepted:
//   - native: theme.json + optional style.css/images
//   - CrystalDiskInfo: theme.ini ([Color]/[Alpha]/[Info]) + "<asset>-<zoom>.png"
//     files; a best-effort manifest is generated (AGENTS section 15, L1/L2).

type themeInfo struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Builtin bool   `json:"builtin"`
}

type themeManifest struct {
	ID         string            `json:"id,omitempty"`
	Name       string            `json:"name,omitempty"`
	Vars       map[string]string `json:"vars,omitempty"`
	CSS        string            `json:"css,omitempty"`
	Images     map[string]string `json:"images,omitempty"`
	FrameCount map[string]int    `json:"frame_count,omitempty"`
}

var builtinThemes = []themeInfo{
	{ID: "classic", Name: "CrystalDiskInfo Classic", Builtin: true},
	{ID: "dark", Name: "Dark", Builtin: true},
	{ID: "follow", Name: "Follow fnOS", Builtin: true},
}

var themeIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,31}$`)

var allowedThemeExt = map[string]bool{
	".json": true, ".css": true, ".ini": true,
	".png": true, ".jpg": true, ".jpeg": true, ".webp": true, ".gif": true, ".svg": true, ".ico": true, ".bmp": true,
}

var themeImageExt = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".webp": true, ".gif": true, ".ico": true, ".bmp": true,
}

const (
	maxThemeZip      = 64 << 20
	maxThemeUnpacked = 256 << 20
	maxThemeFiles    = 2000
	maxThemeFileSize = 32 << 20
)

func isBuiltinTheme(id string) bool {
	for _, t := range builtinThemes {
		if t.ID == id {
			return true
		}
	}
	return false
}

func themesDir(dataDir string) string { return filepath.Join(dataDir, "themes") }

func listThemes(dataDir string) []themeInfo {
	out := append([]themeInfo{}, builtinThemes...)
	entries, err := os.ReadDir(themesDir(dataDir))
	if err != nil {
		return out
	}
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") || strings.Contains(e.Name(), ".old-") {
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

// importThemes imports one or more theme folders from a zip. Multi-folder zips
// (for example an archive of the whole themes directory) import every theme.
// nameHint (the uploaded file name) is used when the zip has no folder to name
// the theme after.
func importThemes(dataDir string, body io.Reader, nameHint string) ([]themeInfo, []string, error) {
	tmpZip, err := os.CreateTemp("", "cdifn-theme-*.zip")
	if err != nil {
		return nil, nil, err
	}
	defer os.Remove(tmpZip.Name())
	defer tmpZip.Close()

	n, err := io.Copy(tmpZip, io.LimitReader(body, maxThemeZip+1))
	if err != nil {
		return nil, nil, err
	}
	if n > maxThemeZip {
		return nil, nil, errors.New("theme package exceeds 64 MB")
	}
	zr, err := zip.OpenReader(tmpZip.Name())
	if err != nil {
		return nil, nil, errors.New("not a valid zip file")
	}
	defer zr.Close()
	if len(zr.File) > maxThemeFiles {
		return nil, nil, errors.New("too many files in theme package")
	}

	if err := os.MkdirAll(themesDir(dataDir), 0755); err != nil {
		return nil, nil, err
	}
	staging, err := os.MkdirTemp(themesDir(dataDir), ".import-")
	if err != nil {
		return nil, nil, err
	}
	defer os.RemoveAll(staging)

	var total uint64
	for _, f := range zr.File {
		name := f.Name
		if strings.Contains(name, "\\") {
			return nil, nil, errors.New("invalid entry name")
		}
		clean := path.Clean(name)
		if clean == "." || clean == "/" {
			continue
		}
		if strings.HasPrefix(clean, "..") || path.IsAbs(clean) || strings.Contains(clean, "../") {
			return nil, nil, fmt.Errorf("path traversal in %q", name)
		}
		if f.Mode()&os.ModeSymlink != 0 {
			return nil, nil, errors.New("symlinks are not allowed")
		}
		if f.FileInfo().IsDir() {
			continue
		}
		ext := strings.ToLower(path.Ext(clean))
		if !allowedThemeExt[ext] {
			return nil, nil, fmt.Errorf("file type not allowed: %s", name)
		}
		if ext == ".ini" && !strings.EqualFold(path.Base(clean), "theme.ini") {
			return nil, nil, fmt.Errorf("only theme.ini is allowed, got %s", name)
		}
		total += f.UncompressedSize64
		if total > maxThemeUnpacked {
			return nil, nil, errors.New("theme package too large when unpacked")
		}
		if f.UncompressedSize64 > maxThemeFileSize {
			return nil, nil, fmt.Errorf("file too large: %s", name)
		}

		dst := filepath.Join(staging, filepath.FromSlash(clean))
		if !strings.HasPrefix(dst, staging+string(os.PathSeparator)) {
			return nil, nil, fmt.Errorf("path traversal in %q", name)
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return nil, nil, err
		}
		rc, err := f.Open()
		if err != nil {
			return nil, nil, err
		}
		out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
		if err != nil {
			rc.Close()
			return nil, nil, err
		}
		_, err = io.Copy(out, io.LimitReader(rc, maxThemeFileSize+1))
		out.Close()
		rc.Close()
		if err != nil {
			return nil, nil, err
		}
	}

	roots := findThemeRoots(staging)
	if len(roots) == 0 {
		return nil, nil, errors.New("no theme found in package")
	}

	byName := map[string]string{}
	for _, r := range roots {
		byName[strings.ToLower(filepath.Base(r))] = r
	}

	var imported []themeInfo
	var skipped []string
	for _, root := range roots {
		info, err := importOne(dataDir, root, byName, nameHint)
		if err != nil {
			skipped = append(skipped, filepath.Base(root)+": "+err.Error())
			continue
		}
		imported = append(imported, info)
	}
	if len(imported) == 0 {
		if len(skipped) > 0 {
			return nil, skipped, errors.New(strings.Join(skipped, "; "))
		}
		return nil, nil, errors.New("no theme imported")
	}
	return imported, skipped, nil
}

// findThemeRoots locates directories that contain theme files, unwrapping a
// single wrapper directory if needed.
func findThemeRoots(dir string) []string {
	if hasThemeFiles(dir) {
		return []string{dir}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var roots []string
	var subdirs []string
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		subdirs = append(subdirs, filepath.Join(dir, e.Name()))
	}
	for _, sub := range subdirs {
		if hasThemeFiles(sub) {
			roots = append(roots, sub)
		}
	}
	if len(roots) == 0 && len(subdirs) == 1 {
		return findThemeRoots(subdirs[0])
	}
	return roots
}

func hasThemeFiles(dir string) bool {
	if _, err := os.Stat(filepath.Join(dir, "theme.ini")); err == nil {
		return true
	}
	if _, err := os.Stat(filepath.Join(dir, "theme.json")); err == nil {
		return true
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() && themeImageExt[strings.ToLower(filepath.Ext(e.Name()))] {
			return true
		}
	}
	return false
}

type themeFile struct {
	name string
	dir  string
}

func importOne(dataDir, root string, byName map[string]string, nameHint string) (themeInfo, error) {
	merged, err := mergeThemeChain(root, byName, map[string]bool{})
	if err != nil {
		return themeInfo{}, err
	}

	base := filepath.Base(root)
	flat := strings.HasPrefix(base, ".") || base == "." || base == string(os.PathSeparator)
	hint := strings.TrimSpace(strings.TrimSuffix(nameHint, ".zip"))
	if len(hint) > 80 {
		hint = hint[:80]
	}

	id := themeIDPattern.FindString(strings.ToLower(merged.ID))
	if id == "" {
		nameForID := base
		if flat && hint != "" {
			nameForID = hint
		}
		id = sanitizeThemeID(nameForID)
	}
	if id == "" {
		id = "theme-" + strconv.FormatInt(time.Now().Unix(), 10)
	}
	if isBuiltinTheme(id) {
		return themeInfo{}, errors.New("id conflicts with a built-in theme")
	}
	name := merged.Name
	if name == "" {
		switch {
		case flat && hint != "":
			name = hint
		case flat:
			name = id
		default:
			name = base
		}
	}

	// materialize referenced images into the root dir, then write theme.json.
	// CDI button art is a vertical sprite strip (normal/hover/focus/selected);
	// split it into per-frame files so the web UI can crop like CDI does.
	refs := map[string]bool{}
	images := map[string]string{}
	frameCount := map[string]int{}
	for slot, f := range merged.Images {
		src := filepath.Join(f.dir, f.name)
		if names := splitPNGFrames(src, root, slot, frameCountForSlot(slot)); names != nil {
			images[slot] = names[0]
			frameCount[slot] = len(names)
			for _, n := range names {
				refs[n] = true
			}
			continue
		}
		dst := filepath.Join(root, f.name)
		if f.dir != root {
			if err := copyFile(src, dst); err != nil {
				continue
			}
		}
		images[slot] = f.name
		refs[f.name] = true
	}
	manifest := themeManifest{ID: id, Name: name, Vars: merged.Vars, CSS: merged.CSS, Images: images, FrameCount: frameCount}
	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return themeInfo{}, err
	}
	if err := os.WriteFile(filepath.Join(root, "theme.json"), raw, 0644); err != nil {
		return themeInfo{}, err
	}
	trimUnusedFiles(root, refs)

	target := filepath.Join(themesDir(dataDir), id)
	if _, err := os.Stat(target); err == nil {
		// overwrite: move the old theme aside first so a failed rename keeps it
		backup := target + fmt.Sprintf(".old-%d", time.Now().Unix())
		if err := os.Rename(target, backup); err != nil {
			return themeInfo{}, err
		}
		if err := os.Rename(root, target); err != nil {
			_ = os.Rename(backup, target)
			return themeInfo{}, err
		}
		_ = os.RemoveAll(backup)
	} else if err := os.Rename(root, target); err != nil {
		return themeInfo{}, err
	}
	return themeInfo{ID: id, Name: name}, nil
}

type mergedTheme struct {
	ID     string
	Name   string
	Vars   map[string]string
	CSS    string
	Images map[string]themeFile
}

// mergeThemeChain walks ParentTheme1/2 references declared in theme.ini,
// overlaying child values on top of parent values (CDI fallback order).
func mergeThemeChain(root string, byName map[string]string, seen map[string]bool) (mergedTheme, error) {
	out := mergedTheme{Vars: map[string]string{}, Images: map[string]themeFile{}}

	ini := iniFile{}
	if raw, err := os.ReadFile(filepath.Join(root, "theme.ini")); err == nil {
		ini = parseINI(raw)
	}

	// 1. parents first (parent 2 then parent 1), shallow recursion with cycle guard
	var parents []string
	for _, key := range []string{"parenttheme1", "parenttheme2"} {
		if v := ini.get("setting", key); v != "" {
			parents = append(parents, v)
		}
	}
	if len(parents) >= 2 {
		parents[0], parents[1] = parents[1], parents[0]
	}
	for _, p := range parents {
		pr, ok := byName[strings.ToLower(strings.TrimSpace(p))]
		if !ok || pr == root || seen[pr] {
			continue
		}
		seen[pr] = true
		pm, err := mergeThemeChain(pr, byName, seen)
		if err != nil {
			continue
		}
		for k, v := range pm.Vars {
			out.Vars[k] = v
		}
		for k, v := range pm.Images {
			out.Images[k] = v
		}
	}

	// 2. native theme.json wins outright if present
	if raw, err := os.ReadFile(filepath.Join(root, "theme.json")); err == nil {
		var m themeManifest
		if json.Unmarshal(raw, &m) == nil {
			out.ID = m.ID
			out.Name = m.Name
			out.CSS = m.CSS
			for k, v := range m.Vars {
				out.Vars[k] = v
			}
			for slot, file := range m.Images {
				out.Images[slot] = themeFile{name: file, dir: root}
			}
		}
		return out, nil
	}

	// 3. theme.ini colors
	out.Vars = mergeVars(out.Vars, themeIniVars(ini))

	// 4. CDI image assets (deterministic slot priority; disk button art must
	// never collide with status/temperature/life art)
	entries, err := os.ReadDir(root)
	if err != nil {
		return out, err
	}
	type variant struct {
		name string
		zoom int
		prio int
	}
	best := map[string]variant{} // key: slot
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if !themeImageExt[ext] {
			continue
		}
		base, zoom := splitZoom(e.Name())
		slot, prio := slotForBase(base)
		if slot == "" {
			continue
		}
		cur, ok := best[slot]
		switch {
		case !ok, prio > cur.prio:
			best[slot] = variant{name: e.Name(), zoom: zoom, prio: prio}
		case prio == cur.prio && preferZoom(slot, zoom, cur.zoom):
			best[slot] = variant{name: e.Name(), zoom: zoom, prio: prio}
		}
	}
	for slot, v := range best {
		out.Images[slot] = themeFile{name: v.name, dir: root}
	}
	return out, nil
}

// preferZoom keeps the largest zoom for backdrop art and the smallest for
// icons (keeps imported packs small; browsers scale the rest).
func preferZoom(slot string, zoom, cur int) bool {
	if slot == "background" || slot == "character" {
		return zoom > cur
	}
	return zoom < cur
}

func mergeVars(base, over map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range base {
		out[k] = v
	}
	for k, v := range over {
		out[k] = v
	}
	return out
}

// themeIniVars maps CDI [Color]/[Alpha] entries onto our CSS variables.
func themeIniVars(ini iniFile) map[string]string {
	vars := map[string]string{}
	set := func(iniKey, cssVar string) {
		if v := parseColor(ini.get("color", iniKey)); v != "" {
			vars[cssVar] = v
		}
	}
	set("labeltext", "--cdi-text")
	set("buttontext", "--cdi-text")
	set("listtext1", "--cdi-text")
	set("listtext2", "--cdi-text")
	set("listbk1", "--cdi-panel")
	set("listbk2", "--cdi-bg")
	set("listline2", "--cdi-border")
	set("listline1", "--cdi-border")
	set("combobk", "--cdi-panel")
	set("listtextselected", "--cdi-accent")
	set("combotextselected", "--cdi-accent")
	if alpha := ini.get("alpha", "glassalpha"); alpha != "" {
		if n, err := strconv.Atoi(strings.TrimSpace(strings.TrimSuffix(alpha, ";"))); err == nil && n >= 0 && n < 255 {
			vars["--cdi-panel-alpha"] = strconv.FormatFloat(float64(n)/255, 'f', 3, 64)
		}
	}
	return vars
}

// slotForBase maps CDI asset names (already stripped of the -<zoom> suffix)
// onto our image slots, with a priority so specific assets beat generic ones.
// CDI asset families:
//   diskGood*        top disk-button art
//   diskStatusGood*  health block art
//   SDdiskStatusGood* life block art
//   temperatureGood* temperature block art
func slotForBase(base string) (string, int) {
	b := strings.ToLower(base)
	repl := strings.NewReplacer("-", "", "_", "", " ", "", "~", "")
	b = repl.Replace(b)
	class := classOf(b)
	mini := strings.Contains(b, "mini")

	switch {
	case strings.Contains(b, "background"):
		return "background", 100
	case strings.Contains(b, "copyright"):
		return "", 0
	case b == "logo" || strings.HasSuffix(b, "logo"):
		return "logo", 100
	case strings.Contains(b, "predisk"):
		return "pre", 100
	case strings.Contains(b, "nextdisk"):
		return "next", 100
	case strings.Contains(b, "nodisk"):
		return "nodisk", 100
	case strings.Contains(b, "temperature") || strings.HasPrefix(b, "temp"):
		if class != "" {
			return "temp_" + class, 100
		}
		return "", 0
	case strings.Contains(b, "sddiskstatus"):
		if class == "" {
			return "", 0
		}
		return "sd_" + class, 100
	case strings.Contains(b, "diskstatus"):
		if class == "" {
			return "", 0
		}
		slot := "status_" + class
		if mini {
			slot += "_mini"
		}
		return slot, 100
	case strings.HasPrefix(b, "disk"):
		if class == "" {
			return "", 0
		}
		slot := "disk_" + class
		if mini {
			slot += "_mini"
		}
		return slot, 100
	case class != "":
		return "status_" + class, 50 // generic good.png etc.
	}
	return "", 0
}

func classOf(b string) string {
	switch {
	case strings.Contains(b, "goodgreen"):
		return "good"
	case strings.Contains(b, "good"):
		return "good"
	case strings.Contains(b, "caution"):
		return "caution"
	case strings.Contains(b, "bad"):
		return "bad"
	case strings.Contains(b, "unknown"):
		return "unknown"
	}
	return ""
}

// splitZoom splits "diskGood-100.png" into ("diskGood", 100).
func splitZoom(name string) (string, int) {
	base := strings.TrimSuffix(name, filepath.Ext(name))
	if i := strings.LastIndexByte(base, '-'); i > 0 {
		if n, err := strconv.Atoi(base[i+1:]); err == nil && n >= 10 && n <= 1000 {
			return base[:i], n
		}
	}
	return base, 100
}

func parseColor(v string) string {
	v = strings.TrimSpace(v)
	if i := strings.IndexByte(v, ';'); i >= 0 {
		v = strings.TrimSpace(v[:i])
	}
	v = strings.Trim(v, `"'`)
	if v == "" {
		return ""
	}
	base := 10
	if strings.HasPrefix(strings.ToLower(v), "0x") {
		base = 16
		v = v[2:]
	} else if strings.HasPrefix(v, "#") {
		base = 16
		v = v[1:]
	}
	n, err := strconv.ParseUint(v, base, 32)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("#%06X", n&0xFFFFFF)
}

func trimUnusedFiles(dir string, refs map[string]bool) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	keep := map[string]bool{"theme.json": true, "theme.ini": true, "style.css": true}
	for name := range refs {
		keep[name] = true
	}
	for _, e := range entries {
		if e.IsDir() || keep[e.Name()] {
			continue
		}
		if themeImageExt[strings.ToLower(filepath.Ext(e.Name()))] {
			_ = os.Remove(filepath.Join(dir, e.Name()))
		}
	}
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// frameCountForSlot mirrors CDI's control image counts (ButtonFx/CommonFx.h):
// disk buttons carry 4 frames (normal/hover/focus/selected), pager buttons 2.
func frameCountForSlot(slot string) int {
	if strings.HasPrefix(slot, "disk_") {
		return 4
	}
	if slot == "pre" || slot == "next" {
		return 2
	}
	return 1
}

// splitPNGFrames splits a vertical sprite strip into <slot>.<n>.png files and
// returns their names, or nil when the asset is not a splittable PNG strip.
func splitPNGFrames(srcPath, dstDir, slot string, count int) []string {
	if count < 2 || !strings.EqualFold(filepath.Ext(srcPath), ".png") {
		return nil
	}
	f, err := os.Open(srcPath)
	if err != nil {
		return nil
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		return nil
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h%count != 0 {
		return nil
	}
	fh := h / count
	names := make([]string, 0, count)
	for n := 0; n < count; n++ {
		sub := image.NewNRGBA(image.Rect(0, 0, w, fh))
		draw.Draw(sub, sub.Bounds(), img, image.Pt(b.Min.X, b.Min.Y+n*fh), draw.Src)
		name := fmt.Sprintf("%s.%d.png", slot, n)
		out, err := os.Create(filepath.Join(dstDir, name))
		if err != nil {
			return nil
		}
		err = png.Encode(out, sub)
		out.Close()
		if err != nil {
			return nil
		}
		names = append(names, name)
	}
	return names
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

// ---------- INI parsing (Windows INI, ANSI or UTF-16) ----------

type iniFile map[string]map[string]string

func (f iniFile) get(section, key string) string {
	if m, ok := f[strings.ToLower(section)]; ok {
		return m[strings.ToLower(key)]
	}
	return ""
}

func parseINI(raw []byte) iniFile {
	text := decodeINI(raw)
	out := iniFile{}
	section := ""
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") {
			if i := strings.IndexByte(line, ']'); i > 1 {
				section = strings.ToLower(strings.TrimSpace(line[1:i]))
				if out[section] == nil {
					out[section] = map[string]string{}
				}
			}
			continue
		}
		i := strings.IndexByte(line, '=')
		if i <= 0 || section == "" {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(line[:i]))
		val := strings.TrimSpace(line[i+1:])
		out[section][key] = val
	}
	return out
}

// decodeINI handles UTF-16 LE/BE BOM files and falls back to raw bytes
// (ANSI/UTF-8; invalid bytes are kept as-is for later numeric parsing).
func decodeINI(raw []byte) string {
	if len(raw) >= 2 {
		if raw[0] == 0xFF && raw[1] == 0xFE {
			return string(utf16.Decode(bytesToUint16(raw[2:], true)))
		}
		if raw[0] == 0xFE && raw[1] == 0xFF {
			return string(utf16.Decode(bytesToUint16(raw[2:], false)))
		}
	}
	return string(raw)
}

func bytesToUint16(b []byte, little bool) []uint16 {
	out := make([]uint16, 0, len(b)/2)
	for i := 0; i+1 < len(b); i += 2 {
		if little {
			out = append(out, uint16(b[i])|uint16(b[i+1])<<8)
		} else {
			out = append(out, uint16(b[i])<<8|uint16(b[i+1]))
		}
	}
	return out
}
