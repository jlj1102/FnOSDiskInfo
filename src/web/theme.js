"use strict";

// Theme management: built-in themes use data-theme CSS, imported themes
// inject vars/CSS/images from /themes/<id>/.
const Theme = (() => {
  let list = [];
  let images = {};

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
      return;
    }
    el.dataset.theme = "custom";
    try {
      const m = await (await fetch(`/themes/${id}/theme.json`)).json();
      for (const [k, v] of Object.entries(m.vars || {})) {
        if (k.startsWith("--cdi-")) {
          el.style.setProperty(k, v);
        }
      }
      if (m.css) {
        const css = await (await fetch(`/themes/${id}/${m.css}`)).text();
        const st = document.createElement("style");
        st.id = "theme-css";
        st.textContent = css;
        document.head.append(st);
      }
      for (const [slot, file] of Object.entries(m.images || {})) {
        images[slot] = `/themes/${id}/${file}`;
      }
    } catch (e) {
      // keep default look
    }
  }

  async function importFile(file) {
    const r = await fetch("/api/themes/import", { method: "POST", body: file });
    const body = await r.json().catch(() => ({}));
    if (!r.ok) {
      throw new Error(body.error || r.statusText);
    }
    await loadList();
    return body;
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
    get list() {
      return list;
    },
    get images() {
      return images;
    }
  };
})();
