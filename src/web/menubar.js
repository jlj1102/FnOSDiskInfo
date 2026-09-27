"use strict";

// Classic dropdown menu bar.
// menus: [{label, items: [entry...]}]
// entry: {label, action, checked?: bool|fn, disabled?, separator?, items?: [...]}
const Menubar = (() => {
  let root = null;
  let model = [];
  let openIndex = -1;

  function render(el, menus) {
    root = el;
    model = menus;
    paint();
  }

  function refresh() {
    if (root && openIndex === -1) {
      paint();
    }
  }

  function close() {
    if (openIndex !== -1) {
      openIndex = -1;
      paint();
    }
  }

  function toggle(i) {
    openIndex = openIndex === i ? -1 : i;
    paint();
  }

  function open(i) {
    openIndex = i;
    paint();
  }

  function paint() {
    if (!root) {
      return;
    }
    root.textContent = "";
    model.forEach((m, i) => {
      const wrap = document.createElement("div");
      wrap.className = "menu-root" + (openIndex === i ? " open" : "");
      const btn = document.createElement("button");
      btn.type = "button";
      btn.className = "menu-top";
      btn.textContent = m.label;
      btn.addEventListener("click", (e) => {
        e.stopPropagation();
        toggle(i);
      });
      btn.addEventListener("mouseenter", () => {
        if (openIndex !== -1 && openIndex !== i) {
          open(i);
        }
      });
      wrap.append(btn);
      if (openIndex === i) {
        wrap.append(buildList(m.items));
      }
      root.append(wrap);
    });
  }

  function buildList(items) {
    const box = document.createElement("div");
    box.className = "menu-drop";
    box.addEventListener("click", (e) => e.stopPropagation());
    for (const it of items || []) {
      if (!it) {
        continue;
      }
      if (it.separator) {
        const sep = document.createElement("div");
        sep.className = "menu-sep";
        box.append(sep);
        continue;
      }
      const checked = typeof it.checked === "function" ? it.checked() : it.checked;
      const entry = document.createElement("div");
      entry.className = "menu-entry" +
        (checked ? " checked" : "") +
        (it.disabled ? " disabled" : "") +
        (it.items && it.items.length ? " has-sub" : "");
      const label = document.createElement("span");
      label.textContent = it.label;
      entry.append(label);
      if (it.items && it.items.length) {
        entry.append(buildList(it.items));
      } else if (!it.disabled && it.action) {
        entry.addEventListener("click", () => {
          close();
          it.action();
        });
      }
      box.append(entry);
    }
    return box;
  }

  document.addEventListener("click", () => close());
  document.addEventListener("keydown", (e) => {
    if (e.key === "Escape") {
      close();
    }
  });

  return { render, refresh, close, isOpen: () => openIndex !== -1 };
})();
