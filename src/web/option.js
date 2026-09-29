"use strict";

// Graph option window, ported from CrystalDiskInfo's Option.html (MIT):
// 64-slot line color grid + graph background image.

const PREF_KEY = "cdifnos.graph";

function defaultColors() {
  const base = ["#e05050", "#206ec8", "#2e9e4f", "#c98a00", "#8e44ad", "#00838f", "#d35400", "#555555",
    "#c0392b", "#2980b9", "#27ae60", "#f39c12", "#9b59b6", "#16a085", "#e67e22", "#7f8c8d"];
  const out = [];
  for (let i = 0; i < 64; i++) {
    out.push(base[i % base.length]);
  }
  return out;
}

function loadPrefs() {
  const base = { colors: defaultColors(), background: "" };
  try {
    const saved = JSON.parse(localStorage.getItem(PREF_KEY) || "{}");
    return Object.assign(base, saved, { colors: Object.assign(defaultColors(), saved.colors || []) });
  } catch (e) {
    return base;
  }
}

const prefs = loadPrefs();

function setText(id, text) {
  document.getElementById(id).textContent = text;
}

function buildGrid() {
  const table = document.getElementById("colorGrid");
  table.textContent = "";
  for (let r = 0; r < 8; r++) {
    const tr = document.createElement("tr");
    for (let c = 0; c < 8; c++) {
      const idx = r * 8 + c;
      const td = document.createElement("td");
      const input = document.createElement("input");
      input.type = "color";
      input.value = prefs.colors[idx];
      input.dataset.index = String(idx);
      input.title = t("disk") + " " + (idx + 1);
      const label = document.createElement("span");
      label.className = "cellLabel";
      label.textContent = String(idx + 1);
      td.append(input, label);
      tr.append(td);
    }
    table.append(tr);
  }
}

function applyLanguage() {
  if (typeof setLang === "function") {
    setLang(localStorage.getItem("cdifnos.lang") || LANG);
  }
  document.title = t("g_options");
  setText("colorTitle", t("g_color"));
  setText("bgLabel", t("g_background"));
  setText("optSave", t("apply"));
  setText("optClose", t("close"));
  const bgTheme = document.getElementById("bgTheme");
  bgTheme.textContent = t("g_use_theme");
  bgTheme.onclick = async () => {
    const id = localStorage.getItem("cdifnos.theme") || "";
    if (!id || ["classic", "dark", "follow"].includes(id)) {
      return;
    }
    try {
      const m = await (await fetch("/themes/" + encodeURIComponent(id) + "/theme.json")).json();
      const file = m.images && m.images.background;
      if (file) {
        document.getElementById("bgImage").value = "/themes/" + id + "/" + file;
      }
    } catch (e) {
      // ignore
    }
  };
  document.getElementById("bgClear").textContent = t("disable");
}

function save() {
  const inputs = document.querySelectorAll("#colorGrid input[type=color]");
  const colors = prefs.colors.slice();
  for (const input of inputs) {
    colors[Number(input.dataset.index)] = input.value;
  }
  const background = document.getElementById("bgImage").value.trim();
  if (/["']/.test(background)) {
    return;
  }
  prefs.colors = colors;
  prefs.background = background;
  localStorage.setItem(PREF_KEY, JSON.stringify(prefs));
  window.parent.postMessage({ type: "cdifnos-option-saved" }, "*");
}

applyLanguage();
buildGrid();
document.getElementById("bgImage").value = prefs.background;
document.getElementById("optSave").addEventListener("click", save);
document.getElementById("optClose").addEventListener("click", () => {
  window.parent.postMessage({ type: "cdifnos-close-option" }, "*");
});
