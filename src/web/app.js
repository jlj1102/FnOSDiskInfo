"use strict";

const $ = (id) => document.getElementById(id);

const THEMES = ["classic", "dark", "follow"];
const LANGS = [["en", "English"], ["zh-CN", "简体中文"], ["zh-TW", "繁體中文"], ["ja", "日本語"]];
const METRICS = ["temperature", "life", "power_on_hours", "power_on_count", "reallocated",
  "realloc_events", "pending", "uncorrectable", "host_reads", "host_writes"];

const state = {
  disks: [],
  id: localStorage.getItem("cdifnos.id") || null,
  attrs: [],
  updatedAt: null,
  error: null,
  settings: null,
  alarms: [],
  ui: {
    unit: localStorage.getItem("cdifnos.unit") || "C",
    raw: localStorage.getItem("cdifnos.raw") || "hex",
    hideSerial: localStorage.getItem("cdifnos.hideSerial") === "1"
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

function fmtCapacity(bytes) {
  if (!bytes) return "—";
  const gb = bytes / 1e9;
  return gb >= 1000 ? (gb / 1000).toFixed(2) + " TB" : gb.toFixed(1) + " GB";
}

function fmtTemp(c) {
  if (c == null) return "—";
  return state.ui.unit === "F" ? Math.round((c * 9) / 5 + 32) + "°F" : c + "°C";
}

function fmtRaw(a) {
  const v = a.raw_value;
  if (v === undefined || v === null) return a.raw || "";
  switch (state.ui.raw) {
    case "dec": return String(v);
    case "byte2": return String(v & 0xffff);
    case "byte1": return String(v & 0xff);
    default: return "0x" + v.toString(16).toUpperCase();
  }
}

function healthClass(h) {
  return ["good", "caution", "bad"].includes(h) ? h : "unknown";
}

function fmtNum(n) {
  return n == null ? "—" : Number(n).toLocaleString();
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
  const box = $("tabs");
  box.textContent = "";
  for (const d of state.disks) {
    const cls = d.error ? "unknown" : healthClass(d.health);
    const b = document.createElement("button");
    b.type = "button";
    b.className = "tab " + cls + (d.id === state.id ? " active" : "");
    const st = document.createElement("span");
    st.className = "st";
    st.textContent = d.error ? t("unknown") : t(cls);
    const tm = document.createElement("span");
    tm.className = "tm";
    tm.textContent = fmtTemp(d.temperature);
    b.append(st, tm);
    b.addEventListener("click", () => {
      state.id = d.id;
      localStorage.setItem("cdifnos.id", d.id);
      refresh();
    });
    box.append(b);
  }
}

function metaRow(label, value, cls) {
  const tr = document.createElement("tr");
  const th = document.createElement("th");
  th.textContent = label;
  const td = document.createElement("td");
  td.textContent = value;
  if (cls) td.className = cls;
  tr.append(th, td);
  return tr;
}

function renderHead() {
  const meta = $("meta");
  const health = $("health");
  const temp = $("temp");
  meta.textContent = "";
  const d = currentDisk();
  if (!d) {
    $("headline").textContent = state.error ? state.error : t("no_disks");
    health.textContent = "—";
    health.className = "box health unknown";
    temp.textContent = "—";
    return;
  }
  $("headline").textContent = (d.model || d.device) + " : " + fmtCapacity(d.capacity_bytes);
  const cls = healthClass(d.health);
  health.textContent = t(cls) + (d.life != null ? " " + d.life + "%" : "");
  health.className = "box health " + cls;
  temp.textContent = fmtTemp(d.temperature);

  const serial = d.serial ? (state.ui.hideSerial ? "********" : d.serial) : "—";
  meta.append(
    metaRow(t("firmware"), d.firmware || "—"),
    metaRow(t("serial"), serial),
    metaRow(t("interface"), d.protocol || "—"),
    metaRow(t("power_on"), d.power_on_hours == null ? "—" : fmtNum(d.power_on_hours) + " " + t("hours")),
    metaRow(t("power_on_count"), d.power_on_count == null ? "—" : fmtNum(d.power_on_count)),
    metaRow(t("rotation"), d.rotation_rate ? d.rotation_rate + " " + t("rpm") : t("ssd")),
    metaRow(t("device"), d.device)
  );
  if (d.self_test) {
    meta.append(metaRow(t("last_test"), d.self_test.type + ": " + d.self_test.status));
  }
  if (d.status_reasons && d.status_reasons.length) {
    meta.append(metaRow(t("status_reasons"), d.status_reasons.join("; "), "reasons"));
  }
  if (d.error) {
    meta.append(metaRow(t("smart"), d.error, "reasons"));
  }
}

const ATTR_COLS = ["col_id", "col_attr", "col_cur", "col_worst", "col_thr", "col_raw", "col_status"];

function renderAttrHead() {
  const tr = $("attrhead");
  tr.textContent = "";
  for (const c of ATTR_COLS) {
    const th = document.createElement("th");
    th.textContent = t(c);
    tr.append(th);
  }
}

function renderAttrs() {
  const tb = $("attrs");
  tb.textContent = "";
  if (!state.attrs.length) {
    const tr = document.createElement("tr");
    const td = document.createElement("td");
    td.colSpan = ATTR_COLS.length;
    td.className = "empty";
    td.textContent = t("no_attrs");
    tr.append(td);
    tb.append(tr);
    return;
  }
  for (const a of state.attrs) {
    const tr = document.createElement("tr");
    if (a.status === "bad") tr.className = "bad";
    const cells = [
      a.id.toString(16).toUpperCase().padStart(2, "0"),
      attrName(a.id, a.name),
      a.current,
      a.worst,
      a.threshold,
      fmtRaw(a),
      t(a.status === "bad" ? "bad" : "ok")
    ];
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
  const dot = $("alarmDot");
  if (last && last.t * 1000 > seen && last.to !== "good") {
    banner.classList.remove("hidden");
    $("bannerText").textContent = new Date(last.t * 1000).toLocaleString() + " — " + last.message;
    dot.classList.remove("hidden");
  } else {
    banner.classList.add("hidden");
    if (last && last.t * 1000 > seen) {
      dot.classList.remove("hidden");
    } else {
      dot.classList.add("hidden");
    }
  }
}

// ---------- data refresh ----------

async function refresh() {
  try {
    const data = await api("/api/disks");
    state.disks = data.disks || [];
    state.updatedAt = data.updated_at;
    state.error = data.error || null;
    if (state.id && !state.disks.some((d) => d.id === state.id)) {
      state.id = state.disks.length ? state.disks[0].id : null;
    }
    if (!state.id && state.disks.length) {
      state.id = state.disks[0].id;
    }
    renderTabs();
    renderHead();
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
}

async function loadAlarms() {
  try {
    state.alarms = (await api("/api/alarms")).alarms || [];
  } catch (e) {
    state.alarms = [];
  }
  updateBanner();
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
  const url = "/api/disks/" + encodeURIComponent(d.id) + "/" + action;
  const body = action === "self-test" ? { type: $("stType").value } : {};
  const res = await runRequest(url, body);
  $("stOut").textContent = res.output || res.error || JSON.stringify(res);
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

async function aamApmAction(kind, value) {
  const d = currentDisk();
  if (!d) return;
  $("aaOut").textContent = "...";
  const res = await runRequest("/api/disks/" + encodeURIComponent(d.id) + "/aam-apm", { kind, value });
  $("aaOut").textContent = res.output || res.error || JSON.stringify(res);
  refresh();
}

// ---------- dialogs: graph ----------

function openGraph() {
  buildGraphDisks();
  $("dlgGraph").showModal();
  drawGraphNow();
}

function buildGraphDisks() {
  const box = $("grDisks");
  box.textContent = "";
  for (const d of state.disks) {
    const label = document.createElement("label");
    const cb = document.createElement("input");
    cb.type = "checkbox";
    cb.value = d.id;
    cb.checked = d.id === state.id;
    const span = document.createElement("span");
    span.textContent = d.model || d.device;
    label.append(cb, span);
    box.append(label);
  }
}

async function drawGraphNow() {
  const metric = $("grMetric").value;
  const points = $("grPoints").value;
  const ids = [...$("grDisks").querySelectorAll("input:checked")].map((c) => c.value);
  const series = [];
  for (const id of ids) {
    const d = state.disks.find((x) => x.id === id);
    try {
      const res = await api("/api/disks/" + encodeURIComponent(id) + "/history?metric=" + metric + "&points=" + points);
      series.push({ name: d ? (d.model || d.device) : id, points: res.points || [] });
    } catch (e) {
      // skip failed series
    }
  }
  drawGraph($("grCanvas"), series);
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
  const def = s.built_in_defaults || {};
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

  const secClient = document.createElement("section");
  const h4c = document.createElement("h4");
  h4c.textContent = t("client");
  secClient.append(h4c);

  const rowUnit = document.createElement("div");
  rowUnit.className = "client-row";
  const lUnit = document.createElement("label");
  lUnit.textContent = t("unit");
  const selUnit = document.createElement("select");
  selUnit.id = "seUnit";
  for (const [v, label] of [["C", t("celsius")], ["F", t("fahrenheit")]]) {
    const o = document.createElement("option");
    o.value = v;
    o.textContent = label;
    selUnit.append(o);
  }
  selUnit.value = state.ui.unit;
  rowUnit.append(lUnit, selUnit);

  const rowRaw = document.createElement("div");
  rowRaw.className = "client-row";
  const lRaw = document.createElement("label");
  lRaw.textContent = t("raw_format");
  const selRaw = document.createElement("select");
  selRaw.id = "seRaw";
  for (const v of ["hex", "dec", "byte2", "byte1"]) {
    const o = document.createElement("option");
    o.value = v;
    o.textContent = t(v);
    selRaw.append(o);
  }
  selRaw.value = state.ui.raw;
  rowRaw.append(lRaw, selRaw);

  const rowHide = document.createElement("div");
  rowHide.className = "client-row";
  const lHide = document.createElement("label");
  lHide.textContent = t("hide_serial");
  const cbHide = document.createElement("input");
  cbHide.type = "checkbox";
  cbHide.id = "seHide";
  cbHide.checked = state.ui.hideSerial;
  rowHide.append(lHide, cbHide);

  secClient.append(rowUnit, rowRaw, rowHide);
  wrap.append(secClient);
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
  state.ui.unit = $("seUnit").value;
  state.ui.raw = $("seRaw").value;
  state.ui.hideSerial = $("seHide").checked;
  localStorage.setItem("cdifnos.unit", state.ui.unit);
  localStorage.setItem("cdifnos.raw", state.ui.raw);
  localStorage.setItem("cdifnos.hideSerial", state.ui.hideSerial ? "1" : "0");
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
  refresh();
}

async function loadSettings() {
  try {
    state.settings = await api("/api/settings");
  } catch (e) {
    state.settings = null;
  }
}

// ---------- init ----------

function applyStaticTexts() {
  $("rescan").textContent = t("rescan");
  $("btnSelftest").textContent = t("selftest");
  $("btnAamapm").textContent = t("aam_apm");
  $("btnAlarmsLabel").textContent = t("alarms");
  $("btnGraph").textContent = t("graph");
  $("btnReport").textContent = t("report");
  $("btnRaw").textContent = t("raw_json");
  $("btnSettings").textContent = t("settings");
  $("stTitle").textContent = t("selftest");
  $("stStart").textContent = t("start");
  $("stAbort").textContent = t("abort");
  $("aaTitle").textContent = t("aam_apm");
  $("aaAamSet").textContent = t("set");
  $("aaAamOff").textContent = t("off");
  $("aaApmSet").textContent = t("set");
  $("aaApmOff").textContent = t("off");
  $("grTitle").textContent = t("graph");
  $("grDraw").textContent = t("draw");
  $("seTitle").textContent = t("settings");
  $("seSave").textContent = t("apply");
  $("alTitle").textContent = t("alarms");
  for (const b of document.querySelectorAll(".close")) {
    b.textContent = t("close");
    b.onclick = () => b.closest("dialog").close();
  }
}

function initControls() {
  const themeSel = $("theme");
  for (const v of THEMES) {
    const o = document.createElement("option");
    o.value = v;
    o.textContent = t("theme_" + v);
    themeSel.append(o);
  }
  const savedTheme = localStorage.getItem("cdifnos.theme") || "classic";
  themeSel.value = savedTheme;
  document.documentElement.dataset.theme = savedTheme;
  themeSel.onchange = () => {
    document.documentElement.dataset.theme = themeSel.value;
    localStorage.setItem("cdifnos.theme", themeSel.value);
  };

  const langSel = $("lang");
  const savedLang = localStorage.getItem("cdifnos.lang") || LANG;
  for (const [v, label] of LANGS) {
    const o = document.createElement("option");
    o.value = v;
    o.textContent = label;
    langSel.append(o);
  }
  langSel.value = savedLang;
  langSel.onchange = () => {
    localStorage.setItem("cdifnos.lang", langSel.value);
    location.reload();
  };

  const stType = $("stType");
  for (const v of ["short", "long", "conveyance"]) {
    const o = document.createElement("option");
    o.value = v;
    o.textContent = t(v);
    stType.append(o);
  }

  const grMetric = $("grMetric");
  for (const m of METRICS) {
    const o = document.createElement("option");
    o.value = m;
    o.textContent = t("m_" + m);
    grMetric.append(o);
  }
  const grPoints = $("grPoints");
  for (const p of [["100", "100"], ["500", "500"], ["2000", "2000"], ["all", t("all")]]) {
    const o = document.createElement("option");
    o.value = p[0];
    o.textContent = p[1];
    grPoints.append(o);
  }
  grPoints.value = "500";

  $("rescan").onclick = async () => {
    try {
      await api("/api/disks/rescan", { method: "POST" });
    } catch (e) {
      // next poll surfaces errors
    }
    setTimeout(refresh, 800);
  };
  $("btnSelftest").onclick = openSelftest;
  $("stStart").onclick = () => selftestAction("self-test");
  $("stAbort").onclick = () => selftestAction("abort-test");

  $("btnAamapm").onclick = openAamApm;
  $("aaAamSet").onclick = () => aamApmAction("aam", $("aaAam").value.trim());
  $("aaAamOff").onclick = () => aamApmAction("aam", "off");
  $("aaApmSet").onclick = () => aamApmAction("apm", $("aaApm").value.trim());
  $("aaApmOff").onclick = () => aamApmAction("apm", "off");

  $("btnAlarms").onclick = openAlarms;
  $("btnGraph").onclick = openGraph;
  $("grDraw").onclick = drawGraphNow;
  window.addEventListener("resize", () => {
    if ($("dlgGraph").open) drawGraphNow();
  });
  $("btnReport").onclick = () => {
    const d = currentDisk();
    if (d) location.href = "/api/disks/" + encodeURIComponent(d.id) + "/report.txt";
  };
  $("btnRaw").onclick = () => {
    const d = currentDisk();
    if (d) location.href = "/api/disks/" + encodeURIComponent(d.id) + "/raw";
  };
  $("btnSettings").onclick = openSettings;
  $("seSave").onclick = saveSettings;
  $("bannerClose").onclick = () => {
    const last = state.alarms[state.alarms.length - 1];
    if (last) localStorage.setItem("cdifnos.alarmSeen", last.t * 1000);
    updateBanner();
  };
}

function init() {
  setLang(localStorage.getItem("cdifnos.lang") || LANG);
  renderAttrHead();
  applyStaticTexts();
  initControls();
  loadSettings();
  refresh();
  loadAlarms();
  setInterval(refresh, 3000);
  setInterval(loadAlarms, 15000);
}

init();
