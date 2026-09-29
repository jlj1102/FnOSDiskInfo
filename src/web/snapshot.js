"use strict";

// "Save image": renders the current view into a canvas and downloads a PNG.
// Generic DOM painter: background art + backgrounds/borders for container
// elements, then images, then text on top (so overlay text stays readable).
// No dependencies.
async function snapshotMain() {
  const root = document.querySelector(".cdi");
  if (!root) return;
  const rect = root.getBoundingClientRect();
  const dpr = Math.min(2, window.devicePixelRatio || 1);
  const canvas = document.createElement("canvas");
  canvas.width = Math.round(rect.width * dpr);
  canvas.height = Math.round(rect.height * dpr);
  const ctx = canvas.getContext("2d");
  ctx.scale(dpr, dpr);

  const css = getComputedStyle(document.documentElement);
  const bg = css.getPropertyValue("--cdi-bg").trim() || "#fff";
  ctx.fillStyle = bg;
  ctx.fillRect(0, 0, rect.width, rect.height);

  const visible = (el) => el.getClientRects().length > 0;
  const insideRoot = (el) => root.contains(el);

  // 1. main-area backdrop (theme art, cover-fitted like the CSS)
  const mainEl = document.getElementById("mainArea");
  if (mainEl) {
    const st = getComputedStyle(mainEl);
    const m = /url\(["']?([^"')]+)["']?\)/.exec(st.backgroundImage || "");
    if (m) {
      const img = await loadImageSrc(m[1]);
      if (img) {
        const r = mainEl.getBoundingClientRect();
        const x = r.left - rect.left;
        const y = r.top - rect.top;
        const scale = Math.max(r.width / img.naturalWidth, r.height / img.naturalHeight);
        ctx.save();
        ctx.beginPath();
        ctx.rect(x, y, r.width, r.height);
        ctx.clip();
        ctx.drawImage(img, x, y, img.naturalWidth * scale, img.naturalHeight * scale);
        ctx.restore();
      }
    }
  }

  // 2. containers: background + border
  const boxSelectors = [".menubar", ".menu-top", ".diskbar", ".tab", ".pager", ".workarea", ".box", ".statusline", ".attrs th", ".attrs td", ".geo-cell", ".banner"];
  for (const sel of boxSelectors) {
    for (const el of root.querySelectorAll(sel)) {
      if (!visible(el)) continue;
      const r = el.getBoundingClientRect();
      const st = getComputedStyle(el);
      const x = r.left - rect.left;
      const y = r.top - rect.top;
      const bgc = st.backgroundColor;
      if (bgc && bgc !== "rgba(0, 0, 0, 0)" && bgc !== "transparent") {
        ctx.fillStyle = bgc;
        ctx.fillRect(x, y, r.width, r.height);
      }
      if (st.borderTopWidth !== "0px" && st.borderTopStyle !== "none") {
        ctx.strokeStyle = st.borderTopColor;
        ctx.lineWidth = parseFloat(st.borderTopWidth) || 1;
        ctx.strokeRect(x + 0.5, y + 0.5, r.width - 1, r.height - 1);
      }
    }
  }

  // 3. images (art must sit under the overlay text)
  const imgs = [];
  for (const el of root.querySelectorAll("img")) {
    if (!visible(el)) continue;
    imgs.push(loadImage(el).then((img) => {
      if (!img) return;
      const r = el.getBoundingClientRect();
      const st = getComputedStyle(el);
      img._rect = { x: r.left - rect.left, y: r.top - rect.top, w: r.width, h: r.height };
      img._opacity = st.opacity;
      return img;
    }));
  }
  for (const img of await Promise.all(imgs)) {
    if (!img || !img._rect) continue;
    ctx.globalAlpha = Number(img._opacity) || 1;
    ctx.drawImage(img, img._rect.x, img._rect.y, img._rect.w, img._rect.h);
    ctx.globalAlpha = 1;
  }

  // 4. text leaves
  const clipEl = document.getElementById("attrwrap");
  const clip = clipEl ? clipEl.getBoundingClientRect() : null;
  for (const el of root.querySelectorAll("*")) {
    if (!insideRoot(el) || !visible(el)) continue;
    let text = "";
    for (const node of el.childNodes) {
      if (node.nodeType === Node.TEXT_NODE) {
        text += node.nodeValue;
      }
    }
    text = text.replace(/\s+/g, " ").trim();
    if (!text) continue;
    const r = el.getBoundingClientRect();
    if (clip && clipEl.contains(el) && (r.bottom < clip.top || r.top > clip.bottom)) {
      continue; // scrolled out of the attribute table
    }
    const st = getComputedStyle(el);
    ctx.fillStyle = st.color;
    ctx.font = st.font || `${st.fontSize} ${st.fontFamily}`;
    ctx.textBaseline = "middle";
    const x = r.left - rect.left;
    const y = r.top - rect.top;
    const padL = parseFloat(st.paddingLeft) || 0;
    const padR = parseFloat(st.paddingRight) || 0;
    if (st.textAlign === "right") {
      ctx.textAlign = "right";
      ctx.fillText(text, x + r.width - padR, y + r.height / 2);
    } else if (st.textAlign === "center") {
      ctx.textAlign = "center";
      ctx.fillText(text, x + r.width / 2, y + r.height / 2);
    } else {
      ctx.textAlign = "left";
      ctx.fillText(text, x + padL, y + r.height / 2);
    }
  }

  canvas.toBlob((blob) => {
    if (!blob) return;
    const d = currentDisk();
    const name = "cdifnos-" + ((d && (d.model || d.device)) || "disk").replace(/[^\w.-]+/g, "_") +
      "-" + new Date().toISOString().slice(0, 10) + ".png";
    const a = document.createElement("a");
    a.href = URL.createObjectURL(blob);
    a.download = name;
    document.body.append(a);
    a.click();
    setTimeout(() => {
      URL.revokeObjectURL(a.href);
      a.remove();
    }, 1000);
  }, "image/png");
}

function loadImageSrc(src) {
  return new Promise((resolve) => {
    if (!src) {
      resolve(null);
      return;
    }
    const img = new Image();
    img.onload = () => resolve(img);
    img.onerror = () => resolve(null);
    img.src = src;
  });
}

function loadImage(el) {
  return loadImageSrc(el.src);
}
