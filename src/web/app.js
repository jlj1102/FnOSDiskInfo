"use strict";

const $ = (id) => document.getElementById(id);

const PAGE_SIZE = 12;
const GEO_PAGE_SIZE = 8;
const INTERVALS = [10, 60, 180, 300, 600, 1800, 3600];
const RAW_FORMATS = ["hex", "dec", "byte2", "byte1"];
const ZOOMS = ["100", "125", "150", "200", "250", "300", "auto"];
const FONT_SIZES = ["11", "12", "13", "14"];

const state = {
  disks: [],
  id: localStorage.getItem("cdifnos.id") || null,
  attrs: [],
  updatedAt: null,
  error: null,
  version: null,
  page: 0,
  geoPage: 0,
  settings: null,
  alarms: [],
  ui: {
    unit: localStorage.getItem("cdifnos.unit") || "C",
    raw: localStorage.getItem("cdifnos.raw") || "hex",
    hideSerial: localStorage.getItem("cdifnos.hideSerial") === "1",
    hideSmart: localStorage.getItem("cdifnos.hideSmart") === "1",
    hideNoSmart: localStorage.getItem("cdifnos.hideNoSmart") === "1",
    sort: localStorage.getItem("cdifnos.sort") || "device",
    zoom: localStorage.getItem("cdifnos.zoom") || "100",
    fontFamily: localStorage.getItem("cdifnos.fontFamily") || "",
    fontSize: localStorage.getItem("cdifnos.fontSize") || "12"
  }
};

async function api(path, opts) {
  const r = await fetch(path, opts);
  const body = await r.json().catch(() => ({}));
  if (!r.ok) throw new Error(body.error || r.statusText);
  return body;
}

function currentDisk() {
  return state.disks.find((d) => d.id === state.id) || null;
}

function visibleDisks() {
  const list = state.disks.filter((d) => !(state.ui.hideNoSmart && (d.error || (d.health === "unknown" && !(d.attributes && d.attributes.length)))));
  const s = state.ui.sort;
  return [...list].sort((a, b) => {
    if (s === "model") return (a.model || "").localeCompare(b.model || "");
    if (s === "serial") return (a.serial || "").localeCompare(b.serial || "");
    return (a.device || "").localeCompare(b.device || "");
  });
}

function fmtCapacity(bytes) {
  if (!bytes) return "—";
  const gb = bytes / 1e9;
  return gb >= 1000 ? (gb / 1000).toFixed(2) + " TB" : gb.toFixed(1) + " GB";
}

function fmtTemp(c) {
  if (c == null) return "—";
  return state.ui.unit === "F" ? Math.round((c * 9) / 5 + 32) + "°F" : c + "°C";
}

function healthClass(h) {
  return ["good", "caution", "bad"].includes(h) ? h : "unknown";
}

function fmtNum(n) {
  return n == null ? "—" : Number(n).toLocaleString();
}

let toastTimer = null;
function toast(msg) {
  const el = $("toast");
  el.textContent = msg;
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => {
    el.textContent = "";
  }, 3500);
}

async function runRequest(path, body) {
  const r = await api(path, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body || {})
  });
  const id = r.request;
  for (let i = 0; i < 120; i++) {
    await new Promise((res) => setTimeout(res, 1000));
    try {
      const st = await api("/api/requests/" + encodeURIComponent(id));
      if (st.state === "done") return st;
    } catch (e) {
      // request may still be queued
    }
  }
  return { error: "timeout" };
}

// ---------- rendering ----------

function renderTabs() {
  const list = visibleDisks();
  const pages = Math.max(1, Math.ceil(list.length / PAGE_SIZE));
  state.page = Math.min(Math.max(0, state.page), pages - 1);
  const slice = list.slice(state.page * PAGE_SIZE, (state.page + 1) * PAGE_SIZE);

  const box = $("tabs");
  box.textContent = "";
  for (const d of slice) {
    const cls = d.error ? "unknown" : healthClass(d.health);
    const b = document.createElement("button");
    b.type = "button";
    b.className = "tab " + cls + (d.id === state.id ? " active" : "");
    const active = d.id === state.id;
    const icon = document.createElement("span");
    icon.className = "ticon";
    const src = tabIconSrc(cls, active ? 3 : 0) || themeImg("status_" + cls + "_mini", "status_" + cls);
    let iconImg = null;
    if (src) {
      icon.classList.add("img");
      iconImg = document.createElement("img");
      iconImg.src = src;
      iconImg.alt = "";
      icon.append(iconImg);
    }
    const st = document.createElement("span");
    st.className = "st";
    st.textContent = (d.error ? t("unknown") : t(cls)) + (d.stale ? " *" : "");
    const tm = document.createElement("span");
    tm.className = "tm";
    tm.textContent = fmtTemp(d.temperature);
    b.append(icon, st, tm);
    if (iconImg) {
      b.addEventListener("mouseenter", () => {
        const u = tabIconSrc(cls, 1);
        if (u) {
          iconImg.src = u;
        }
      });
      b.addEventListener("mouseleave", () => {
        const u = tabIconSrc(cls, d.id === state.id ? 3 : 0);
        if (u) {
          iconImg.src = u;
        }
      });
    }
    b.addEventListener("click", () => selectDisk(d.id));
    box.append(b);
  }

  const showPager = list.length > PAGE_SIZE;
  $("preDisk").classList.toggle("hidden", !showPager);
  $("nextDisk").classList.toggle("hidden", !showPager);
}

function selectDisk(id) {
  state.id = id;
  const list = visibleDisks();
  const idx = Math.max(0, list.findIndex((d) => d.id === id));
  state.page = Math.floor(idx / PAGE_SIZE);
  state.geoPage = Math.floor(idx / GEO_PAGE_SIZE);
  localStorage.setItem("cdifnos.id", id);
  refresh();
}

// ---------- CDI Shizuku geometry (active when the theme ships art) ----------

function setGeometryMode() {
  document.body.classList.toggle("cdi-geometry", Theme.hasArt());
}

// geometric info rows accept plain strings or {v, tip} (CDI's tooltips).
function geoRows(boxId, items, cell) {
  const box = $(boxId);
  box.textContent = "";
  for (const it of items) {
    const text = typeof it === "string" ? it : it.v;
    const div = document.createElement("div");
    div.className = "geo-row" + (cell ? " geo-cell" : "");
    div.textContent = text;
    if (it && typeof it === "object" && it.tip) {
      div.title = it.tip;
    }
    box.append(div);
  }
}

// humanGB mirrors the backend's tooltip size format ("4.534 TB").
function humanGB(gb) {
  if (gb == null) {
    return "";
  }
  if (gb >= 1024 * 1024) {
    return (gb / 1024 / 1024).toFixed(3) + " PB";
  }
  if (gb >= 1024) {
    return (gb / 1024).toFixed(3) + " TB";
  }
  return gb + " GB";
}

// powerOnTip is CDI's "X years X days X hours" tooltip.
function powerOnTip(hours) {
  if (hours == null) {
    return "";
  }
  const y = Math.floor(hours / 24 / 365);
  const dd = Math.floor(hours / 24) % 365;
  const hh = hours % 24;
  const parts = [];
  if (y > 0) {
    parts.push(y + " " + t("years"));
  }
  parts.push(dd + " " + t("days"), hh + " " + t("hours"));
  return parts.join(" ");
}

// featureTipText lists only the features the disk actually has, with CDI's
// per-feature descriptions.
const FEATURE_TIPS = [
  ["S.M.A.R.T.", "feat_smart"],
  ["APM", "feat_apm"],
  ["AAM", "feat_aam"],
  ["NCQ", "feat_ncq"],
  ["TRIM", "feat_trim"],
  ["DevSleep", "feat_devsleep"],
  ["Streaming", "feat_streaming"],
  ["GPL", "feat_gpl"],
  ["VolatileWriteCache", "feat_vwc"]
];

function featureTipText(d) {
  if (!d || !d.features || !d.features.length) {
    return "";
  }
  return d.features.map((f) => {
    const hit = FEATURE_TIPS.find((x) => x[0] === f);
    return hit ? t(hit[1]) : f;
  }).join("\n");
}

// rotationCell is CDI's Rotation Rate row: NAND writes / "---- (SSD)" / RPM.
function rotationCell(d) {
  if (!d) {
    return { v: "--" };
  }
  if (d.rotation_rate > 0) {
    return { v: fmtNum(d.rotation_rate) + " " + t("rpm") };
  }
  if (d.is_ssd) {
    return { v: "---- (" + t("ssd") + ")" };
  }
  return { v: "--" };
}

function setGeoArt(el, src, text, px) {
  el.textContent = "";
  el.classList.toggle("has-img", !!src);
  if (src) {
    const img = document.createElement("img");
    img.src = src;
    img.alt = "";
    el.append(img);
  }
  const span = document.createElement("span");
  span.className = "overlay";
  span.style.fontSize = px + "px";
  span.textContent = text;
  el.append(span);
}

function renderGeoDisks() {
  const list = visibleDisks();
  const pages = Math.max(1, Math.ceil(list.length / GEO_PAGE_SIZE));
  state.geoPage = Math.min(Math.max(0, state.geoPage), pages - 1);
  const slice = list.slice(state.geoPage * GEO_PAGE_SIZE, (state.geoPage + 1) * GEO_PAGE_SIZE);
  const box = $("geoDisks");
  box.textContent = "";
  for (const d of slice) {
    const cls = d.error ? "unknown" : healthClass(d.health);
    const active = d.id === state.id;
    const b = document.createElement("button");
    b.type = "button";
    b.className = "geo-disk";
    b.title = d.model || d.device;
    const src = geoIconSrc(cls, active ? 3 : 0);
    if (src) {
      const img = document.createElement("img");
      img.src = src;
      img.alt = "";
      b.append(img);
      b.addEventListener("mouseenter", () => {
        const u = geoIconSrc(cls, 1);
        if (u) img.src = u;
      });
      b.addEventListener("mouseleave", () => {
        const u = geoIconSrc(cls, active ? 3 : 0);
        if (u) img.src = u;
      });
    }
    const stack = document.createElement("span");
    stack.className = "stack";
    const lines = [
      ["t1", d.error ? t("unknown") : t(cls)],
      ["t2", fmtTemp(d.temperature)],
      ["t3", (d.device || "").replace(/^\/dev\//, "")]
    ];
    for (const [c, v] of lines) {
      const s = document.createElement("span");
      s.className = c;
      s.textContent = v;
      stack.append(s);
    }
    b.append(stack);
    b.addEventListener("click", () => selectDisk(d.id));
    box.append(b);
  }
  const showPager = list.length > GEO_PAGE_SIZE;
  $("geoPreDisk").classList.toggle("hidden", !showPager);
  $("geoNextDisk").classList.toggle("hidden", !showPager);
}

function renderGeoInfo() {
  const d = currentDisk();
  const dash = "--";
  const model = d ? (d.model || d.device) + " : " + fmtCapacity(d.capacity_bytes) : (state.error || t("no_disks"));
  $("geoModel").textContent = model;
  $("geoModel").title = d ? model + (d.firmware ? " [" + d.firmware + "]" : "") : "";
  const serial = d && d.serial ? (state.ui.hideSerial ? "********" : d.serial) : dash;
  const features = d && d.features && d.features.length ? d.features.join(", ") : dash;
  geoRows("geoLabelsLeft", [t("firmware"), t("serial"), t("interface"), t("transfer_mode"), t("drive_map"), t("standard"), t("feature")], false);
  geoRows("geoValuesLeft", d
    ? [{ v: d.firmware || dash },
       { v: serial, tip: state.ui.hideSerial ? "" : (d.serial || "") },
       { v: d.protocol || dash },
       { v: d.transfer_mode || dash, tip: t("tip_transfer_mode") },
       { v: d.device || dash },
       { v: d.standard || dash, tip: t("tip_standard") },
       { v: features, tip: featureTipText(d) }]
    : [dash, dash, dash, dash, dash, dash, dash], true);

  // CDI reuses the Buffer Size / NV Cache / Rotation Rate rows for the SSD
  // and NVMe totals (DiskInfoDlgUpdate.cpp ChangeDisk).
  const hostReads = d && d.host_reads_gb != null ? d.host_reads_gb : null;
  const hostWrites = d && d.host_writes_gb != null ? d.host_writes_gb : null;
  const nandWrites = d && d.nand_writes_gb != null ? d.nand_writes_gb : null;
  const gbRow = (v) => ({ v: v == null ? dash : fmtNum(v) + " GB", tip: humanGB(v) });
  const hours = d && d.power_on_hours != null ? { v: fmtNum(d.power_on_hours) + " " + t("hours"), tip: powerOnTip(d.power_on_hours) } : { v: dash };
  geoRows("geoLabelsRight", [
    t(hostReads != null ? "total_host_reads" : "buffer_size"),
    t(hostWrites != null ? "total_host_writes" : "nv_cache"),
    t(nandWrites != null ? "total_nand_writes" : "rotation"),
    t("power_on_count"), t("power_on")
  ], false);
  geoRows("geoValuesRight", d
    ? [gbRow(hostReads), gbRow(hostWrites), nandWrites != null ? gbRow(nandWrites) : rotationCell(d),
       { v: fmtNum(d.power_on_count) }, hours]
    : [dash, dash, dash, dash, dash], true);

  const cls = d ? healthClass(d.health) : "unknown";
  const lifePct = d && d.life != null ? d.life : null;
  $("geoStatusLabel").textContent = t("health");
  // CDI Shizuku puts the life percentage in the health status text:
  // m_DiskStatus.Format("%s (%d %%)") — DiskInfoDlgUpdate.cpp.
  const statusText = d ? t(cls) + (lifePct == null ? "" : " (" + lifePct + " %)") : dash;
  setGeoArt($("geoStatusArt"), d ? themeImg("status_" + cls, "disk_" + cls) : "", statusText, 16);
  const tempCls = !d || d.temperature == null ? "unknown" : (d.temperature >= (d.alarm_temp || 99) ? "bad" : "good");
  $("geoTempLabel").textContent = t("temperature");
  setGeoArt($("geoTempArt"), d ? themeImg("temp_" + tempCls) : "", d ? fmtTemp(d.temperature) : dash, 18);

  const lifeBox = $("geoLife");
  lifeBox.textContent = "";
  // CDI swaps in the "100" art variant for a full-life good disk.
  const sdSlot = lifePct === 100 && cls === "good" ? "sd_good100" : "sd_" + cls;
  const lifeSrc = d ? themeImg(sdSlot, "sd_" + cls) : "";
  if (lifeSrc) {
    const img = document.createElement("img");
    img.src = lifeSrc;
    img.alt = "";
    lifeBox.append(img);
  }
}

function metaRow(c1, v1, t1, c2, v2, t2) {
  const tr = document.createElement("tr");
  for (const [label, value, tip] of [[c1, v1, t1], [c2, v2, t2]]) {
    const th = document.createElement("th");
    th.textContent = label;
    const td = document.createElement("td");
    td.textContent = value;
    if (tip) {
      td.title = tip;
    }
    tr.append(th, td);
  }
  return tr;
}

function applyThemeImages() {
  const logo = $("themeLogo");
  if (Theme.images.logo) {
    logo.src = Theme.images.logo;
    logo.classList.remove("hidden");
  } else {
    logo.removeAttribute("src");
    logo.classList.add("hidden");
  }
  const pre = Theme.frameUrl("pre", 0) || Theme.images.pre;
  const next = Theme.frameUrl("next", 0) || Theme.images.next;
  $("preDisk").innerHTML = pre ? `<img src="${pre}" alt="">` : "&#9664;";
  $("nextDisk").innerHTML = next ? `<img src="${next}" alt="">` : "&#9654;";
  $("geoPreDisk").innerHTML = pre ? `<img src="${pre}" alt="">` : "&#9664;";
  $("geoNextDisk").innerHTML = next ? `<img src="${next}" alt="">` : "&#9654;";
}

function themeImg(...slots) {
  for (const s of slots) {
    if (Theme.images[s]) {
      return Theme.images[s];
    }
  }
  return "";
}

// tabIconSrc picks the disk-button art frame: 0 normal, 1 hover, 3 selected
// (CDI sprite strip order, CommonFx.h).
function tabIconSrc(cls, frame) {
  for (const slot of ["disk_" + cls + "_mini", "disk_" + cls]) {
    const u = Theme.frameUrl(slot, frame);
    if (u) {
      return u;
    }
    if (frame === 0 && Theme.images[slot]) {
      return Theme.images[slot];
    }
  }
  return "";
}

// geoIconSrc is for the 84x48 geometry buttons: full-size art first, the
// 42x48 mini art (meant for the small classic tabs) would blur when stretched.
function geoIconSrc(cls, frame) {
  for (const slot of ["disk_" + cls, "disk_" + cls + "_mini"]) {
    const u = Theme.frameUrl(slot, frame);
    if (u) {
      return u;
    }
    if (frame === 0 && Theme.images[slot]) {
      return Theme.images[slot];
    }
  }
  return "";
}

function setBoxImage(el, src, text, cls) {
  el.textContent = "";
  el.title = text;
  el.className = "box " + cls + (src ? " has-img" : "");
  el.style.height = "";
  if (src) {
    const img = document.createElement("img");
    img.className = "sicon";
    img.src = src;
    img.alt = text;
    img.addEventListener("load", () => {
      // life art (e.g. Shizuku SD* 128x192) drives the block height like CDI
      if (cls.indexOf("life") >= 0 && img.naturalWidth > 0) {
        const w = el.clientWidth || 132;
        const h = Math.max(30, Math.min(260, Math.round((w * img.naturalHeight) / img.naturalWidth)));
        el.style.height = h + "px";
      }
    });
    el.append(img);
  } else {
    el.textContent = text;
  }
}

function renderHead() {
  const meta = $("meta");
  meta.textContent = "";
  const d = currentDisk();
  if (!d) {
    $("headline").textContent = state.error ? state.error : t("no_disks");
    setBoxImage($("health"), "", "—", "health unknown");
    setBoxImage($("temp"), "", "—", "temp");
    setBoxImage($("life"), "", "—", "life");
    return;
  }
  const cls = healthClass(d.health);
  $("headline").textContent = (d.model || d.device) + " : " + fmtCapacity(d.capacity_bytes);
  setBoxImage($("health"), themeImg("status_" + cls, "disk_" + cls), t(cls), "health " + cls);
  const tempCls = d.temperature == null ? "unknown" : (d.temperature >= (d.alarm_temp || 99) ? "bad" : "good");
  setBoxImage($("temp"), themeImg("temp_" + tempCls), fmtTemp(d.temperature), "temp");
  setBoxImage($("life"), themeImg("sd_" + cls), d.life == null ? "—" : t("life") + " " + d.life + " %", "life");

  const serial = d.serial ? (state.ui.hideSerial ? "********" : d.serial) : "—";
  const hostReads = d.host_reads_gb != null ? d.host_reads_gb : null;
  const hostWrites = d.host_writes_gb != null ? d.host_writes_gb : null;
  const nandWrites = d.nand_writes_gb != null ? d.nand_writes_gb : null;
  const gbText = (v) => (v == null ? "—" : fmtNum(v) + " GB");
  const rot = nandWrites != null
    ? { label: t("total_nand_writes"), value: gbText(nandWrites), tip: humanGB(nandWrites) }
    : { label: t("rotation"), value: rotationCell(d).v, tip: "" };
  const hours = d.power_on_hours == null ? { v: "—" } : { v: fmtNum(d.power_on_hours) + " " + t("hours"), tip: powerOnTip(d.power_on_hours) };
  meta.append(
    metaRow(t("firmware"), d.firmware || "—", "", rot.label, rot.value, rot.tip),
    metaRow(t("serial"), serial, state.ui.hideSerial ? "" : (d.serial || ""), t("power_on_count"), fmtNum(d.power_on_count), ""),
    metaRow(t("interface"), d.protocol || "—", "", t("power_on"), hours.v, hours.tip),
    metaRow(t("device"), d.device, "", t("last_test"), d.self_test ? d.self_test.type + ": " + d.self_test.status : "—", "")
  );
  if (hostReads != null || hostWrites != null) {
    meta.append(metaRow(
      hostReads != null ? t("total_host_reads") : "", hostReads != null ? gbText(hostReads) : "", humanGB(hostReads),
      hostWrites != null ? t("total_host_writes") : "", hostWrites != null ? gbText(hostWrites) : "", humanGB(hostWrites)
    ));
  }
  if (d.status_reasons && d.status_reasons.length) {
    const tr = document.createElement("tr");
    const th = document.createElement("th");
    th.textContent = t("status_reasons");
    const td = document.createElement("td");
    td.className = "reasons";
    td.colSpan = 3;
    td.textContent = d.status_reasons.join("; ");
    tr.append(th, td);
    meta.append(tr);
  }
  if (d.error) {
    const tr = document.createElement("tr");
    const th = document.createElement("th");
    th.textContent = t("smart");
    const td = document.createElement("td");
    td.className = "reasons";
    td.colSpan = 3;
    td.textContent = d.error;
    tr.append(th, td);
    meta.append(tr);
  }
}

// attrCols mirrors CDI's per-vendor list layout (DiskInfoDlgUpdate.cpp
// RebuildListHeader): NVMe and Indilinx hide Current/Worst/Threshold,
// JMicron60x hides Worst/Threshold (zero-width columns in CDI).
function attrCols(d) {
  const key = (d && d.smart_key) || "";
  const cols = ["", "col_id", "col_attr"];
  if (!(d && d.nvme) && key !== "SmartIndilinx" && key !== "SmartScsi") {
    cols.push("col_cur");
    if (key !== "SmartJMicron60x") {
      cols.push("col_worst", "col_thr");
    }
  }
  cols.push("col_raw");
  return cols;
}

function renderAttrHead() {
  const cols = attrCols(currentDisk());
  const tr = $("attrhead");
  tr.textContent = "";
  for (const c of cols) {
    const th = document.createElement("th");
    th.textContent = c ? t(c) : "";
    tr.append(th);
  }
}

// ledIcon is CDI's per-attribute status LED: theme art (led_* slot, fallback
// chain included) or the bundled icons converted from CDI's res/*.ico.
function ledIcon(status) {
  const s = status === "good" || status === "caution" || status === "bad" ? status : "unknown";
  return Theme.images["led_" + s] || "/icons/led_" + s + ".png";
}

// fmtRaw mirrors CDI's raw value formats (DiskInfoDlgUpdate.cpp): hex is
// zero-padded uppercase 6 bytes (NVMe: 7, with the reserved byte), dec is the
// 48-bit value, byte2/byte1 are big-endian words/bytes.
function fmtRaw(a, nvme) {
  const v = a.raw_value;
  if (v === undefined || v === null) return a.raw || "";
  const u = ((Math.trunc(Number(v)) || 0) % 0x1000000000000 + 0x1000000000000) % 0x1000000000000;
  const bytes = [];
  for (let i = 5; i >= 0; i--) bytes.push(Math.floor(u / 256 ** i) % 256);
  switch (state.ui.raw) {
    case "dec": return String(u);
    case "byte2": {
      const words = [(bytes[0] << 8) | bytes[1], (bytes[2] << 8) | bytes[3], (bytes[4] << 8) | bytes[5]];
      return (nvme ? ["0"].concat(words) : words).join(" ");
    }
    case "byte1":
      return (nvme ? ["0"].concat(bytes) : bytes).join(" ");
    default:
      return (nvme ? "00" : "") + bytes.map((b) => b.toString(16).toUpperCase().padStart(2, "0")).join("");
  }
}

function renderAttrs() {
  $("attrwrap").classList.toggle("hidden", state.ui.hideSmart);
  const d = currentDisk() || {};
  const isNvme = !!d.nvme;
  const isScsi = d.smart_key === "SmartScsi";
  const cols = attrCols(d);
  const showCur = cols.includes("col_cur");
  const showWorst = cols.includes("col_worst");
  const showThr = cols.includes("col_thr");
  renderAttrHead();
  const tb = $("attrs");
  tb.textContent = "";
  if (!state.attrs.length) {
    const tr = document.createElement("tr");
    const td = document.createElement("td");
    td.colSpan = cols.length;
    td.className = "empty";
    td.textContent = t("no_attrs");
    tr.append(td);
    tb.append(tr);
    return;
  }
  for (const a of state.attrs) {
    const tr = document.createElement("tr");
    const led = document.createElement("td");
    led.className = "led";
    const ledImg = document.createElement("img");
    ledImg.src = ledIcon(a.status);
    ledImg.alt = "";
    led.append(ledImg);
    tr.append(led);
    const cells = [
      a.id.toString(16).toUpperCase().padStart(2, "0"),
      attrName(a.id, a.name, d.smart_key, d.is_ssd, isNvme)
    ];
    if (showCur) cells.push(a.current);
    if (showWorst) cells.push(a.worst);
    if (showThr) cells.push(a.threshold);
    // SCSI pseudo rows carry their own raw text (counters, status words),
    // not an ATA-style 48-bit value.
    cells.push(isScsi ? (a.raw || "") : fmtRaw(a, isNvme));
    for (const c of cells) {
      const td = document.createElement("td");
      td.textContent = c;
      tr.append(td);
    }
    tb.append(tr);
  }
}

function renderStatus() {
  const el = $("status");
  if (state.error) {
    el.textContent = state.error;
    el.className = "status err";
    return;
  }
  el.className = "status";
  el.textContent = state.updatedAt ? t("updated") + " " + new Date(state.updatedAt).toLocaleTimeString() : "";
}

function updateBanner() {
  const last = state.alarms[state.alarms.length - 1];
  const seen = Number(localStorage.getItem("cdifnos.alarmSeen") || 0);
  const banner = $("banner");
  const ind = $("alarmIndicator");
  const unseen = last && last.t * 1000 > seen;
  if (unseen && last.to !== "good") {
    banner.classList.remove("hidden");
    $("bannerText").textContent = new Date(last.t * 1000).toLocaleString() + " — " + last.message;
  } else {
    banner.classList.add("hidden");
  }
  ind.classList.toggle("hidden", !unseen || last.to === "good");
}

// ---------- refresh ----------

async function refresh() {
  try {
    const data = await api("/api/disks");
    state.disks = data.disks || [];
    state.updatedAt = data.updated_at;
    state.error = data.error || null;
    state.version = data.version || state.version;
    const list = visibleDisks();
    if (!list.some((d) => d.id === state.id)) {
      state.id = list.length ? list[0].id : null;
    }
    setGeometryMode();
    if (document.body.classList.contains("cdi-geometry")) {
      renderGeoDisks();
      renderGeoInfo();
    } else {
      renderTabs();
      renderHead();
    }
    state.attrs = [];
    const d = currentDisk();
    if (d && !d.error) {
      try {
        const s = await api("/api/disks/" + encodeURIComponent(d.id) + "/smart");
        state.attrs = s.attributes || [];
      } catch (e) {
        // keep empty
      }
    }
    renderAttrs();
    renderStatus();
  } catch (e) {
    state.error = e.message;
    renderHead();
    renderStatus();
  }
  refreshMenus();
}

async function loadAlarms() {
  try {
    state.alarms = (await api("/api/alarms")).alarms || [];
  } catch (e) {
    state.alarms = [];
  }
  updateBanner();
}

async function loadSettings() {
  try {
    state.settings = await api("/api/settings");
  } catch (e) {
    state.settings = null;
  }
}

function refreshMenus() {
  if (Menubar.isOpen()) {
    Menubar.refresh();
  } else {
    buildMenus();
  }
}

function excludedSet() {
  return new Set((state.settings && state.settings.exclude_disks) || []);
}

async function putSettings(patch) {
  await loadSettings();
  if (!state.settings) {
    return;
  }
  const payload = {
    interval_seconds: state.settings.interval_seconds,
    default: state.settings.default,
    disks: state.settings.disks || {},
    exclude_disks: state.settings.exclude_disks || []
  };
  Object.assign(payload, patch || {});
  await api("/api/settings", {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload)
  });
  await loadSettings();
  refreshMenus();
}

async function toggleExclude(id) {
  const list = excludedSet();
  if (list.has(id)) {
    list.delete(id);
  } else {
    list.add(id);
  }
  try {
    await putSettings({ exclude_disks: [...list] });
    refresh();
  } catch (e) {
    toast(e.message);
  }
}

// ---------- display preferences ----------

function applyZoom() {
  let factor = 1;
  if (state.ui.zoom === "auto") {
    factor = (window.devicePixelRatio || 1) >= 1.5 ? 1.25 : 1;
  } else {
    factor = Number(state.ui.zoom) / 100 || 1;
  }
  document.body.style.zoom = factor;
}

function applyFont() {
  const el = document.documentElement;
  if (state.ui.fontFamily) {
    el.style.setProperty("--cdi-font-family", state.ui.fontFamily);
  } else {
    el.style.removeProperty("--cdi-font-family");
  }
  el.style.setProperty("--cdi-font-size", state.ui.fontSize + "px");
}

function setZoom(z) {
  state.ui.zoom = z;
  localStorage.setItem("cdifnos.zoom", z);
  applyZoom();
  refreshMenus();
}

function setFont(family, size) {
  if (family !== null) {
    state.ui.fontFamily = family;
    localStorage.setItem("cdifnos.fontFamily", family);
  }
  if (size !== null) {
    state.ui.fontSize = size;
    localStorage.setItem("cdifnos.fontSize", size);
  }
  applyFont();
  refreshMenus();
}

function togglePref(key, render) {
  state.ui[key] = !state.ui[key];
  localStorage.setItem("cdifnos." + key, state.ui[key] ? "1" : "0");
  render();
  refreshMenus();
}

// ---------- language ----------

function currentLangPref() {
  return localStorage.getItem("cdifnos.lang") || "auto";
}

function setLangPref(lang) {
  if (lang === "auto") {
    localStorage.removeItem("cdifnos.lang");
    setLang(browserLang());
  } else {
    localStorage.setItem("cdifnos.lang", lang);
    setLang(lang);
  }
  applyStaticTexts();
  refresh();
}

// ---------- actions ----------

function rescanNow() {
  api("/api/disks/rescan", { method: "POST" }).catch(() => {});
  setTimeout(refresh, 800);
}

function downloadReport() {
  const d = currentDisk();
  if (d) {
    location.href = "/api/disks/" + encodeURIComponent(d.id) + "/report.txt";
  }
}

async function copyInfo() {
  const d = currentDisk();
  if (!d) return;
  try {
    const text = await (await fetch("/api/disks/" + encodeURIComponent(d.id) + "/report.txt")).text();
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(text);
    } else {
      const ta = document.createElement("textarea");
      ta.value = text;
      ta.style.position = "fixed";
      ta.style.opacity = "0";
      document.body.append(ta);
      ta.select();
      document.execCommand("copy");
      ta.remove();
    }
    toast(t("copy_ok"));
  } catch (e) {
    toast(t("copy_fail"));
  }
}

async function setIntervalSec(sec) {
  try {
    await putSettings({ interval_seconds: sec });
    toast(t("saved"));
  } catch (e) {
    toast(e.message);
  }
}

async function importThemeFile(file) {
  try {
    const res = await Theme.importFile(file);
    const first = res.themes && res.themes[0];
    if (first) {
      await Theme.apply(first.id);
    }
    setGeometryMode();
    applyThemeImages();
    const count = (res.themes || []).length;
    if (res.skipped && res.skipped.length) {
      toast(t("theme_import_ok") + " (" + count + ") — " + res.skipped[0]);
    } else {
      toast(t("theme_import_ok") + (count > 1 ? " (" + count + ")" : ""));
    }
  } catch (e) {
    toast(t("theme_import_fail") + ": " + e.message);
  }
  buildMenus();
  refresh();
}

async function deleteTheme(id) {
  id = id || Theme.current();
  if (!confirm(t("theme_delete_confirm") + "\n" + id)) {
    return;
  }
  try {
    await Theme.remove(id);
    if (Theme.current() === id) {
      await Theme.apply("classic");
      setGeometryMode();
      applyThemeImages();
    }
    toast(t("saved"));
  } catch (e) {
    toast(t("theme_import_fail") + ": " + e.message);
  }
  buildMenus();
  refresh();
}

function openAbout() {
  $("abTitle").textContent = t("about");
  const box = $("abText");
  box.textContent = "";
  const p1 = document.createElement("p");
  p1.textContent = "DiskInfo for fnOS " + (state.version || "");
  const p2 = document.createElement("p");
  p2.textContent = t("about_text");
  const p3 = document.createElement("p");
  p3.innerHTML = '<a href="https://crystalmark.info/en/software/crystaldiskinfo/" target="_blank" rel="noopener">CrystalDiskInfo</a> · ' +
    '<a href="https://www.smartmontools.org/" target="_blank" rel="noopener">smartmontools</a> · ' +
    '<a href="https://en.wikipedia.org/wiki/S.M.A.R.T." target="_blank" rel="noopener">S.M.A.R.T.</a>';
  const p4 = document.createElement("p");
  p4.textContent = t("about_graph");
  box.append(p1, p2, p3, p4);
  $("dlgAbout").showModal();
}

// ---------- menus ----------

async function setFallbackTheme(id) {
  localStorage.setItem("cdifnos.fallback", id);
  await Theme.apply(Theme.current());
  setGeometryMode();
  applyThemeImages();
  refresh();
}

function buildMenus() {
  const themeItems = Theme.list.map((ti) => ({
    label: ti.name,
    checked: () => Theme.current() === ti.id,
    action: async () => {
      await Theme.apply(ti.id);
      setGeometryMode();
      applyThemeImages();
      refresh();
    }
  }));
  const currentTheme = Theme.current();
  const importedThemes = Theme.list.filter((ti) => !ti.builtin);

  const menus = [
    {
      label: t("menu_file"),
      items: [
        { label: t("save_text"), action: downloadReport },
        { label: t("save_image"), action: () => snapshotMain() }
      ]
    },
    {
      label: t("menu_edit"),
      items: [
        { label: t("copy"), action: copyInfo }
      ]
    },
    {
      label: t("menu_function"),
      items: [
        { label: t("refresh") + " (F5)", action: () => refresh() },
        {
          label: t("auto_refresh"),
          items: INTERVALS.map((sec) => ({
            label: sec >= 60 ? sec / 60 + " min" : sec + " s",
            checked: () => (state.settings ? state.settings.interval_seconds === sec : sec === 10),
            action: () => setIntervalSec(sec)
          }))
        },
        { label: t("rescan") + " (F6)", action: rescanNow },
        { label: t("selftest"), action: openSelftest },
        { separator: true },
        { label: t("graph"), action: openGraph },
        { label: t("g_options"), action: openGraphOptions },
        { separator: true },
        { label: t("hide_serial_number"), checked: () => state.ui.hideSerial, action: () => togglePref("hideSerial", renderHead) },
        {
          label: t("alerts"),
          items: [{ label: t("alarm_list"), action: openAlarms }]
        },
        {
          label: t("advanced"),
          items: [
            { label: t("aam_apm"), action: openAamApm },
            { label: t("health_status_setting"), action: openSettings },
            { label: t("temperature_setting"), action: openSettings },
            {
              label: t("temp_unit"),
              items: [
                { label: t("celsius"), checked: () => state.ui.unit === "C", action: () => { state.ui.unit = "C"; localStorage.setItem("cdifnos.unit", "C"); refresh(); } },
                { label: t("fahrenheit"), checked: () => state.ui.unit === "F", action: () => { state.ui.unit = "F"; localStorage.setItem("cdifnos.unit", "F"); refresh(); } }
              ]
            },
            {
              label: t("raw_values"),
              items: RAW_FORMATS.map((r) => ({
                label: t(r),
                checked: () => state.ui.raw === r,
                action: () => {
                  state.ui.raw = r;
                  localStorage.setItem("cdifnos.raw", r);
                  renderAttrs();
                }
              }))
            },
            { label: t("hide_smart"), checked: () => state.ui.hideSmart, action: () => togglePref("hideSmart", renderAttrs) },
            { label: t("hide_no_smart"), checked: () => state.ui.hideNoSmart, action: () => togglePref("hideNoSmart", () => refresh()) },
            {
              label: t("auto_refresh_target"),
              items: state.disks.map((d) => ({
                label: d.model || d.device,
                checked: () => !excludedSet().has(d.id),
                action: () => toggleExclude(d.id)
              }))
            },
            {
              label: t("disk_sort"),
              items: [
                { label: t("sort_device"), checked: () => state.ui.sort === "device", action: () => { state.ui.sort = "device"; localStorage.setItem("cdifnos.sort", "device"); refresh(); } },
                { label: t("sort_model"), checked: () => state.ui.sort === "model", action: () => { state.ui.sort = "model"; localStorage.setItem("cdifnos.sort", "model"); refresh(); } },
                { label: t("sort_serial"), checked: () => state.ui.sort === "serial", action: () => { state.ui.sort = "serial"; localStorage.setItem("cdifnos.sort", "serial"); refresh(); } }
              ]
            }
          ]
        }
      ]
    },
    {
      label: t("menu_theme"),
      items: [
        ...themeItems,
        { separator: true },
        {
          label: t("fallback_theme"),
          items: [
            { label: t("none"), checked: () => Theme.fallback() === "none", action: () => setFallbackTheme("none") },
            { label: t("auto"), checked: () => Theme.fallback() === "auto", action: () => setFallbackTheme("auto") },
            ...(importedThemes.length
              ? [
                  { separator: true },
                  ...importedThemes.map((ti) => ({
                    label: ti.name,
                    checked: () => Theme.fallback() === ti.id,
                    action: () => setFallbackTheme(ti.id)
                  }))
                ]
              : [])
          ]
        },
        { separator: true },
        {
          label: t("zoom"),
          items: ZOOMS.map((z) => ({
            label: z === "auto" ? t("zoom_auto") : z + "%",
            checked: () => state.ui.zoom === z,
            action: () => setZoom(z)
          }))
        },
        {
          label: t("font_setting"),
          items: [
            { label: t("font_default"), checked: () => !state.ui.fontFamily, action: () => setFont("", null) },
            { label: "Microsoft YaHei", checked: () => state.ui.fontFamily === "Microsoft YaHei", action: () => setFont("Microsoft YaHei", null) },
            { label: "Segoe UI", checked: () => state.ui.fontFamily === "Segoe UI", action: () => setFont("Segoe UI", null) },
            { label: "Consolas", checked: () => state.ui.fontFamily === "Consolas", action: () => setFont("Consolas", null) },
            { separator: true },
            ...FONT_SIZES.map((s) => ({
              label: t("font_size") + " " + s,
              checked: () => state.ui.fontSize === s,
              action: () => setFont(null, s)
            }))
          ]
        },
        { separator: true },
        { label: t("theme_import"), action: () => $("themeFile").click() },
        ...(importedThemes.length
          ? [{
              label: t("theme_delete"),
              items: importedThemes.map((ti) => ({
                label: ti.name,
                action: () => deleteTheme(ti.id)
              }))
            }]
          : []),
        { separator: true },
        {
          label: t("lang"),
          items: [
            { label: t("lang_auto"), checked: () => currentLangPref() === "auto", action: () => setLangPref("auto") },
            { label: "English", checked: () => currentLangPref() === "en", action: () => setLangPref("en") },
            { label: "简体中文", checked: () => currentLangPref() === "zh-CN", action: () => setLangPref("zh-CN") },
            { label: "繁體中文", checked: () => currentLangPref() === "zh-TW", action: () => setLangPref("zh-TW") },
            { label: "日本語", checked: () => currentLangPref() === "ja", action: () => setLangPref("ja") }
          ]
        }
      ]
    },
    {
      label: t("menu_disk"),
      items: visibleDisks().map((d) => ({
        label: d.model || d.device,
        checked: () => d.id === state.id,
        action: () => selectDisk(d.id)
      }))
    },
    {
      label: t("menu_help"),
      items: [
        { label: t("about_smart"), action: () => window.open("https://en.wikipedia.org/wiki/S.M.A.R.T.", "_blank", "noopener") },
        { separator: true },
        { label: t("about"), action: openAbout }
      ]
    }
  ];
  Menubar.render($("menubar"), menus);
}

// ---------- dialogs: self-test ----------

function openSelftest() {
  const d = currentDisk();
  if (!d) return;
  $("stTitle").textContent = t("selftest") + " — " + (d.model || d.device);
  $("stLast").textContent = d.self_test
    ? t("last_test") + ": " + d.self_test.type + " / " + d.self_test.status
    : t("last_test") + ": —";
  $("stOut").textContent = "";
  $("dlgSelftest").showModal();
}

async function selftestAction(action) {
  const d = currentDisk();
  if (!d) return;
  $("stOut").textContent = "...";
  try {
    const url = "/api/disks/" + encodeURIComponent(d.id) + "/" + action;
    const body = action === "self-test" ? { type: $("stType").value } : {};
    const res = await runRequest(url, body);
    $("stOut").textContent = res.output || res.error || JSON.stringify(res);
  } catch (e) {
    $("stOut").textContent = e.message;
  }
  refresh();
}

// ---------- dialogs: AAM/APM ----------

function openAamApm() {
  const d = currentDisk();
  if (!d) return;
  $("aaTitle").textContent = t("aam_apm") + " — " + (d.model || d.device);
  $("aaCurrent").textContent = t("current") + ": AAM " + (d.aam ?? "—") + " / APM " + (d.apm ?? "—");
  $("aaAam").value = d.aam ?? "";
  $("aaApm").value = d.apm ?? "";
  $("aaOut").textContent = "";
  $("dlgAamApm").showModal();
}

function aamValue(v) {
  if (!v) {
    return "—";
  }
  return v.enabled ? String(v.level) : t("disabled");
}

function renderAamOutput(msg, raw) {
  const box = $("aaOut");
  box.textContent = "";
  const p = document.createElement("div");
  p.textContent = msg;
  box.append(p);
  if (raw && raw.trim() && raw.trim() !== msg.trim()) {
    const det = document.createElement("details");
    const sum = document.createElement("summary");
    sum.textContent = t("raw_output");
    const pre = document.createElement("pre");
    pre.textContent = raw;
    det.append(sum, pre);
    box.append(det);
  }
}

async function aamApmQuery() {
  const d = currentDisk();
  if (!d) return;
  $("aaOut").textContent = "...";
  try {
    const res = await runRequest("/api/disks/" + encodeURIComponent(d.id) + "/aam-apm-get", {});
    const raw = res.output || res.error || "";
    let parsed = null;
    try {
      parsed = JSON.parse(raw);
    } catch (e) {
      // smartctl did not return JSON; fall through to the raw text
    }
    if (parsed) {
      const aam = parsed.ata_aam;
      const apm = parsed.ata_apm;
      if (!aam && !apm) {
        renderAamOutput(t("not_available"), raw);
        return;
      }
      const msg = t("current") + ": AAM " + aamValue(aam) + " / APM " + aamValue(apm);
      $("aaCurrent").textContent = msg;
      renderAamOutput(msg, raw);
      return;
    }
    renderAamOutput(res.error || raw, raw);
  } catch (e) {
    renderAamOutput(e.message, "");
  }
}

async function aamApmAction(kind, value) {
  const d = currentDisk();
  if (!d) return;
  if (!/^(off|[1-9][0-9]{0,2})$/.test(value)) {
    renderAamOutput(t("value_required"), "");
    return;
  }
  $("aaOut").textContent = "...";
  try {
    const res = await runRequest("/api/disks/" + encodeURIComponent(d.id) + "/aam-apm", { kind, value });
    if (res.error && !res.output) {
      renderAamOutput(res.error, "");
    } else {
      renderAamOutput(res.output || JSON.stringify(res), res.output || "");
    }
  } catch (e) {
    renderAamOutput(e.message, "");
  }
  refresh();
}

// ---------- dialogs: graph ----------

function openGraph() {
  const d = currentDisk();
  const url = "graph.html?disk=" + encodeURIComponent(d ? d.id : "") +
    "&theme=" + encodeURIComponent(Theme.current());
  $("grFrame").src = url;
  $("dlgGraph").showModal();
}

function openGraphOptions() {
  // Reload so the option window picks up the current language and colors.
  $("optFrame").src = "option.html";
  $("dlgOption").showModal();
}

// ---------- dialogs: alarms ----------

function openAlarms() {
  const body = $("alBody");
  body.textContent = "";
  if (!state.alarms.length) {
    body.textContent = t("no_alarms");
    $("dlgAlarms").showModal();
    return;
  }
  const table = document.createElement("table");
  table.className = "alarms-table";
  const head = document.createElement("tr");
  for (const k of ["time", "disk", "kind", "from", "to"]) {
    const th = document.createElement("th");
    th.textContent = t(k);
    head.append(th);
  }
  table.append(head);
  for (const a of [...state.alarms].reverse()) {
    const tr = document.createElement("tr");
    const tds = [
      new Date(a.t * 1000).toLocaleString(),
      a.model || a.disk,
      t(a.kind === "temperature" ? "temperature_change" : "health_change"),
      a.from,
      a.to
    ];
    tds.forEach((v, i) => {
      const td = document.createElement("td");
      td.textContent = v;
      if (i === 4) td.className = healthClass(a.to);
      tr.append(td);
    });
    table.append(tr);
  }
  body.append(table);
  $("dlgAlarms").showModal();
}

// ---------- dialogs: settings ----------

function settingsNumber(label, value, min, max, cls) {
  const tr = document.createElement("tr");
  const th = document.createElement("th");
  th.textContent = label;
  const td = document.createElement("td");
  const input = document.createElement("input");
  input.type = "number";
  input.min = min;
  input.max = max;
  if (value !== undefined && value !== null) {
    input.value = value;
  }
  if (cls) input.className = cls;
  td.append(input);
  tr.append(th, td);
  return { tr, input };
}

function openSettings() {
  const s = state.settings;
  if (!s) return;
  const body = $("seBody");
  body.textContent = "";
  const wrap = document.createElement("div");
  wrap.className = "settings";

  const secServer = document.createElement("section");
  const h4s = document.createElement("h4");
  h4s.textContent = t("server");
  secServer.append(h4s);

  const tbl = document.createElement("table");
  const intervalRow = settingsNumber(t("interval_seconds"), s.interval_seconds, 2, 3600, "se-interval");
  tbl.append(intervalRow.tr);
  const cur = s.default || {};
  const dTemp = settingsNumber(t("alarm_temp"), cur.alarm_temp, 1, 100, "se-def-alarm_temp");
  const d05 = settingsNumber(t("threshold_05"), cur.threshold_05, 0, 100000, "se-def-threshold_05");
  const dc5 = settingsNumber(t("threshold_c5"), cur.threshold_c5, 0, 100000, "se-def-threshold_c5");
  const dc6 = settingsNumber(t("threshold_c6"), cur.threshold_c6, 0, 100000, "se-def-threshold_c6");
  const dff = settingsNumber(t("threshold_ff"), cur.threshold_ff, 0, 100, "se-def-threshold_ff");
  tbl.append(dTemp.tr, d05.tr, dc5.tr, dc6.tr, dff.tr);
  secServer.append(tbl);

  if (state.disks.length) {
    const h4p = document.createElement("h4");
    h4p.textContent = t("per_disk");
    secServer.append(h4p);
    const pt = document.createElement("table");
    const head = document.createElement("tr");
    for (const label of [t("disk"), t("alarm_temp"), "05", "C5", "C6", "FF"]) {
      const th = document.createElement("th");
      th.textContent = label;
      head.append(th);
    }
    pt.append(head);
    for (const d of state.disks) {
      const tr = document.createElement("tr");
      tr.dataset.id = d.id;
      const name = document.createElement("td");
      name.textContent = d.model || d.device;
      tr.append(name);
      const ov = (s.disks && s.disks[d.id]) || {};
      const eff = (s.effective && s.effective[d.id]) || {};
      const fields = ["alarm_temp", "threshold_05", "threshold_c5", "threshold_c6", "threshold_ff"];
      for (const f of fields) {
        const td = document.createElement("td");
        const input = document.createElement("input");
        input.type = "number";
        input.dataset.field = f;
        input.className = "se-ov";
        input.placeholder = String(eff[f] ?? "");
        if (ov[f] !== undefined && ov[f] !== null) {
          input.value = ov[f];
        }
        td.append(input);
        tr.append(td);
      }
      pt.append(tr);
    }
    secServer.append(pt);
  }
  wrap.append(secServer);
  body.append(wrap);
  $("dlgSettings").showModal();
}

async function saveSettings() {
  const s = {
    interval_seconds: Number($("seBody").querySelector(".se-interval").value),
    default: {
      alarm_temp: Number($("seBody").querySelector(".se-def-alarm_temp").value),
      threshold_05: Number($("seBody").querySelector(".se-def-threshold_05").value),
      threshold_c5: Number($("seBody").querySelector(".se-def-threshold_c5").value),
      threshold_c6: Number($("seBody").querySelector(".se-def-threshold_c6").value),
      threshold_ff: Number($("seBody").querySelector(".se-def-threshold_ff").value)
    },
    disks: {}
  };
  for (const tr of $("seBody").querySelectorAll("tr[data-id]")) {
    const ov = {};
    let any = false;
    for (const input of tr.querySelectorAll(".se-ov")) {
      const raw = input.value.trim();
      if (raw === "") continue;
      ov[input.dataset.field] = Number(raw);
      any = true;
    }
    if (any) s.disks[tr.dataset.id] = ov;
  }
  try {
    await api("/api/settings", {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(s)
    });
  } catch (e) {
    alert(e.message);
  }
  await loadSettings();
  $("dlgSettings").close();
  toast(t("saved"));
  refresh();
}

// ---------- init ----------

function applyStaticTexts() {
  $("stTitle").textContent = t("selftest");
  $("stStart").textContent = t("start");
  $("stAbort").textContent = t("abort");
  $("aaTitle").textContent = t("aam_apm");
  $("aaAamSet").textContent = t("set");
  $("aaAamOff").textContent = t("off");
  $("aaApmSet").textContent = t("set");
  $("aaApmOff").textContent = t("off");
  $("aaGet").textContent = t("query");
  $("seTitle").textContent = t("settings");
  $("seSave").textContent = t("apply");
  $("alTitle").textContent = t("alarms");
  $("abTitle").textContent = t("about");
  for (const b of document.querySelectorAll(".close")) {
    b.textContent = t("close");
    b.onclick = () => b.closest("dialog").close();
  }
}

function initControls() {
  const stType = $("stType");
  for (const v of ["short", "long", "conveyance"]) {
    const o = document.createElement("option");
    o.value = v;
    o.textContent = t(v);
    stType.append(o);
  }

  $("preDisk").onclick = () => {    state.page--;
    renderTabs();
  };
  $("nextDisk").onclick = () => {
    state.page++;
    renderTabs();
  };
  $("geoPreDisk").onclick = () => {
    state.geoPage--;
    renderGeoDisks();
  };
  $("geoNextDisk").onclick = () => {
    state.geoPage++;
    renderGeoDisks();
  };
  $("stStart").onclick = () => selftestAction("self-test");
  $("stAbort").onclick = () => selftestAction("abort-test");
  $("aaAamSet").onclick = () => aamApmAction("aam", $("aaAam").value.trim());
  $("aaAamOff").onclick = () => aamApmAction("aam", "off");
  $("aaApmSet").onclick = () => aamApmAction("apm", $("aaApm").value.trim());
  $("aaApmOff").onclick = () => aamApmAction("apm", "off");
  $("aaGet").onclick = aamApmQuery;
  $("seSave").onclick = saveSettings;
  $("bannerClose").onclick = () => {
    const last = state.alarms[state.alarms.length - 1];
    if (last) localStorage.setItem("cdifnos.alarmSeen", last.t * 1000);
    updateBanner();
  };
  $("alarmIndicator").onclick = openAlarms;
  $("themeFile").addEventListener("change", () => {
    const f = $("themeFile").files[0];
    $("themeFile").value = "";
    if (f) importThemeFile(f);
  });
  window.addEventListener("message", (e) => {
    const msg = e.data;
    if (!msg || typeof msg !== "object") {
      return;
    }
    if (msg.type === "cdifnos-open-option") {
      $("dlgOption").showModal();
    } else if (msg.type === "cdifnos-close-option") {
      $("dlgOption").close();
    } else if (msg.type === "cdifnos-option-saved") {
      const f = $("grFrame");
      if (f && f.contentWindow) {
        f.contentWindow.postMessage({ type: "cdifnos-prefs-changed" }, "*");
      }
    }
  });
  document.addEventListener("keydown", (e) => {
    if (e.key === "F5") {
      e.preventDefault();
      refresh();
    } else if (e.key === "F6") {
      e.preventDefault();
      rescanNow();
    }
  });
}

async function init() {
  setLang(localStorage.getItem("cdifnos.lang") || LANG);
  applyZoom();
  applyFont();
  renderAttrHead();
  applyStaticTexts();
  initControls();
  await Theme.loadList();
  await Theme.apply(Theme.current());
  setGeometryMode();
  applyThemeImages();
  await loadSettings();
  buildMenus();
  refresh();
  loadAlarms();
  setInterval(refresh, 3000);
  setInterval(loadAlarms, 15000);
}

init();
