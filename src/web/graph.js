"use strict";

// Graph window, ported from CrystalDiskInfo's Graph.html (MIT):
// flot line chart with all-disk toggles, attribute selection, overview strip,
// weekend shading and selection zoom. Data comes from our history API.

const PREF_KEY = "cdifnos.graph";

const FIXED_METRICS = [
  ["temperature", "m_temperature"],
  ["life", "m_life"],
  ["power_on_hours", "m_power_on_hours"],
  ["power_on_count", "m_power_on_count"],
  ["host_reads", "m_host_reads"],
  ["host_writes", "m_host_writes"],
  ["reallocated", "m_reallocated"],
  ["realloc_events", "m_realloc_events"],
  ["pending", "m_pending"],
  ["uncorrectable", "m_uncorrectable"]
];

const POINTS = [100, 200, 300, 400, 500, 600, 700, 800, 900, 1000, 2000, 3000, 4000, 5000, 0];

const TIMEFORMATS = [
  ["%m/%d %H:%M", "MM/DD hh:mm"],
  ["%m/%d", "MM/DD"],
  ["%Y/%m/%d %H:%M", "YYYY/MM/DD hh:mm"],
  ["%Y/%m/%d", "YYYY/MM/DD"],
  ["%d/%m/%Y %H:%M", "DD/MM/YYYY hh:mm"],
  ["%d/%m/%Y", "DD/MM/YYYY"],
  ["%d.%m.%Y %H:%M", "DD.MM.YYYY hh:mm"],
  ["%d.%m.%Y", "DD.MM.YYYY"]
];

let prefs = loadPrefs();
let disks = [];
let enabled = {};
let metric = "temperature";
let currentSeries = [];
let plot = null;
let overview = null;
let internalSelection = false;
let previousPoint = null;

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
  const base = {
    colors: defaultColors(),
    legend: "ne",
    points: 500,
    timeformat: "%Y/%m/%d %H:%M",
    weekend: false,
    background: ""
  };
  try {
    const saved = JSON.parse(localStorage.getItem(PREF_KEY) || "{}");
    return Object.assign(base, saved, { colors: Object.assign(defaultColors(), saved.colors || []) });
  } catch (e) {
    return base;
  }
}

async function api(path) {
  const r = await fetch(path);
  if (!r.ok) {
    throw new Error(r.statusText);
  }
  return r.json();
}

function applyBackground() {
  document.body.style.backgroundImage = prefs.background ? `url("${prefs.background}")` : "";
}

function buildToolbar() {
  document.getElementById("legendLabel").textContent = t("g_legend") + ":";
  document.getElementById("pointsLabel").textContent = t("points") + ":";
  document.getElementById("timeLabel").textContent = t("g_timeformat") + ":";
  document.getElementById("weekendLabel").textContent = t("g_weekend");

  const points = document.getElementById("MaxPoints");
  points.textContent = "";
  for (const n of POINTS) {
    const o = document.createElement("option");
    o.value = String(n);
    o.textContent = n === 0 ? t("all") : String(n);
    points.append(o);
  }
  points.value = String(prefs.points);
  points.onchange = () => {
    prefs.points = Number(points.value);
    savePrefs();
    refresh();
  };

  const tf = document.getElementById("TimeFormat");
  tf.textContent = "";
  for (const [v, label] of TIMEFORMATS) {
    const o = document.createElement("option");
    o.value = v;
    o.textContent = label;
    tf.append(o);
  }
  tf.value = prefs.timeformat;
  tf.onchange = () => {
    prefs.timeformat = tf.value;
    savePrefs();
    redraw();
  };

  const legend = document.getElementById("LegendPosition");
  legend.value = prefs.legend;
  legend.onchange = () => {
    prefs.legend = legend.value;
    savePrefs();
    redraw();
  };

  const weekend = document.getElementById("PaintWeekend");
  weekend.checked = prefs.weekend;
  weekend.onchange = () => {
    prefs.weekend = weekend.checked;
    savePrefs();
    redraw();
  };
}

function buildToggles() {
  const box = document.getElementById("diskToggles");
  box.textContent = "";
  disks.forEach((d, i) => {
    const a = document.createElement("a");
    a.href = "#";
    a.className = enabled[d.id] ? "on" : "";
    a.textContent = d.model || d.device;
    a.title = d.device;
    a.style.color = prefs.colors[i % 64];
    a.addEventListener("click", (e) => {
      e.preventDefault();
      enabled[d.id] = !enabled[d.id];
      a.classList.toggle("on", enabled[d.id]);
      refresh();
    });
    box.append(a);
  });
}

function addOption(parent, value, label) {
  const o = document.createElement("option");
  o.value = value;
  o.textContent = label;
  parent.append(o);
}

async function buildSelect() {
  const sel = document.getElementById("SelectAttributeId");
  sel.textContent = "";
  for (const [m, key] of FIXED_METRICS) {
    addOption(sel, m, t(key));
  }
  const d = disks[0];
  if (d && !d.error) {
    try {
      const s = await api("/api/disks/" + encodeURIComponent(d.id) + "/smart");
      const group = document.createElement("optgroup");
      group.label = t("col_attr");
      for (const a of s.attributes || []) {
        const hex = a.id.toString(16).toUpperCase().padStart(2, "0");
        addOption(group, "attr-" + hex, hex + " " + attrName(a.id, a.name, d.is_ssd));
      }
      sel.append(group);
    } catch (e) {
      // no attributes
    }
  }
  sel.value = metric;
  sel.onchange = () => {
    metric = sel.value;
    refresh();
  };
}

async function refresh() {
  const series = [];
  for (let i = 0; i < disks.length; i++) {
    const d = disks[i];
    if (!enabled[d.id]) {
      continue;
    }
    const points = prefs.points === 0 ? "all" : String(prefs.points);
    try {
      const res = await api("/api/disks/" + encodeURIComponent(d.id) + "/history?metric=" +
        encodeURIComponent(metric) + "&points=" + points);
      const data = (res.points || []).map(([ts, v]) => [ts * 1000, v]);
      if (data.length) {
        series.push({ label: d.model || d.device, data, color: prefs.colors[i % 64] });
      }
    } catch (e) {
      // skip failed disk
    }
  }
  currentSeries = series;
  redraw();
}

function baseOptions() {
  return {
    lines: { show: true, lineWidth: 1 },
    points: { show: false },
    xaxis: { mode: "time", timeformat: prefs.timeformat, twelveHourClock: false },
    yaxis: {},
    legend: { show: true, position: prefs.legend, backgroundOpacity: 0.6 },
    grid: {
      hoverable: true,
      clickable: true,
      markings: prefs.weekend ? weekendMarking : []
    },
    colors: prefs.colors
  };
}

function weekendMarking(axes) {
  return weekendAreas(axes.xaxis);
}

// weekends in the visible range (from the original Graph.html)
function weekendAreas(plotarea) {
  const areas = [];
  const d = new Date(plotarea.xmin);
  d.setDate(d.getDate() - ((d.getDay() + 1) % 7));
  d.setSeconds(0);
  d.setMinutes(0);
  d.setHours(0);
  let i = d.getTime() - d.getTimezoneOffset() * 60 * 1000;
  do {
    areas.push({ x1: i, x2: i + 2 * 24 * 60 * 60 * 1000, color: "rgba(0,0,0,0.06)" });
    i += 7 * 24 * 60 * 60 * 1000;
  } while (i < plotarea.xmax);
  return areas;
}

function redraw() {
  const options = baseOptions();
  const overViewOptions = $.extend(true, {}, options, {
    legend: { show: false },
    yaxis: { ticks: 2 },
    xaxis: { ticks: 4 },
    grid: { hoverable: false, markings: options.grid.markings }
  });
  plot = $.plot($("#placeholder"), currentSeries, options);
  overview = $.plot($("#overview"), currentSeries, overViewOptions);
  bindSelection(options);
}

function bindSelection(options) {
  $("#placeholder").unbind("selected").bind("selected", function (event, area) {
    plot = $.plot($("#placeholder"), currentSeries,
      $.extend(true, {}, options, {
        xaxis: { min: area.x1, max: area.x2 },
        yaxis: { min: area.y1, max: area.y2 }
      }));
    if (internalSelection) {
      return;
    }
    internalSelection = true;
    overview.setSelection(area);
    internalSelection = false;
  });
  $("#overview").unbind("selected").bind("selected", function (event, area) {
    if (internalSelection) {
      return;
    }
    internalSelection = true;
    plot.setSelection(area);
    internalSelection = false;
  });
}

function showTooltip(x, y, contents) {
  const str = "" + contents;
  $("<div id='tooltip'>" + contents + "</div>").css({
    top: y - 30,
    left: Math.max(0, x - str.length * 5)
  }).appendTo("body").fadeIn(150);
}

function bindHover() {
  $("#placeholder").bind("plothover", function (event, pos, item) {
    if (item) {
      if (previousPoint !== item.datapoint) {
        previousPoint = item.datapoint;
        $("#tooltip").remove();
        showTooltip(item.pageX, item.pageY, Math.round(item.datapoint[1] * 100) / 100);
      }
    } else {
      $("#tooltip").remove();
      previousPoint = null;
    }
  });
}

function bindToolbar() {
  document.getElementById("AllOn").addEventListener("click", (e) => {
    e.preventDefault();
    disks.forEach((d) => {
      enabled[d.id] = true;
    });
    buildToggles();
    refresh();
  });
  document.getElementById("AllOff").addEventListener("click", (e) => {
    e.preventDefault();
    disks.forEach((d) => {
      enabled[d.id] = false;
    });
    buildToggles();
    refresh();
  });
  document.getElementById("Refresh").addEventListener("click", (e) => {
    e.preventDefault();
    refresh();
  });
  document.getElementById("Customize").addEventListener("click", (e) => {
    e.preventDefault();
    window.parent.postMessage({ type: "cdifnos-open-option" }, "*");
  });
}

async function init() {
  if (typeof setLang === "function") {
    setLang(localStorage.getItem("cdifnos.lang") || LANG);
  }
  applyBackground();
  buildToolbar();
  bindToolbar();
  bindHover();
  const wanted = new URLSearchParams(location.search).get("disk");
  try {
    const resp = await api("/api/disks");
    disks = resp.disks || [];
  } catch (e) {
    disks = [];
  }
  disks.forEach((d) => {
    enabled[d.id] = true;
  });
  if (wanted && disks.some((d) => d.id === wanted)) {
    disks.forEach((d) => {
      enabled[d.id] = d.id === wanted;
    });
  }
  buildToggles();
  await buildSelect();
  await refresh();
  window.addEventListener("resize", () => redraw());
  window.addEventListener("message", (e) => {
    if (e.data && e.data.type === "cdifnos-prefs-changed") {
      prefs = loadPrefs();
      applyBackground();
      buildToolbar();
      buildToggles();
      refresh();
    }
  });
}

init();
