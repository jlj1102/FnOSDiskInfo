"use strict";

const $ = (id) => document.getElementById(id);

const state = {
  disks: [],
  id: localStorage.getItem("cdifnos.id") || null,
  attrs: [],
  updatedAt: null,
  error: null
};

async function api(path, opts) {
  const r = await fetch(path, opts);
  const body = await r.json().catch(() => ({}));
  if (!r.ok) throw new Error(body.error || r.statusText);
  return body;
}

function fmtCapacity(bytes) {
  if (!bytes) return "—";
  const gb = bytes / 1e9;
  return gb >= 1000 ? (gb / 1000).toFixed(2) + " TB" : gb.toFixed(1) + " GB";
}

function healthClass(h) {
  return ["good", "caution", "bad"].includes(h) ? h : "unknown";
}

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
    tm.textContent = d.temperature == null ? "—" : d.temperature + "°C";
    b.append(st, tm);
    b.addEventListener("click", () => {
      state.id = d.id;
      localStorage.setItem("cdifnos.id", d.id);
      refresh();
    });
    box.append(b);
  }
}

function metaRow(label, value) {
  const tr = document.createElement("tr");
  const th = document.createElement("th");
  th.textContent = label;
  const td = document.createElement("td");
  td.textContent = value;
  tr.append(th, td);
  return tr;
}

function renderHead(d) {
  const meta = $("meta");
  const health = $("health");
  const temp = $("temp");
  meta.textContent = "";
  if (!d) {
    $("headline").textContent = state.error ? state.error : t("no_disks");
    health.textContent = "—";
    health.className = "box health unknown";
    temp.textContent = "—";
    return;
  }
  $("headline").textContent = (d.model || d.device) + " : " + fmtCapacity(d.capacity_bytes);
  health.textContent = t(healthClass(d.health));
  health.className = "box health " + healthClass(d.health);
  temp.textContent = d.temperature == null ? "—" : d.temperature + "°C";
  meta.append(
    metaRow(t("firmware"), d.firmware || "—"),
    metaRow(t("serial"), d.serial || "—"),
    metaRow(t("interface"), d.protocol || "—"),
    metaRow(t("power_on"), d.power_on_hours == null ? "—" : d.power_on_hours.toLocaleString() + " " + t("hours")),
    metaRow(t("rotation"), d.rotation_rate ? d.rotation_rate + " " + t("rpm") : t("ssd")),
    metaRow(t("device"), d.device)
  );
  if (d.error) meta.append(metaRow(t("smart"), d.error));
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
      String(a.id),
      attrName(a.id, a.name),
      a.current,
      a.worst,
      a.threshold,
      a.raw,
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

async function refresh() {
  try {
    const data = await api("/api/disks");
    state.disks = data.disks || [];
    state.updatedAt = data.updated_at;
    state.error = data.error || null;
    if (!state.disks.some((d) => d.id === state.id)) {
      state.id = state.disks.length ? state.disks[0].id : null;
    }
    renderTabs();
    const cur = state.disks.find((d) => d.id === state.id) || null;
    renderHead(cur);
    state.attrs = [];
    if (cur && !cur.error) {
      try {
        const s = await api("/api/disks/" + encodeURIComponent(cur.id) + "/smart");
        state.attrs = s.attributes || [];
      } catch (e) {
        // keep empty attribute list
      }
    }
    renderAttrs();
    renderStatus();
  } catch (e) {
    state.error = e.message;
    renderHead(null);
    renderStatus();
  }
}

function initTheme() {
  const sel = $("theme");
  for (const v of ["classic", "dark", "follow"]) {
    const o = document.createElement("option");
    o.value = v;
    o.textContent = t("theme_" + v);
    sel.append(o);
  }
  const saved = localStorage.getItem("cdifnos.theme") || "classic";
  sel.value = saved;
  document.documentElement.dataset.theme = saved;
  sel.addEventListener("change", () => {
    document.documentElement.dataset.theme = sel.value;
    localStorage.setItem("cdifnos.theme", sel.value);
  });
}

function init() {
  renderAttrHead();
  initTheme();
  $("rescan").textContent = t("rescan");
  $("rescan").addEventListener("click", async () => {
    try {
      await api("/api/disks/rescan", { method: "POST" });
    } catch (e) {
      // next poll will surface the error
    }
    setTimeout(refresh, 800);
  });
  refresh();
  setInterval(refresh, 3000);
}

init();
