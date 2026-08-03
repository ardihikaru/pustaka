/* ============================================================
   PUSTAKA runtime — v0.6
   Static mode : open any page from disk. Shell is injected
                 client-side, search runs over toc.js sections.
   Server mode : when served by the `pustaka` Go binary, the
                 runtime auto-detects /__pustaka/*, replaces the
                 hand-written index with the server-generated
                 one, and navigates via partial swaps of
                 <main.doc> (the HTMX pattern) with the View
                 Transitions API for animation.
   ============================================================ */
(function () {
  "use strict";
  const doc = document;

  /* Docs root = where assets/site.js was loaded from. Works from any
     folder depth, in static (file://) and server mode alike. */
  const self = doc.currentScript || doc.querySelector('script[src$="site.js"]');
  const SITE_ROOT = new URL(self.getAttribute("src").replace(/assets\/site\.js$/, "") || ".", location.href);
  const abs = p => new URL(p, SITE_ROOT).href;

  const loadPart = src => new Promise((res, rej) => {
    const t = doc.createElement("script");
    t.src = abs(src); t.onload = res; t.onerror = () => rej(new Error(src));
    doc.head.append(t);
  });

  async function boot() {
    const D = window.DOCS;
    if (Array.isArray(D.parts)) {
      for (const p of D.parts) {
        try { await loadPart(p); } catch (e) { console.error("pustaka: failed to load toc part:", e.message); }
      }
    }
    init(D);
  }

  function init(D) {
  const isMenu = node => Array.isArray(node && node.children);
  const childrenOf = node => isMenu(node) ? node.children : [];
  const flattenPages = nodes => (nodes || []).flatMap(node =>
    isMenu(node) ? flattenPages(childrenOf(node)) : (node && node.file ? [node] : []));
  const firstPage = nodes => flattenPages(nodes)[0] || null;
  const FLAT = D.nav.flatMap(g => flattenPages(g.pages));
  const byId = id => FLAT.find(p => p.id === id);
  const byFile = f => FLAT.find(p => p.file === f);
  let HERE = byId(doc.body.dataset.page) || FLAT[0];

  /* Layout modes. "landing" replaces the docs chrome (sidebar + page toc)
     with a marketing top bar: primary groups become top-level links and the
     article runs full-bleed. Set with <body data-layout="landing">. */
  const isLanding = () => doc.body.dataset.layout === "landing";
  const SVR = abs("__pustaka");
  let serverMode = false;

  /* ---------- helpers ---------- */
  const el = (html) => { const t = doc.createElement("template"); t.innerHTML = html.trim(); return t.content.firstElementChild; };
  const esc = (s) => s.replace(/[&<>"]/g, c => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" }[c]));
  const reduced = matchMedia("(prefers-reduced-motion: reduce)").matches;
  const IO = window.IntersectionObserver || class {
    constructor(cb) { this.cb = cb; }
    observe(t) { this.cb([{ isIntersecting: true, target: t }], this); }
    unobserve() { } disconnect() { }
  };
  const store = {
    get(k) { try { return localStorage.getItem(k); } catch { return null; } },
    set(k, v) { try { localStorage.setItem(k, v); } catch { } }
  };


  /* ------------------------------------------------------------
     ECharts loader. Pages must not hand-roll <script src=…>:
     on a partial swap the tag re-executes and ordering gets
     subtle. Call `pustaka.echarts()` and await it instead.
     Order: locally vendored copy → CDN → visible error state.
     ------------------------------------------------------------ */
  let echartsPromise = null;
  const ECHARTS_SOURCES = [abs("assets/vendor/echarts.min.js"),
                           "https://cdnjs.cloudflare.com/ajax/libs/echarts/5.5.1/echarts.min.js"];

  function loadScript(src) {
    return new Promise((res, rej) => {
      const t = doc.createElement("script");
      t.src = src;
      t.onload = () => res(src);
      t.onerror = () => rej(new Error("failed: " + src));
      doc.head.append(t);
    });
  }

  function loadEcharts() {
    if (window.echarts) return Promise.resolve(window.echarts);
    if (echartsPromise) return echartsPromise;
    echartsPromise = (async () => {
      for (const src of ECHARTS_SOURCES) {
        try {
          await loadScript(src);
          if (window.echarts) return window.echarts;
        } catch (e) { /* try the next source */ }
      }
      throw new Error("ECharts could not be loaded from any source");
    })();
    return echartsPromise;
  }

  /* Renders a readable failure inside the chart box rather than
     leaving an empty rectangle the reader cannot diagnose. */
  function chartFailed(node, msg) {
    if (!node) return;
    node.innerHTML = `<div class="chart-fail">
      <b>Chart unavailable</b>
      <span>${esc(msg)}</span>
      <span class="hint">Vendor <code>assets/vendor/echarts.min.js</code> or allow the CDN, then reload.</span>
    </div>`;
  }

  /* ---------- theme ---------- */
  const root = doc.documentElement;
  let theme = store.get("pustaka-theme") || (matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light");
  const applyTheme = () => { root.dataset.theme = theme; doc.dispatchEvent(new CustomEvent("pustaka:theme", { detail: theme })); };
  applyTheme();

  /* ---------- icons ---------- */
  const I = {
    logo: '<svg viewBox="0 0 24 24" aria-hidden="true" focusable="false"><path d="M7 19V5h6.2a4.6 4.6 0 0 1 0 9.2H7M7 9.6h6.2a.8.8 0 0 0 0-1.6H7" fill="currentColor"/></svg>',
    search: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round"><circle cx="11" cy="11" r="7"/><path d="m20 20-3.5-3.5"/></svg>',
    burger: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round"><path d="M4 7h16M4 12h16M4 17h16"/></svg>',
    sun: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><circle cx="12" cy="12" r="4.5"/><path d="M12 2.5v2.5M12 19v2.5M2.5 12H5M19 12h2.5M5.3 5.3l1.8 1.8M16.9 16.9l1.8 1.8M18.7 5.3l-1.8 1.8M7.1 16.9l-1.8 1.8"/></svg>',
    moon: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20.5 14.5A8.5 8.5 0 0 1 9.5 3.5a8.5 8.5 0 1 0 11 11z"/></svg>',
    up: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round" width="17"><path d="M12 19V6M6 12l6-6 6 6"/></svg>',
    close: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round"><path d="M6 6l12 12M18 6 6 18"/></svg>',
    signOut: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><path d="M14.5 4.5H6.8A1.8 1.8 0 0 0 5 6.3v11.4a1.8 1.8 0 0 0 1.8 1.8h7.7"/><path d="M18.5 12H10m8.5 0-3-3m3 3-3 3"/></svg>',
    signIn: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><path d="M9.5 4.5h7.7A1.8 1.8 0 0 1 19 6.3v11.4a1.8 1.8 0 0 1-1.8 1.8H9.5"/><path d="M5.5 12H14M5.5 12l3-3m-3 3 3 3"/></svg>'
  };

  /* ============================================================
     SHELL — built once
     ============================================================ */
  const header = el(`
    <header class="hd">
      <button class="burger icon-btn" aria-label="Open navigation">${I.burger}</button>
      <a class="brand" href="${abs('index.html')}">
        <span class="glyph">${I.logo}</span><span>${D.site.name}</span><span class="ver">${D.site.version}</span>
      </a>
      <nav class="topnav" aria-label="Sections">
        ${D.nav.map(g => {
          const first = firstPage(g.pages);
          return first ? `<a href="${abs(first.file)}" data-group="${g.group}">${g.group}</a>` : "";
        }).join("")}
      </nav>
      <div class="spacer"></div>
      <button class="search-btn" aria-label="Search docs">
        ${I.search}<span class="lbl">Search docs…</span><kbd>Ctrl K</kbd>
      </button>
      <button class="icon-btn theme" aria-label="Toggle color theme"></button>
    </header>`);
  doc.body.prepend(header);
  const applyLayout = () => {
    const L = isLanding();
    doc.body.classList.toggle("layout-landing", L);
    header.classList.toggle("hd-landing", L);
  };
  applyLayout();

  const themeBtn = header.querySelector(".theme");
  const paintThemeBtn = () => themeBtn.innerHTML = theme === "dark" ? I.sun : I.moon;
  paintThemeBtn();
  themeBtn.addEventListener("click", () => {
    theme = theme === "dark" ? "light" : "dark";
    store.set("pustaka-theme", theme); applyTheme(); paintThemeBtn();
  });

  const shell = doc.querySelector(".shell");
  const menuStateKey = "pustaka-nav-open";
  let savedMenus = new Set();
  try { savedMenus = new Set(JSON.parse(store.get(menuStateKey) || "[]")); } catch { savedMenus = new Set(); }
  const menuPaths = new Map();
  const mapPaths = (nodes, ancestors = []) => (nodes || []).forEach(node => {
    if (isMenu(node)) {
      const next = [...ancestors, node.id];
      flattenPages(childrenOf(node)).forEach(page => menuPaths.set(page.id, next));
      mapPaths(childrenOf(node), next);
    }
  });
  D.nav.forEach(g => mapPaths(g.pages));
  const renderNav = (nodes, depth = 0) => (nodes || []).map(node => {
    if (!isMenu(node)) {
      return `<a href="${abs(node.file)}" data-nav="${node.id}" style="--nav-depth:${depth}">${esc(node.title)}</a>`;
    }
    const open = savedMenus.has(node.id);
    return `<div class="nav-branch" data-menu="${esc(node.id)}" style="--nav-depth:${depth}">
      <button class="nav-toggle" type="button" aria-expanded="${open}" aria-controls="nav-${esc(node.id)}">
        <span>${esc(node.title)}</span><span class="nav-chevron" aria-hidden="true">›</span>
      </button>
      <div class="nav-children" id="nav-${esc(node.id)}" ${open ? "" : "hidden"}>${renderNav(childrenOf(node), depth + 1)}</div>
    </div>`;
  }).join("");
  const sidebar = el(`<nav class="sidebar" aria-label="Documentation">
      <div class="drawer-head">
        <span class="brand"><span class="glyph">${I.logo}</span>${D.site.name}</span>
        <button class="icon-btn drawer-close" aria-label="Close navigation">${I.close}</button>
      </div>
      ${D.nav.map(g => `
        <div class="group">
          <p class="group-label">${g.group}</p>
          ${renderNav(g.pages)}
        </div>`).join("")}
    </nav>`);
  shell.prepend(sidebar);

  const veil = el(`<div class="veil"></div>`);
  doc.body.append(veil);
  const openDrawer = (v) => { sidebar.classList.toggle("open", v); veil.classList.toggle("open", v); };
  header.querySelector(".burger").addEventListener("click", () => openDrawer(true));
  sidebar.querySelector(".drawer-close").addEventListener("click", () => openDrawer(false));
  sidebar.querySelectorAll(".nav-toggle").forEach(btn => btn.addEventListener("click", () => {
    const branch = btn.closest(".nav-branch");
    const id = branch.dataset.menu;
    const open = btn.getAttribute("aria-expanded") !== "true";
    btn.setAttribute("aria-expanded", String(open));
    branch.querySelector(":scope > .nav-children").hidden = !open;
    if (open) savedMenus.add(id); else savedMenus.delete(id);
    store.set(menuStateKey, JSON.stringify([...savedMenus]));
  }));

  function paintSidebar() {
    sidebar.querySelectorAll("a[data-nav]").forEach(a =>
      a.classList.toggle("active", a.dataset.nav === HERE.id));
    const activeMenus = new Set(menuPaths.get(HERE.id) || []);
    activeMenus.forEach(id => {
      const branch = sidebar.querySelector(`.nav-branch[data-menu="${CSS.escape(id)}"]`);
      if (!branch) return;
      const btn = branch.querySelector(":scope > .nav-toggle");
      btn.setAttribute("aria-expanded", "true");
      branch.querySelector(":scope > .nav-children").hidden = false;
    });
    sidebar.querySelectorAll(".nav-branch").forEach(branch =>
      branch.classList.toggle("has-active", activeMenus.has(branch.dataset.menu)));
    /* highlight the top-bar group that owns the current page */
    const owner = D.nav.find(g => flattenPages(g.pages).some(p => p.id === HERE.id));
    header.querySelectorAll("a[data-group]").forEach(a =>
      a.classList.toggle("active", !!owner && a.dataset.group === owner.group));
  }
  paintSidebar();

  const topBtn = el(`<button class="top-btn" aria-label="Back to top">${I.up}</button>`);
  doc.body.append(topBtn);
  topBtn.addEventListener("click", () => scrollTo({ top: 0, behavior: reduced ? "auto" : "smooth" }));
  addEventListener("scroll", () => topBtn.classList.toggle("show", scrollY > 600), { passive: true });

  const rvObs = new IO(es => es.forEach(e => {
    if (e.isIntersecting) { e.target.classList.add("in"); rvObs.unobserve(e.target); }
  }), { rootMargin: "0px 0px -8% 0px" });

  /* ============================================================
     PER-PAGE WIRING — runs at load and after every partial swap
     ============================================================ */
  const article = doc.querySelector(".doc");
  let spyObs = null;

  function wirePage() {
    applyLayout();
    /* ---- optional declarative page metadata ---- */
    article.querySelectorAll(".page-meta:not([data-rendered])").forEach(meta => {
      const split = value => String(value || "").split(",").map(x => x.trim()).filter(Boolean);
      const tags = split(meta.dataset.tags);
      const hashtags = split(meta.dataset.hashtags);
      const from = meta.dataset.versionFrom || "";
      const to = meta.dataset.versionTo || "";
      meta.dataset.rendered = "1";
      meta.setAttribute("aria-label", "Page metadata");
      meta.innerHTML = `
        <div class="page-meta-tags"><span class="page-meta-label">Tags</span>${tags.map(x => `<span class="page-chip">${esc(x)}</span>`).join("")}</div>
        <div class="page-meta-hash">${hashtags.map(x => `<span>#${esc(x.replace(/^#/, ""))}</span>`).join("")}</div>
        <dl>
          <div><dt>Published</dt><dd><time datetime="${esc(meta.dataset.published || "")}">${esc(meta.dataset.published || "")}</time></dd></div>
          <div><dt>Last updated</dt><dd><time datetime="${esc(meta.dataset.updated || "")}">${esc(meta.dataset.updated || "")}</time></dd></div>
          <div><dt>Applies to</dt><dd><span class="page-version">v${esc(from)}${to ? `–v${esc(to)}` : "+"}</span></dd></div>
        </dl>`;
    });
    /* ---- on-page toc + scrollspy ---- */
    doc.querySelector(".pagetoc")?.remove();
    spyObs?.disconnect();
    const heads = [...article.querySelectorAll("h2[id],h3[id]")];
    if (heads.length && !article.classList.contains("no-toc") && !isLanding()) {
      const toc = el(`<aside class="pagetoc"><p class="label">On this page</p>
        ${heads.map(h => `<a href="#${h.id}" class="${h.tagName === "H3" ? "h3" : ""}" data-for="${h.id}">${esc((h.dataset.short || h.textContent).replace("¶", "").trim())}</a>`).join("")}
      </aside>`);
      shell.append(toc);
      const links = new Map([...toc.querySelectorAll("a")].map(a => [a.dataset.for, a]));
      spyObs = new IO((es) => {
        es.forEach(e => { if (e.isIntersecting) { links.forEach(l => l.classList.remove("active")); links.get(e.target.id)?.classList.add("active"); } });
      }, { rootMargin: "-15% 0px -70% 0px" });
      heads.forEach(h => spyObs.observe(h));
    }
    heads.forEach(h => {
      if (!h.querySelector(".anchor-link"))
        h.append(el(`<a class="anchor-link" href="#${h.id}" aria-label="Link to section">¶</a>`));
    });

    /* ---- prev / next + footer ---- */
    article.querySelector(":scope > .pn")?.remove();
    article.querySelector(":scope > .foot")?.remove();
    if (!article.classList.contains("no-pn") && !isLanding()) {
      const i = FLAT.indexOf(HERE), prev = FLAT[i - 1], next = FLAT[i + 1];
      if (prev || next) {
        article.append(el(`<nav class="pn">
          ${prev ? `<a href="${abs(prev.file)}"><span class="dir">← Previous</span><span class="t">${prev.title}</span></a>` : ""}
          ${next ? `<a class="next" href="${abs(next.file)}"><span class="dir">Next →</span><span class="t">${next.title}</span></a>` : ""}
        </nav>`));
      }
    }
    article.append(el(`<footer class="foot">
      <span>${D.site.name} — ${D.site.tagline}</span>
      <span class="mono">${serverMode ? "go server · htmx partial nav" : "static mode · open from disk"}</span>
    </footer>`));

    /* ---- in-page live filter ---- */
    article.querySelectorAll("input[data-filter]").forEach(inp => {
      if (inp.dataset.wired) return; inp.dataset.wired = "1";
      const wrap = doc.querySelector(inp.dataset.filter);
      if (!wrap) return;
      const kids = [...wrap.children];
      const empty = doc.querySelector(inp.dataset.empty || "___none");
      const count = inp.closest(".filterbox")?.querySelector(".count");
      const paint = () => {
        const terms = inp.value.toLowerCase().split(/\s+/).filter(Boolean);
        let shown = 0;
        kids.forEach(k => {
          const hay = (k.textContent + " " + (k.dataset.tags || "")).toLowerCase();
          const ok = terms.every(t => hay.includes(t));
          const was = !k.classList.contains("hide");
          k.classList.toggle("hide", !ok);
          if (ok) { shown++; if (!was) { k.classList.remove("pop"); void k.offsetWidth; k.classList.add("pop"); } }
        });
        if (count) count.innerHTML = `<b>${shown}</b>/${kids.length}`;
        empty?.classList.toggle("show", shown === 0);
      };
      inp.addEventListener("input", paint); paint();
    });

    /* ---- code copy ---- */
    article.querySelectorAll(".codeblock").forEach(cb => {
      const btn = cb.querySelector(".copy");
      if (!btn || btn.dataset.wired) return; btn.dataset.wired = "1";
      btn.addEventListener("click", async () => {
        const text = cb.querySelector("pre").innerText;
        try { await navigator.clipboard.writeText(text); } catch {
          const ta = doc.createElement("textarea"); ta.value = text; doc.body.append(ta); ta.select(); doc.execCommand("copy"); ta.remove();
        }
        btn.textContent = "copied ✓"; btn.classList.add("done");
        setTimeout(() => { btn.textContent = "copy"; btn.classList.remove("done"); }, 1600);
      });
    });

    /* ---- content tabs ---- */
    article.querySelectorAll(".ctabs").forEach(t => {
      if (t.dataset.wired) return; t.dataset.wired = "1";
      const hs = [...t.querySelectorAll(".heads button")];
      const ps = [...t.querySelectorAll(".panel")];
      hs.forEach((h, i) => h.addEventListener("click", () => {
        hs.forEach(x => x.classList.remove("on")); ps.forEach(x => x.classList.remove("on"));
        h.classList.add("on"); ps[i].classList.add("on");
      }));
    });

    /* ---- reveal ---- */
    doc.querySelectorAll(".rv:not(.in)").forEach(n => rvObs.observe(n));
  }
  wirePage();

  /* ============================================================
     SEARCH — index from toc.js, replaced by the server's
     generated index when running under `pustaka serve`.
     ============================================================ */
  let INDEX = FLAT.flatMap(p => [
    { page: { id: p.id, title: p.title }, title: p.title, text: p.desc + " " + p.tags.join(" "), href: p.file, top: true },
    ...p.sections.map(s => ({ page: { id: p.id, title: p.title }, title: s.title, text: s.text, href: `${p.file}#${s.anchor}` }))
  ]);

  /* Public API for page-level scripts (landing page search demo, etc.).
     Pages must tolerate it being absent: guard with `if (!window.pustaka) return;` */
  const indexHooks = [];
  const API = {
    version: (D.site && D.site.version) || "",
    product: D.product || null,
    pages: FLAT.length,
    records: INDEX.length,
    serverMode: false,
    search: (q, limit) => query(q, limit),
    highlight: (text, terms) => hi(text, terms),
    url: (p) => abs(p),
    onIndex: (fn) => { indexHooks.push(fn); return () => { }; },
    echarts: () => loadEcharts(),
    chartFailed: (node, msg) => chartFailed(node, msg),
    reduced
  };
  window.pustaka = API;

  const modal = el(`<div class="search-modal" role="dialog" aria-label="Search documentation">
      <div class="row">${I.search}
        <input type="search" placeholder="Search across all pages…" autocomplete="off" spellcheck="false">
        <kbd class="esc">esc</kbd>
      </div>
      <div class="search-results"></div>
      <div class="sr-hint"><span><kbd>↑</kbd><kbd>↓</kbd> navigate</span><span><kbd>↵</kbd> open</span><span><kbd>esc</kbd> close</span></div>
    </div>`);
  doc.body.append(modal);
  const input = modal.querySelector("input");
  const results = modal.querySelector(".search-results");
  let sel = 0;

  const hi = (text, terms) => {
    let out = esc(text);
    terms.forEach(t => { out = out.replace(new RegExp(`(${t.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")})`, "ig"), "<mark>$1</mark>"); });
    return out;
  };

  /* The scoring core. Shared by the search overlay and by any page that
     wants to embed search UI (the landing page does — see window.pustaka). */
  function query(q, limit) {
    const terms = String(q || "").toLowerCase().split(/\s+/).filter(Boolean);
    if (!terms.length) return { terms, results: [] };
    const scored = INDEX.map(r => {
      const t = r.title.toLowerCase(), x = r.text.toLowerCase();
      let s = 0;
      for (const w of terms) {
        if (!t.includes(w) && !x.includes(w)) return null;   // AND semantics, like mkdocs
        if (t.startsWith(w)) s += 6; else if (t.includes(w)) s += 4;
        if (x.includes(w)) s += 1;
      }
      return { r, s: s + (r.top ? 1 : 0) };
    }).filter(Boolean).sort((a, b) => b.s - a.s).slice(0, limit || 14);
    return { terms, results: scored.map(o => o.r) };
  }

  function runSearch(q) {
    const { terms, results: found } = query(q, 14);
    if (!terms.length) {
      results.innerHTML = `<p class="sr-empty">Type to search titles, tags and section text across every page${serverMode ? " (server-generated index)" : ""}.</p>`;
      return;
    }
    const scored = found.map(r => ({ r }));

    if (!scored.length) {
      results.innerHTML = `<p class="sr-empty">No matches for “${esc(q)}”. Try a broader term — e.g. <b>search</b>, <b>chart</b>, <b>go</b>.</p>`;
      return;
    }
    const groups = [];
    scored.forEach(({ r }) => {
      let g = groups.find(g => g.page.id === r.page.id);
      if (!g) { g = { page: r.page, items: [] }; groups.push(g); }
      g.items.push(r);
    });
    results.innerHTML = groups.map(g => `
      <p class="sr-group">${esc(g.page.title)}</p>
      ${g.items.map(r => `
        <a class="sr-item" href="${abs(r.href)}">
          <span class="st">${hi(r.title, terms)}</span>
          <span class="sx">${hi(r.text, terms)}</span>
        </a>`).join("")}
    `).join("");
    sel = 0; paintSel();
  }

  const items = () => [...results.querySelectorAll(".sr-item")];
  function paintSel() {
    items().forEach((a, i) => a.classList.toggle("sel", i === sel));
    items()[sel]?.scrollIntoView?.({ block: "nearest" });
  }
  function openSearch() {
    modal.classList.add("open"); veil.classList.add("open");
    input.value = ""; runSearch(""); setTimeout(() => input.focus(), 30);
  }
  function closeSearch() { modal.classList.remove("open"); veil.classList.remove("open"); openDrawer(false); }

  header.querySelector(".search-btn").addEventListener("click", openSearch);
  veil.addEventListener("click", closeSearch);
  input.addEventListener("input", () => runSearch(input.value));
  doc.addEventListener("keydown", (e) => {
    const inModal = modal.classList.contains("open");
    if ((e.key === "k" && (e.ctrlKey || e.metaKey)) || (e.key === "/" && !inModal && !/INPUT|TEXTAREA/.test(doc.activeElement.tagName))) {
      e.preventDefault(); openSearch();
    } else if (e.key === "Escape" && inModal) closeSearch();
    else if (inModal && e.key === "ArrowDown") { e.preventDefault(); sel = Math.min(sel + 1, items().length - 1); paintSel(); }
    else if (inModal && e.key === "ArrowUp") { e.preventDefault(); sel = Math.max(sel - 1, 0); paintSel(); }
    else if (inModal && e.key === "Enter") { const a = items()[sel]; if (a) { closeSearch(); navTo(a.getAttribute("href")); } }
  });

  /* ============================================================
     SERVER MODE — partial navigation (the HTMX pattern)
     GET /__pustaka/partial/<file> → { id, title, html }
     swapped into <main.doc>; page scripts re-execute in order.
     ============================================================ */
  function navTo(href) {
    if (serverMode) swap(href, true);
    else location.href = href;
  }

  async function execScripts(container) {
    for (const old of [...container.querySelectorAll("script")]) {
      const s = doc.createElement("script");
      [...old.attributes].forEach(a => s.setAttribute(a.name, a.value));
      if (old.src) {
        await new Promise((res) => { s.onload = s.onerror = res; old.replaceWith(s); });
      } else { s.textContent = old.textContent; old.replaceWith(s); }
    }
  }

  function relPath(u) {
    if (!u.pathname.startsWith(SITE_ROOT.pathname)) return null;
    const rel = decodeURIComponent(u.pathname.slice(SITE_ROOT.pathname.length));
    return rel === "" ? "index.html" : rel;
  }

  async function swap(href, push) {
    const u = new URL(href, location.href);
    const file = relPath(u);
    const target = file && byFile(file);
    if (!target) { location.href = href; return; }
    let j;
    try {
      const r = await fetch(`${SVR}/partial/${file}`);
      if (!r.ok) throw 0;
      j = await r.json();
    } catch { location.href = href; return; }   // graceful fallback: full load

    const resolveResources = (html, baseURL) => {
      const t = doc.createElement("template");
      t.innerHTML = html;
      t.content.querySelectorAll("[src],[poster]").forEach(node => {
        for (const name of ["src", "poster"]) {
          const value = node.getAttribute(name);
          if (value && !value.startsWith("data:") && !value.startsWith("blob:"))
            node.setAttribute(name, new URL(value, baseURL).href);
        }
      });
      t.content.querySelectorAll("[srcset]").forEach(node => {
        node.setAttribute("srcset", node.getAttribute("srcset").split(",").map(part => {
          const bits = part.trim().split(/\s+/);
          bits[0] = new URL(bits[0], baseURL).href;
          return bits.join(" ");
        }).join(", "));
      });
      return t.innerHTML;
    };

    const apply = async () => {
      /* Make the destination the document base before fragment resources or
         page scripts can resolve relative URLs. Popstate already has it. */
      if (push) history.pushState({ pustaka: true }, "", u.pathname + u.search + u.hash);
      article.className = "doc" + (j.classes ? " " + j.classes : "");
      /* <body data-layout> is outside the swapped region, so carry it over
         explicitly — otherwise the landing chrome would stick to docs pages. */
      if (j.layout) doc.body.dataset.layout = j.layout;
      else delete doc.body.dataset.layout;
      article.innerHTML = resolveResources(j.html, u);
      doc.body.dataset.page = j.id;
      HERE = byId(j.id) || target;
      doc.title = j.title;
      paintSidebar(); openDrawer(false); closeSearch();
      wirePage();
      await execScripts(article);
      const anchor = u.hash && doc.getElementById(u.hash.slice(1));
      if (anchor) anchor.scrollIntoView(); else scrollTo(0, 0);
    };

    if (doc.startViewTransition && !reduced) doc.startViewTransition(apply);
    else { article.style.animation = "fadeup .3s cubic-bezier(.22,.9,.3,1)"; await apply(); setTimeout(() => article.style.animation = "", 350); }
  }

  function enableServerMode() {
    serverMode = true;
    API.serverMode = true;
    doc.querySelector("main.doc > .foot .mono") && (doc.querySelector("main.doc > .foot .mono").textContent = "go server · htmx partial nav");
    // swap the hand-written index for the server-generated one
    fetch(`${SVR}/index.json`).then(r => r.ok ? r.json() : null).then(recs => {
      if (Array.isArray(recs) && recs.length) {
        INDEX = recs.map(r => ({
          page: { id: r.pageId, title: r.pageTitle },
          title: r.title, text: r.text, href: r.href, top: !!r.top
        }));
        API.records = INDEX.length;
        indexHooks.forEach(fn => { try { fn(API); } catch (e) { } });
      }
    }).catch(() => { });
    // intercept internal .html links → partial swap
    doc.addEventListener("click", (e) => {
      if (e.defaultPrevented || e.metaKey || e.ctrlKey || e.shiftKey || e.button !== 0) return;
      const a = e.target.closest("a[href]");
      if (!a || a.target === "_blank" || a.hasAttribute("download")) return;
      const u = new URL(a.getAttribute("href"), location.href);
      if (u.origin !== location.origin || !/\.html$/.test(u.pathname) || !u.pathname.startsWith(SITE_ROOT.pathname)) return;
      if (u.pathname === location.pathname && u.hash) return;      // same-page anchor
      e.preventDefault(); swap(u.href, true);
    });
    addEventListener("popstate", () => swap(location.href, false));
  }

  /* Optional login layer. /info reports it; when auth is off the key is
     absent and no control is injected at all. */
  const paintAuth = (info) => {
    if (!info || !info.enabled || header.querySelector(".auth-ctl")) return;
    const node = info.authenticated
      ? el(`<form class="auth-ctl" method="post" action="${SVR}/auth/logout">
             <button class="icon-btn" type="submit" title="Sign out${info.user ? " (" + esc(info.user) + ")" : ""}"
                     aria-label="Sign out">${I.signOut}</button>
           </form>`)
      : el(`<a class="icon-btn auth-ctl" href="${SVR}/auth/login?next=${
             encodeURIComponent(location.pathname + location.search)}"
             title="Sign in" aria-label="Sign in">${I.signIn}</a>`);
    header.insertBefore(node, themeBtn);
  };

  fetch(`${SVR}/info`, { cache: "no-store" })
    .then(r => (r.ok ? r.json() : Promise.reject()))
    .then(info => {
      const auth = info && info.auth;
      /* A signed-out visitor can only reach the landing page, and every
         content endpoint would answer 401. Staying in static mode keeps the
         page working off the local index with no failed requests: links do
         plain navigations, which the server redirects to the login page. */
      if (!auth || auth.authenticated) enableServerMode();
      paintAuth(auth);
    })
    .catch(() => { /* static mode — everything already works */ });

  /* Page scripts live inside <main>, so on a normal page load they run
     BEFORE this file. They wait for this event; on a partial swap the API
     already exists and they run immediately. Both paths, one contract. */
  doc.dispatchEvent(new CustomEvent("pustaka:ready", { detail: API }));
  }

  boot();
})();
