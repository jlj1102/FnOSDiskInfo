"use strict";

// Theme management: built-in themes use data-theme CSS, imported themes
// inject vars/CSS/images from /themes/<id>/.
const Theme = (() => {
  let list = [];
  let images = {};
  let frames = {};
  let currentId = "";

  function current() {
    return localStorage.getItem("cdifnos.theme") || "classic";
  }

  function isBuiltin(id) {
    return ["classic", "dark", "follow"].includes(id);
  }

  async function loadList() {
    try {
      list = (await api("/api/themes")).themes || [];
    } catch (e) {
      list = [];
    }
    return list;
  }

  async function apply(id) {
    localStorage.setItem("cdifnos.theme", id);
    images = {};
    frames = {};
    currentId = id;
    const el = document.documentElement;
    const oldCss = document.getElementById("theme-css");
    if (oldCss) {
      oldCss.remove();
    }
    for (const k of [...el.style]) {
      if (k.startsWith("--cdi-")) {
        el.style.removeProperty(k);
      }
    }
    const info = list.find((t) => t.id === id);
    if (!info || info.builtin || isBuiltin(id)) {
      el.dataset.theme = id;
      applyChrome({});
      return;
    }
    el.dataset.theme = "custom";
    try {
      const stamp = Date.now();
      const m = await (await fetch(`/themes/${id}/theme.json?t=${stamp}`)).json();
      for (const [k, v] of Object.entries(m.vars || {})) {
        if (k.startsWith("--cdi-")) {
          el.style.setProperty(k, v);
        }
      }
      const css = m.css || "";
      if (css) {
        const st = document.createElement("style");
        st.id = "theme-css";
        st.textContent = await (await fetch(`/themes/${id}/${css}?t=${stamp}`)).text();
        document.head.append(st);
      }
      for (const [slot, file] of Object.entries(m.images || {})) {
        images[slot] = `/themes/${id}/${file}`;
      }
      frames = m.frame_count || {};
      // glass panel: semi-transparent panels when the theme carries an alpha.
      // Readability floor: CDI blends at 128/255 on small windows; a full-page
      // web UI needs a bit more opacity for text to stay legible.
      const vars = m.vars || {};
      if (vars["--cdi-panel"] && vars["--cdi-panel-alpha"]) {
        const alpha = Math.max(0.65, Number(vars["--cdi-panel-alpha"]) || 1);
        el.style.setProperty("--cdi-panel", hexToRgba(vars["--cdi-panel"], alpha) || vars["--cdi-panel"]);
      }
      applyChrome(images);
    } catch (e) {
      // keep default look
    }
  }

  function hexToRgba(hex, alpha) {
    const m = /^#([0-9a-f]{6})$/i.exec(hex || "");
    if (!m || isNaN(alpha)) {
      return "";
    }
    const n = parseInt(m[1], 16);
    return `rgba(${(n >> 16) & 255}, ${(n >> 8) & 255}, ${n & 255}, ${Math.max(0, Math.min(1, alpha))})`;
  }

  function applyChrome(imgs) {
    if (imgs.background) {
      document.body.style.backgroundImage = `url("${imgs.background}")`;
      // CDI draws the backdrop bitmap 1:1 from the top-left (pattern brush),
      // it is never stretched to fill the window.
      document.body.style.backgroundSize = "auto";
      document.body.style.backgroundPosition = "left top";
      document.body.style.backgroundRepeat = "no-repeat";
      document.body.classList.add("themed-bg");
    } else {
      document.body.style.backgroundImage = "";
      document.body.style.backgroundRepeat = "";
      document.body.classList.remove("themed-bg");
    }
  }

  // frameUrl returns the URL of frame n of a split sprite slot, or "" when the
  // theme has no multi-frame art for that slot.
  function frameUrl(slot, n) {
    const count = frames[slot] || 0;
    if (!count || !currentId) {
      return "";
    }
    if (n < 0 || n >= count) {
      n = 0;
    }
    return `/themes/${currentId}/${slot}.${n}.png`;
  }

  async function importFile(file) {
    const r = await fetch("/api/themes/import?name=" + encodeURIComponent(file.name || ""), { method: "POST", body: file });
    const body = await r.json().catch(() => ({}));
    if (!r.ok) {
      throw new Error(body.error || r.statusText);
    }
    await loadList();
    return { themes: body.themes || [], skipped: body.skipped || [] };
  }

  async function remove(id) {
    const r = await fetch("/api/themes/" + encodeURIComponent(id), { method: "DELETE" });
    if (!r.ok) {
      const body = await r.json().catch(() => ({}));
      throw new Error(body.error || r.statusText);
    }
    await loadList();
  }

  return {
    loadList,
    apply,
    current,
    importFile,
    remove,
    frameUrl,
    get list() {
      return list;
    },
    get images() {
      return images;
    }
  };
})();
