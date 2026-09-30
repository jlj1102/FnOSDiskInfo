# AGENTS.md — CrystalDiskInfo-like app for fnOS

> Target agent: low-cost coding model (e.g. DeepSeek V4.1 Flash).
> Goal: give the agent enough local facts + exact references so it does not need broad research every task.
> Verified/reference set assembled: 2026-09-27.

---

# 0.1 Verified environment facts & decisions (2026-09-27, on-device probe)

- `smartctl 7.3` at `/usr/sbin/smartctl`, no setuid, no file capabilities. Unprivileged users get
  `Permission denied` (device node perms + missing `CAP_SYS_RAWIO`).
- `trim-cli` does NOT exist in `/usr/trim/bin` on stock fnOS. Do not plan around it.
- Lifecycle scripts run as the package user (observed uid 989) under `run-as: package`;
  `setcap` fails with `unable to set CAP_SETFCAP`. File capabilities are NOT an option.
- `join-groups: ["disk"]` works: the app user can open `/dev/sd*`, but SG_IO still fails
  without `CAP_SYS_RAWIO`, so only identity/capacity is readable, never SMART.
- Official docs field is `join-groups` (the community mirror's `extra-groups` is outdated).

Decisions that supersede conflicting text below:

- Backend: Go 1.26, stdlib only, one static linux/amd64 binary with `go:embed` frontend.
- Privilege: `run-as: root` (documented root mode). `cmd/main` starts a root `collect`
  subprocess that runs smartctl and writes `$TRIM_PKGVAR/cache.json` atomically, then starts
  the `serve` subprocess via `runuser -u $TRIM_USERNAME`. Web never touches disks or smartctl.
- Rescan: `serve` touches `$TRIM_PKGVAR/rescan`; collector checks/removes it every second.
- Root mode caveat: the package manager owns `$TRIM_PKGVAR` as root after install/upgrade,
  while the web process runs as the package user. `cmd/main` (root) must run
  `ensure_var_perms` (`chown -R` + `setfacl -R`) before starting the processes, or
  requests/settings/theme imports fail with `open ...: permission denied`.
- Entry: `iframe` on plain port 7817 for MVP; unified gateway deferred.
- Repo layout: fnpack package root is `cdifnos/`; Go source is `src/`; build with `build.ps1`.
  fnpack CLI lives in `tools/fnpack.exe` (Windows build, gitignored if repo is initialized).

---

# 0.2 Implemented feature map (v0.2, 2026-09-27)

- Health engine: good/caution/bad/unknown, configurable caution thresholds
  (05/C5/C6 default 1, FF default 10; `settings.json`), NVMe critical-warning /
  available-spare rules, status reasons, Life % (NVMe percentage_used; ATA SSD
  wear-attribute heuristic in health.go).
- History: `$TRIM_PKGVAR/history/<id>.jsonl`, written only on change (60s throttle),
  pruned at 4 MB / 10k points on collector start. API:
  `GET /api/disks/{id}/history?metric=&points=`.
- Alarms: `$TRIM_PKGVAR/alarms.jsonl` on health / temperature class transitions;
  `GET /api/alarms`; frontend banner + dialog.
- Controls (privileged): web writes `$TRIM_PKGVAR/requests/<id>.json`; root collector
  executes self-test / abort / AAM / APM with a whitelist + value regex, resolves the
  disk from its own snapshot, writes `<id>.result`. Frontend polls `GET /api/requests/{id}`.
- Settings: `$TRIM_PKGVAR/settings.json` owned/written by the app user; collector reloads
  on mtime change. Display prefs (unit, raw format, hide serial, theme, language) are client-side.
- Reports: `GET /api/disks/{id}/report.txt` (CDI-style text), `/raw` (raw smartctl JSON).
- UI v0.3: CDI-style menubar (File/Edit/Function/Theme/Disk/Help, functional items only),
  paged 80px disk buttons, left Health/Temp/Life blocks, two-column info grid, bottom SMART table.
- Themes: built-ins `classic`/`dark`/`follow` embedded in `web/themes/<id>/theme.json`;
  imported packs in `$TRIM_PKGVAR/themes/<id>/` (POST /api/themes/import, zip validated:
  no traversal/symlinks/executables, extension whitelist, 20 MB cap). CDI-style packs without
  theme.json get a best-effort manifest from asset names (themes.go). Served via `/themes/`.
- Display prefs (client-side): zoom, font family/size, temperature unit, raw value format,
  hide serial, hide SMART table, hide no-SMART disks, disk sort.
- v0.4: menu checked-state evaluated at paint time (`checked: () => ...`); CDI theme packs  decode `theme.ini` ([Color]/[Alpha]/ParentTheme) plus `<asset>-<zoom>.png` assets
  (themes.go, one zoom per asset kept, multi-theme zips supported, re-importing the same
  theme id overwrites the previous pack); asset slots are family-correct and deterministic:
  `disk_*` (top buttons) / `status_*` (health) / `sd_*` (life) / `temp_*` / `background` /
  `logo` / `pre|next|nodisk`; CDI sprite strips are split into per-frame PNGs at import
  (disk buttons 4 frames normal/hover/focus/selected, pre/next 2) and the UI picks frames
  like CDI; AAM/APM values are
  queried on demand (`aam-apm-get`) instead of inferred, raw smartctl output is collapsed;
  `exclude_disks` removes disks from auto refresh (collector marks them `stale`);
  File > Save Image renders the current view to PNG via snapshot.js. Character themes draw
  the backdrop CDI-style (the `-300` asset scaled to the 1000px design width, tiled both
  axes from the client-area top-left, i.e. `#mainArea`) with blur + alpha floor 0.65
  for readability.
- Local E2E harness: `go -C src build -o ..\fake\smartctl.exe ./testdata/fake-smartctl`
  and prepend `fake\` (repo root) to PATH when running the dev binary.

- v0.5: CDI Shizuku geometry layout (character column 328px + content column 672px, disk
  buttons 84x48 with 3-line text, status/temp/life art with overlaid text) activates only
  when the theme ships art; Graph/Option windows are ports of CDI's Graph.html/Option.html
  using vendored jQuery+flot (MIT, `web/flot/`), fed by the history API; history points now
  carry every attribute's current value and `metric=attr-XX` plots any of them.
- v0.5.1: geometry panel bug — `#geo` carried `hidden` and `.hidden{display:none!important}`
  beat `body.cdi-geometry .geo{display:block}` so themed mode showed only the SMART table;
  the class is gone now. Backdrop is one image, `cover`-fitted (width first, height-fit
  fallback so no empty strip appears), `no-repeat`, left top, on `#mainArea`
  (client-area origin, below the menubar). The character column flexes
  (`1 1 328px`) so the 672px content column hugs the right edge when the window grows.
  Geometry disk buttons must use full-size `disk_*` art (84x48) — `tabIconSrc` prefers the
  42x48 `_mini` strip, which was stretched to 84px and blurred (now `geoIconSrc`).
  Health block text is CDI Shizuku style: `status (life %)` (`DiskInfoDlgUpdate.cpp`); the
  SD life art carries no text overlay, and life==100 with good health swaps in the
  `sd_good100` slot (`SDdiskStatusGood100` — a distinct slot now, not a clash).
- v0.5.2: ATA identity extras parsed from the same `-x -j` output (real samples in
  `%TEMP%\opencode\sda.json`/`sdb.json`, keys verified against smartmontools source):
  top-level `rotation_rate` (0 = SSD; reading `device.rotation_rate` was wrong and showed
  HDDs as "SSD"); `interface_speed.current/max` → CDI-style `SATA/600 | SATA/600` (from
  `units_per_second` 60/30/15, raw string fallback); `ata_version.major_value` bitmask
  (highest bit) + `sata_version.string` → `ACS-3 | SATA 3.1`; features = `smart_support` /
  `ata_aam` / `ata_apm` / `trim.supported` / `gp_logging_supported`, NCQ inferred from the
  NCQ Command Error log (GP log 0x10) — DevSleep/Streaming/buffer_size/nv_cache have no
  smartctl JSON source, NVMe (`nvme_version`) not wired (no sample).
- v0.5.3: theme image fallback mirrors CDI's `IP()` chain (DialogFx.cpp): active theme ->
  `ParentTheme1/2` from theme.ini (case-insensitive dir lookup) -> fallback theme chosen in
  the Theme menu (`localStorage.cdifnos.fallback`: `none`/`auto`/<id>; auto = imported theme
  with the most image slots). `GET /themes/<id>/theme.json?fallback=...` merges server-side;
  merged slots carry absolute `/themes/<provider>/<file>` URLs, plus `frame_count` and
  `frame_dir` for sprite frames (theme.js resolves those). Menubar is host chrome with its
  own palette (`--cdi-navbar-bg/-text/-border`): CDI packs never tint it (see v0.5.4).
- v0.5.4: the menubar follows the fnOS host theme via the official JS SDK. `manifest` has
  `micro_app = true` (required for the SDK); `@trimjs/web-app` 0.4.2 ESM is vendored at
  `web/trim/index.js` and `web/platform.js` calls `getPlatformConfig()` then
  `$on('os/theme')` for live switches, setting `data-navbar="dark|light"`. Standalone
  browsers / older fnOS keep the `prefers-color-scheme` fallback
  (`:root:not([data-navbar])`); the CSS navbar palette is no longer keyed on `data-theme`.
  The host does not push `os/theme` to third-party pages, so `platform.js` also polls
  `getPlatformConfig()` every 3s (no overlap, hidden tabs skipped) — the menubar follows
  without a reload.
- v0.5.5: geometry info block used CDI client-area coordinates inside `.geo-info`, which
  already sits below the 48px button row + title bar, so every row/art was 92px too low
  (big gap under the model title, dead space above the SMART table). Offsets are now
  relative to the info block: life/columns `top: 0`, status/temp art `168`, labels `172`,
  title bar 40px, info height 200px — the geo block ends at CDI's y=288 list start.
- v0.5.6: SMART table follows CDI's LED model (DiskInfoDlgUpdate.cpp UpdateListCtrl):
  a leading 16px LED column (`good|caution|bad|unknown`), zebra stripes from the theme
  ListBk pair, no status text column and no red row tint. Status is computed in
  `health.go attributeStatus` (temperature always good; 05/C5/C6 below ATA threshold ->
  bad, raw above the configured caution limit -> caution; CDI's ATA ID ranges only; else
  good) — `when_failed`/"In_the_past" is deliberately ignored, so e.g. attr BE (Airflow
  Temp) no longer shows as damaged. LED art: theme slot `led_<status>` (fallback chain
  applies) or the bundled icons `web/icons/led_*.png`, converted from CDI's MIT-licensed
  `res/good|caution|bad|unknown.ico`. Save Image (`snapshot.js`) now paints the
  `#mainArea` backdrop (cover-fitted) first and draws images before text, so overlay text
  (disk buttons/health/temp) is not dimmed by the art.
- v0.5.7: SMART attribute names come from CDI's language packs — `web/attr-i18n.js` is
  generated from `Language/*.lang` (`[Smart]` 59 + `[SmartSsd]` 21, en/zh-CN/zh-TW/ja, MIT),
  keys are two-digit hex IDs; `attrName(id, fallback, isSsd)` prefers the SSD table for
  SSDs and falls back to smartctl's English name. The snapshot painter now clips everything
  inside `#attrwrap` to its scroll viewport (boxes, LED images, text), so rows scrolled out
  of the SMART list no longer bleed over the panels above in the exported PNG.
- v0.6: the built-in "CDI Classic" theme is renamed to `Light` (id stays `classic`);
  `LICENSE` (MIT) added at the repo root; `.github/workflows/nightly.yml` builds
  `cdifnos.fpk` on windows-latest (fnpack is Windows-only and stays out of the repo —
  the workflow downloads the official
  `https://static2.fnnas.com/fnpack/fnpack-1.2.3-windows-amd64` to `tools/fnpack.exe`)
  and publishes/updates the `nightly` pre-release with the artifact.
- v0.7: SSD/NVMe display overhaul (CDI fidelity): `ssd.go` ports CheckSsdSupport + all 36
  `IsSsd*` predicates (model/firmware substrings + attribute-ID signatures) to pick a
  `smart_key` and applies the Life/Host Reads/Host Writes/NAND Writes rules (e.g. Kingston
  SA400 E7 raw 93 -> life 93); NVMe gets CDI's 15 pseudo attributes (SmartNVMe names) with
  per-row LED rules, `NVM Express x.y` standard, `NVM Express` interface and
  S.M.A.R.T./TRIM features; `attr-i18n.js` regenerated with all 51 `[Smart*]` sections x 4
  languages (hex keys); `attrName(id, fallback, smartKey, isSsd, isNvme)` walks
  matched section -> SmartSsd/SmartNVMe -> Smart -> smartctl name (Aa_Bb -> Aa Bb); the
  info panel reuses the Buffer/NV Cache/Rotation rows as Total Host Reads/Writes/NAND
  Writes for SSDs (CDI ChangeDisk) plus title tooltips; `fake-smartctl` serves an HDD
  (sda), NVMe (sdb) and SSD (sdc), `FAKE_DEV=nvme|ssd` exposes a single device.
  Local SSD/NVMe samples live outside the repo (`*.json` at repo root is gitignored).
- v0.7.1: SMART table column layouts now follow CDI's RebuildListHeader per vendor:
  NVMe and SmartIndilinx hide Current/Worst/Threshold, SmartJMicron60x hides
  Worst/Threshold. Raw values use CDI's formats (DiskInfoDlgUpdate.cpp): hex =
  zero-padded uppercase bytes (ATA 12 digits, NVMe 14 with the reserved byte), dec =
  48-bit value, byte2/byte1 = big-endian words/bytes. The feature tooltip lists only the
  features the disk reports (`feat_*` i18n strings) instead of CDI's full list. NVMe
  transfer mode comes from `/sys/class/nvme/<ctrl>/device/{current,max}_link_{speed,width}`
  formatted like CDI's `SlotSpeedToString` (`PCIe 3.0 x4 | PCIe 3.0 x4`, empty when the
  sysfs link info is unavailable, e.g. USB bridges). The SanDisk/SF/Indilinx 16-digit raw
  variants are not ported yet.
  ponytail: those vendor raw-byte variants can be added if a user reports them.
- v0.7.2: NVMe TRIM and VolatileWriteCache come from the Identify Controller read with
  a stdlib ioctl (`nvme_linux.go`: NVME_IOCTL_ADMIN_CMD 0xC0484E41, opcode 0x06 CNS 0x01;
  byte 520 bit 2 = Dataset Management, byte 525 bit 0 = VWC — same source as CDI,
  AtaSmart.cpp:4001-4009), because smartctl 7.3's JSON omits
  `nvme_optional_nvm_commands`; the JSON flags stay the fallback and the Windows dev
  build uses the `nvme_other.go` stub. Feature row order: S.M.A.R.T., TRIM,
  VolatileWriteCache. Geometry Standard/Features value cells span the full 408px
  (CDI InitControl width) instead of the 180px value column, so full text shows.
- Release rule: whenever the feature line bumps (v0.x), also bump `cdifnos/manifest`
  `version` and `src/main.go` `const version` so fnOS's app center detects an **upgrade**
  instead of a reinstall; the Nightly workflow additionally stamps
  `<version>-nightly.<run_number>` on its own builds.

---

## 0. Mission

Build a **native fnOS FPK application** that gives a **CrystalDiskInfo-like experience** on a NAS.

This is **not** a Windows binary port.

Use:

- fnOS Native FPK packaging
- Web frontend inside fnOS
- Local backend service
- `smartctl` JSON + Linux disk metadata
- CDI-like dense desktop UI
- Theme system with Classic / Dark / Follow fnOS / optional CDI-theme adapter

Do NOT:

- run `DiskInfo.exe` with Wine
- convert the UI into a generic card dashboard
- run `smartctl` on every browser request
- identify disks only by `/dev/sdX`
- allow frontend-controlled shell commands
- assume Windows drive letters, tray, registry, WMI, or Win32 IOCTL exist

---

# 1. Source priority

When references conflict, use this order:

1. **Current project code**
2. **Official fnOS developer documentation**
3. **Official CrystalDiskInfo source / official website**
4. **smartmontools upstream**
5. **fnOS community mirrors/examples**
6. Generic Linux knowledge

Do not invent fnOS APIs.

If an official fnOS page and a community mirror disagree, the official page wins.

---

# 2. Required fnOS references

## 2.1 Official fnOS developer portal — PRIMARY

URL:

https://developer.fnnas.com/docs/

Use it for:

- FPK project structure
- `manifest`
- `config/privilege`
- `config/resource`
- `wizard/*`
- `cmd/main`
- desktop app entries
- `fnpack`
- official Open API documentation

Important known facts:

- `fnpack create <appname>` creates a Native app skeleton.
- `fnpack build` creates the `.fpk`.
- `manifest` defines app identity/runtime metadata.
- `cmd/main` is used for start/stop/status.
- `config/privilege` controls application privileges.
- `config/resource` declares resources/capabilities.
- desktop entry comes from the UI config referenced by `desktop_uidir`.
- fnOS can launch an app entry as `iframe` or `url`.

Do not guess fields that are not documented.

---

## 2.2 Search-friendly fnOS developer guide mirror — SECONDARY

URL:

https://github.com/conversun/fnos-apps/blob/main/docs/fnos-developer-guide.md

Why it exists in this AGENTS.md:

The official developer portal may be SPA-heavy or expensive for a small model to traverse.  
This mirror is a compact, searchable copy/summary of the official development documentation.

Use it for quick lookup of:

- app directory layout
- `manifest`
- TRIM environment variables
- privilege/resource config
- desktop entries
- `iframe` / `url` entry types
- wizard files
- Native packaging
- `fnpack`
- icon rules

Relevant sections:

- **二、架构概述**
- **三、Manifest 配置**
- **四、环境变量**
- **五、应用权限**
- **六、应用资源**
- **七、应用入口**
- **十三、Native 应用构建**
- **十四、图标规范**
- **十五、CLI 开发工具**

Key entry behavior:

```json
{
  ".url": {
    "app.Application": {
      "title": "App",
      "icon": "images/icon-{0}.png",
      "type": "iframe",
      "protocol": "http",
      "port": "8080",
      "url": "/",
      "allUsers": true
    }
  }
}
```

Known entry fields include:

- `type: "iframe"` → load inside an fnOS desktop window
- `type: "url"` → open as browser URL
- `protocol`
- `port`
- `url`
- `allUsers`

Prefer `iframe` for the CDI-like desktop experience unless the real project requires otherwise.

---

## 2.3 fnOS example FPK/API project — IMPLEMENTATION REFERENCE

URL:

https://github.com/heijiangangqin/FNOS-API

Use only as an example for:

- FPK layout
- backend + frontend + UI entry organization
- lifecycle scripts
- packaging shape

Do NOT treat undocumented API behavior in this repo as official fnOS behavior.

---

## 2.4 fnOS official icon/template asset

URL:

https://static.fnnas.com/appcenter-marketing/fnpack_ICON_256.zip

Use when implementing fnOS-conformant application icons.

Known official icon sizes:

- `ICON.PNG` — 64×64
- `ICON_256.PNG` — 256×256

---

# 3. Required CrystalDiskInfo references

## 3.1 Official source repository — PRIMARY CDI SOURCE

URL:

https://github.com/hiyohiyo/CrystalDiskInfo

Use it to understand:

- application behavior
- disk selector behavior
- health display
- SMART table structure
- menus
- tray-specific behavior that must NOT be copied directly
- theme/language architecture
- device support concepts

Important:

- source code is MIT licensed
- the repository does **not** include all `CdiResource` resource files
- do not assume Windows disk-access code is portable to Linux

---

## 3.2 Official product/manual page

URL:

https://www.crystalmarksoftware.com/en/software/crystaldiskinfo/

Use for:

- current user-facing product structure
- Standard / Shizuku / Kurei Kei visual references
- menu/manual categories
- expected CDI information hierarchy

The main UI identity to preserve:

- compact top drive selector
- drive model/capacity title
- health block
- temperature block
- device metadata
- dense SMART table

---

## 3.3 Windows disk backend reference — DO NOT PORT DIRECTLY

URL:

https://github.com/hiyohiyo/CrystalDiskInfo/blob/master/AtaSmart.h

Also:

https://github.com/hiyohiyo/CrystalDiskInfo/blob/master/AtaSmart.cpp

Important content:

- CDI uses Windows-specific mechanisms.
- `AtaSmart.h` includes `winioctl.h`.
- implementation also uses Windows/COM/WMI paths.
- CDI has multiple command/device modes for ATA, NVMe, USB bridges, RAID, MegaRAID, etc.

Use these files to understand **behavior and supported hardware concepts**, not as a Linux backend implementation.

Linux backend should use smartmontools/sysfs instead.

---

## 3.4 CDI theme/application reference

URL:

https://github.com/hiyohiyo/CrystalDiskInfo/blob/master/DiskInfo.h

Also useful:

https://github.com/hiyohiyo/CrystalDiskInfo/blob/master/DiskInfo.cpp

https://github.com/hiyohiyo/CrystalDiskInfo/blob/master/DiskInfoDlgInit.cpp

Important content:

- CDI has explicit theme/language state.
- `DiskInfo.h` defines theme/language related paths/state.
- CDI resources include dialogs/themes/languages outside the core code repository.
- special editions have separate character/voice/resource concepts.

Use this only to preserve the theme concept.

Do NOT reproduce Windows-only runtime behavior.

---

## 3.5 CDI tray behavior reference

URL:

https://github.com/hiyohiyo/CrystalDiskInfo/blob/master/DiskInfoDlgTray.cpp

Use to understand what the Windows version does.

Do NOT directly clone the tray implementation.

fnOS replacement should be:

- backend monitoring
- fnOS notification integration when documented/available
- in-app alerts/history as fallback

---

# 4. smartmontools references

## 4.1 Upstream repository

URL:

https://github.com/smartmontools/smartmontools

This is the preferred Linux disk-health backend.

---

## 4.2 smartctl manual/source

URL:

https://github.com/smartmontools/smartmontools/blob/main/src/smartctl.8.in

Important commands:

```bash
smartctl --scan-open
smartctl -x -j /dev/sda
smartctl -x -j /dev/nvme0
```

Important:

- `-j` / `--json` enables JSON output.
- prefer JSON over parsing human-readable terminal output.
- device-specific `-d TYPE` may be necessary for some bridges/controllers.

Possible types to support when needed:

```text
auto
sat
scsi
megaraid,N
```

Do NOT hardcode a single disk/controller model.

---

# 5. Visual references — MUST inspect for frontend/theme work

If the model/runtime supports image input or URL vision, inspect these images before changing major frontend layout/theme behavior.

These are **reference images**, not assets to automatically redistribute.

## 5.1 fnOS desktop visual reference

Official fnOS image:

https://static2.fnnas.com/official/download/os.webp

Markdown preview:

![fnOS desktop reference](https://static2.fnnas.com/official/download/os.webp)

Look for:

- browser-based desktop metaphor
- floating app windows
- desktop icon scale
- visual density
- fnOS background/context around an app iframe

Frontend must feel usable inside an fnOS desktop window, not like a standalone fullscreen website.

---

## 5.2 CrystalDiskInfo Standard visual reference

Official screenshot:

https://www.crystalmarksoftware.com/wp-content/uploads/sites/3/crystaldiskinfo-ssd-health-monitor-277e23.png

Markdown preview:

![CrystalDiskInfo Standard](https://www.crystalmarksoftware.com/wp-content/uploads/sites/3/crystaldiskinfo-ssd-health-monitor-277e23.png)

Observe:

- top row disk state selectors
- menu bar
- centered drive model/capacity
- health + temperature blocks at left
- metadata in compact rows
- SMART table using most of the lower window
- high information density
- desktop utility appearance

Do not replace this information hierarchy with large cards.

---

## 5.3 CrystalDiskInfo Shizuku visual reference

Official screenshot:

https://www.crystalmarksoftware.com/wp-content/uploads/sites/3/crystaldiskinfo-ssd-health-monitor-shizuku-edition-b80263.jpg

Markdown preview:

![CrystalDiskInfo Shizuku](https://www.crystalmarksoftware.com/wp-content/uploads/sites/3/crystaldiskinfo-ssd-health-monitor-shizuku-edition-b80263.jpg)

Observe:

- character/background art can coexist with the same functional information
- readability still matters
- theme engine therefore needs optional image layers

Do NOT copy bundled character artwork into this project unless its redistribution/license has been verified separately.

---

## 5.4 CrystalDiskInfo Kurei Kei visual reference

Official screenshot:

https://www.crystalmarksoftware.com/wp-content/uploads/sites/3/crystaldiskinfo-ssd-health-monitor-kurei-kei-edition-f94a62.jpg

Markdown preview:

![CrystalDiskInfo Kurei Kei](https://www.crystalmarksoftware.com/wp-content/uploads/sites/3/crystaldiskinfo-ssd-health-monitor-kurei-kei-edition-f94a62.jpg)

Use this to confirm that themed editions may significantly alter imagery while keeping the same core information model.

---

# 6. Product architecture

```text
fnOS Desktop
    ↓
FPK app entry (prefer iframe)
    ↓
Web Frontend
    ↓
Local HTTP API
    ↓
Backend Service
    ↓
cache
    ↓
smartctl / sysfs / lsblk / udev
    ↓
Physical disks
```

Frontend MUST NOT access `/dev/*` directly.

Backend MUST validate all disk identifiers.

---

# 7. Suggested FPK structure

Do not force this exact tree if the current repo already has a working fnOS layout.

```text
package/
├── manifest
├── ICON.PNG
├── ICON_256.PNG
├── app/
│   ├── backend/
│   ├── frontend/
│   ├── ui/
│   └── bin/
│       └── smartctl
├── config/
│   ├── privilege
│   └── resource
├── cmd/
│   └── main
└── wizard/
```

Use documented fnOS paths/environment variables instead of hardcoded installation paths.

---

# 8. UI contract

Main UI should remain close to CDI:

```text
┌───────────────────────────────────────────────────────┐
│ GOOD 31° │ GOOD 29° │ CAUTION 44° │ GOOD 38°       │
├───────────────────────────────────────────────────────┤
│ ST3000DM008-2DM166 : 3000.5 GB                       │
│                                                       │
│ Health        Firmware      CC26                      │
│ [ GOOD ]      Serial        XXXXXXXX                  │
│               Interface     SATA                      │
│ [ 31°C ]      Power On      18234 Hours               │
├───────────────────────────────────────────────────────┤
│ ID │ Attribute            │ Cur │ Worst │ Thr │ Raw   │
│ 05 │ Reallocated...       │ 100 │ 100   │ 10  │ 0     │
│ 09 │ Power-On Hours       │  80 │  80   │  0  │ ...   │
└───────────────────────────────────────────────────────┘
```

MUST keep:

- top disk selector
- health
- temperature
- device info
- dense SMART table

MUST NOT convert SMART attributes into large cards.

---

# 9. Window/responsive behavior

fnOS app may run in a resizable iframe/window.

Preferred behavior:

```text
>= 900 px    full CDI-like layout
720–899 px   compact layout
< 720 px     preserve desktop layout + horizontal scroll
```

Do not destroy the table to become "mobile friendly".

High information density is a feature.

---

# 10. Disk identity

Never use `/dev/sdX` as the only identity.

Prefer stable backend IDs based on:

- WWN
- disk serial
- NVMe serial
- `/dev/disk/by-id/*`

Display may show:

```text
HDD 1
ST3000DM008
3 TB
Storage Pool 1
/dev/sda   ← optional advanced/debug text
```

If fnOS exposes documented pool/volume mapping APIs, isolate them behind an adapter.

Do not invent those APIs.

---

# 11. Backend/cache rules

Bad:

```text
Browser request
→ smartctl
→ response
```

Good:

```text
periodic disk collector
→ normalized backend cache
→ API
→ all frontend sessions
```

Suggested collection interval:

```text
5–30 seconds
```

Frontend may request cached data every:

```text
2–5 seconds
```

or use SSE/WebSocket.

One backend cache must serve all users/windows.

---

# 12. Suggested API

```text
GET  /api/disks
GET  /api/disks/:id
GET  /api/disks/:id/smart
GET  /api/disks/:id/history

POST /api/disks/rescan
POST /api/disks/:id/self-test
POST /api/disks/:id/apm

GET  /api/settings
PUT  /api/settings
```

Rules:

- stable disk ID in routes
- map stable ID → current device path server-side
- reject unknown IDs
- no arbitrary shell parameters
- no `/api/shell`

---

# 13. Theme contract

Required modes:

```text
Follow fnOS
Light
Dark
Custom
optional CDI-theme import/adapter
```

Theme controls:

- color
- font
- border
- background
- image layers
- status badge appearance

Theme MUST NOT control:

- backend logic
- API
- SMART parsing
- permissions
- component behavior

Use CSS variables.

Example:

```css
:root {
  --cdi-bg: #fff;
  --cdi-panel: #fff;
  --cdi-text: #111;
  --cdi-border: #aaa;
  --cdi-good: #4da3ff;
  --cdi-caution: #ffd75a;
  --cdi-bad: #e05050;
  --cdi-unknown: #888;
}
```

Never use `filter: invert(1)` for dark mode.

Health colors are semantic.

---

# 14. Theme image model

Support optional layers:

```text
background
character
overlay
logo
status assets
```

Suggested custom theme structure:

```text
themes/example/
├── theme.json
├── style.css
├── background.webp
├── character.webp
├── good.svg
├── caution.svg
└── bad.svg
```

Imported theme ZIP:

```text
validate
→ reject path traversal/executables
→ extract
→ parse
→ preview
→ install
```

Never execute JavaScript/binaries from a theme.

---

# 15. CDI theme compatibility rule

Compatibility is best-effort.

Target levels:

```text
L1: colors/fonts/basic backgrounds
L2: background/character/logo/status images
L3: legacy Windows-specific sizing/positioning = partial/unsupported
```

Do NOT promise pixel-perfect Windows theme compatibility.

---

# 16. fnOS-specific UX substitutions

Windows CDI concept → fnOS behavior:

```text
Windows native window
→ fnOS iframe app window

Windows tray temperature icon
→ fnOS notification if documented + in-app alert/history

drive letters
→ NAS disk identity / pool / volume mapping

Win32 synchronous actions
→ async backend API actions

Windows startup
→ fnOS app lifecycle/autostart behavior only if documented
```

Do not expose fake Windows-only menu items.

---

# 17. Permissions/security

Disk access may require elevated backend privilege.

Keep privileged code minimal.

```text
Frontend
→ validated API
→ backend
→ device
```

MUST:

- validate disk IDs
- validate request bodies
- restrict command arguments
- prevent shell injection
- prevent ZIP path traversal
- sanitize text rendered in UI
- never let a theme execute code
- never let frontend choose arbitrary `/dev/*` paths

---

# 18. Multi-user

fnOS can have multiple users.

Recommended split:

Read-only:

- disk list
- SMART
- health
- temperature
- history

Privileged:

- self-test
- APM/control operations
- backend settings
- destructive or device-changing operations

Use actual documented fnOS authorization information when available.

Do not invent user-role APIs.

---

# 19. i18n

UI strings must be centralized.

Initial language targets:

```text
en
zh-CN
zh-TW
ja
```

SMART attribute translation may be separate from general UI translation.

Do not hardcode user-visible strings across components.

---

# 20. MVP order

Implement in this order unless the existing repo already passed a phase:

```text
1. FPK skeleton installs/starts
2. frontend opens in fnOS
3. backend disk scan
4. smartctl JSON normalize
5. CDI-like disk selector
6. device info + health + temperature
7. SMART table
8. backend cache
9. Classic/Dark/Follow fnOS theme
10. SAS/HBA/USB special handling
11. self-test/history/alerts
12. optional CDI theme importer
```

Do not build history/theme-import before basic disk correctness works.

---

# 21. Agent rules

For every implementation task:

1. Read existing relevant files first.
2. Preserve working behavior.
3. Make the smallest useful change.
4. Do not rewrite unrelated modules.
5. Avoid new dependencies unless clearly necessary.
6. Use official fnOS docs before guessing fnOS behavior.
7. Use CDI references for UX/behavior, not Windows backend code.
8. Use smartmontools for Linux disk-health collection.
9. Keep disk access behind backend APIs.
10. Keep the frontend dense and CDI-like.
11. If changing layout/theme and vision is available, inspect the visual references in section 5.
12. If a required API is undocumented, create an adapter/interface and use a safe fallback instead of inventing it.

---

# 22. Fast reference map for cheap models

When task mentions...

```text
FPK / manifest / fnpack / privilege
→ Section 2

iframe / fnOS desktop entry
→ 2.1 + 2.2

CDI layout / menu / behavior
→ 3.1 + 3.2 + Section 5

Windows CDI backend details
→ 3.3
→ understand only, DO NOT port

theme / Shizuku / character UI
→ 3.4 + 5.2–5.4 + 13–15

SMART / SATA / NVMe / SAS / HBA
→ Section 4 + 10–12

tray / notifications
→ 3.5 + 16

security
→ 17

implementation order
→ 20
```

Do not re-read all references for every task.  
Open only the reference relevant to the current change.
