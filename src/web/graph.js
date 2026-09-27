"use strict";

const GRAPH_COLORS = ["#e05050", "#206ec8", "#2e9e4f", "#c98a00", "#8e44ad", "#00838f", "#d35400", "#555555"];

// drawGraph renders line series on a canvas with axes and legend.
// series: [{name, points: [[unixSeconds, value], ...]}]
function drawGraph(canvas, series) {
  const dpr = window.devicePixelRatio || 1;
  const w = canvas.clientWidth || 600;
  const h = canvas.clientHeight || 320;
  canvas.width = Math.round(w * dpr);
  canvas.height = Math.round(h * dpr);
  const ctx = canvas.getContext("2d");
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0);

  const css = getComputedStyle(document.documentElement);
  const text = css.getPropertyValue("--cdi-text").trim() || "#111";
  const border = css.getPropertyValue("--cdi-border").trim() || "#aaa";
  const panel = css.getPropertyValue("--cdi-panel").trim() || "#fff";
  ctx.fillStyle = panel;
  ctx.fillRect(0, 0, w, h);

  const all = [];
  for (const s of series) {
    for (const p of s.points) {
      all.push(p);
    }
  }
  if (!all.length) {
    ctx.fillStyle = text;
    ctx.font = "12px sans-serif";
    ctx.textAlign = "center";
    ctx.textBaseline = "middle";
    ctx.fillText("—", w / 2, h / 2);
    return;
  }

  const padL = 50, padR = 10, padT = 26, padB = 26;
  const plotW = w - padL - padR;
  const plotH = h - padT - padB;

  let xmin = Infinity, xmax = -Infinity, ymin = Infinity, ymax = -Infinity;
  for (const [x, y] of all) {
    if (x < xmin) xmin = x;
    if (x > xmax) xmax = x;
    if (y < ymin) ymin = y;
    if (y > ymax) ymax = y;
  }
  if (xmax === xmin) xmax = xmin + 1;
  if (ymax === ymin) {
    ymax += 1;
    ymin -= 1;
  } else {
    const pad = (ymax - ymin) * 0.08;
    ymin -= pad;
    ymax += pad;
  }
  const X = (x) => padL + ((x - xmin) / (xmax - xmin)) * plotW;
  const Y = (y) => padT + plotH - ((y - ymin) / (ymax - ymin)) * plotH;

  ctx.strokeStyle = border;
  ctx.fillStyle = text;
  ctx.lineWidth = 1;
  ctx.font = "10px sans-serif";

  ctx.textAlign = "right";
  ctx.textBaseline = "middle";
  for (let i = 0; i <= 4; i++) {
    const yv = ymin + ((ymax - ymin) * i) / 4;
    const yy = Y(yv);
    ctx.globalAlpha = 0.35;
    ctx.beginPath();
    ctx.moveTo(padL, yy);
    ctx.lineTo(w - padR, yy);
    ctx.stroke();
    ctx.globalAlpha = 1;
    ctx.fillText(fmtNum(yv), padL - 4, yy);
  }

  ctx.textAlign = "center";
  ctx.textBaseline = "top";
  for (let i = 0; i <= 4; i++) {
    const xv = xmin + ((xmax - xmin) * i) / 4;
    const xx = X(xv);
    ctx.globalAlpha = 0.35;
    ctx.beginPath();
    ctx.moveTo(xx, padT);
    ctx.lineTo(xx, padT + plotH);
    ctx.stroke();
    ctx.globalAlpha = 1;
    ctx.fillText(new Date(xv * 1000).toLocaleDateString(), xx, padT + plotH + 5);
  }

  series.forEach((s, i) => {
    if (!s.points.length) {
      return;
    }
    const color = GRAPH_COLORS[i % GRAPH_COLORS.length];
    s.color = color;
    ctx.strokeStyle = color;
    ctx.lineWidth = 1.5;
    ctx.beginPath();
    s.points.forEach(([x, y], j) => {
      const px = X(x);
      const py = Y(y);
      if (j) {
        ctx.lineTo(px, py);
      } else {
        ctx.moveTo(px, py);
      }
    });
    ctx.stroke();
    if (s.points.length === 1) {
      ctx.fillStyle = color;
      ctx.fillRect(X(s.points[0][0]) - 2, Y(s.points[0][1]) - 2, 4, 4);
    }
  });

  ctx.textAlign = "left";
  ctx.textBaseline = "middle";
  let lx = padL + 2;
  series.forEach((s) => {
    if (!s.color) {
      return;
    }
    ctx.fillStyle = s.color;
    ctx.fillRect(lx, 8, 10, 8);
    ctx.fillStyle = text;
    ctx.fillText(s.name, lx + 14, 12);
    lx += 14 + ctx.measureText(s.name).width + 14;
  });
}

function fmtNum(v) {
  if (Math.abs(v) >= 1000) {
    return Math.round(v).toLocaleString();
  }
  return Math.round(v * 100) / 100;
}
