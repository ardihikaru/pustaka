# Pustaka Authoring Spec (v0.6)

The contract every documentation page must follow. `pustaka check ./docs`
enforces the STRUCTURAL rules mechanically; the COMPONENT vocabulary keeps
pages visually and behaviorally consistent. AI models: produce pages from
this spec only — do not invent new CSS classes or layouts.

---

## 1. Structural rules (validated by `pustaka check`)

1. File is `kebab-case.html`, either in the docs root or in nested
   kebab-case folders (`guide/…`, `guide/deploy/…` — any depth). Names
   starting with `_` (files or folders) are ignored by the engine.
   **Asset prefix rule:** links to `assets/…` use `../` once per folder
   level — root pages use `assets/site.css`, `guide/…` pages use
   `../assets/site.css`, `guide/deploy/…` pages use `../../assets/site.css`.
   Internal links are relative to the page's own folder and must resolve to
   real files (`check` verifies both).
2. Starts with `<!DOCTYPE html>` and includes the viewport meta tag.
3. `<body data-page="ID">` where ID equals the `id` registered in
   `assets/toc.js`.
4. Exactly one `<main class="doc">…</main>` containing ALL authored content.
   Optional extra classes: `wide` (full-width, landing-style) and `no-pn`
   (suppress prev/next).
5. Every `h2` has a unique kebab-case `id` (it becomes a search record and an
   "on this page" entry). `h3` ids are optional but indexed when present.
6. Last two elements of `<body>`: `<script src="assets/toc.js">` then
   `<script src="assets/site.js">` — with the folder-depth prefix from
   rule 1 applied to both.
7. The page is registered in the ToC (see §4).

Never author: headers, sidebars, search UI, prev/next, footers, back-to-top.
The runtime injects all of it.

## 2. Content skeleton

```html
<p class="eyebrow">Guide</p>            <!-- group name, matches toc.js group -->
<h1>Page title</h1>
<p class="lede">One–two sentence summary (indexed for search).</p>

<h2 id="section-id">Section title</h2>
<p>…</p>
```

Long headings may set `data-short="Short label"` on the h2/h3 to control the
"on this page" label.

## 3. Component vocabulary (use these, nothing else)

### Optional page metadata

Place at most one metadata component immediately after the lede. The element
stays empty: the runtime renders its accessible chips, dates and version badge.

```html
<div class="page-meta"
     data-tags="deployment, production"
     data-hashtags="deploy, operations"
     data-published="2026-07-31"
     data-updated="2026-08-04"
     data-version-from="0.6.0"
     data-version-to="0.8.x"></div>
```

The component is optional, but all attributes except `data-version-to` are
required and non-empty when it is used. Dates are ISO `YYYY-MM-DD`, updated
cannot precede published, hashtags are lowercase kebab-case without `#`, and
the lower version is a full SemVer. An empty/omitted upper bound means `+`;
an upper bound may be full SemVer or end in `x`. The HTML component is the
display source of truth; registry tags remain the static-search vocabulary.

### Admonitions — note | tip | warn
```html
<div class="adm note">
  <p class="adm-title">Note</p>
  <p>Body text.</p>
</div>
```

### Code block (copy button auto-wired)
```html
<div class="codeblock">
  <div class="bar"><span class="lang">go</span><button class="copy">copy</button></div>
  <pre>code here — escape &lt; &gt; &amp;</pre>
</div>
```
Optional highlight spans inside `<pre>`: `<span class="c">comment</span>`,
`k` keyword, `s` string, `f` function.

### Content tabs (first head/panel get class `on`)
```html
<div class="ctabs">
  <div class="heads"><button class="on">macOS</button><button>Windows</button></div>
  <div class="panel on">…</div>
  <div class="panel">…</div>
</div>
```

### Live filter (input + target list + empty state)
```html
<div class="filterbox">
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"
       stroke-linecap="round"><circle cx="11" cy="11" r="7"/><path d="m20 20-3.5-3.5"/></svg>
  <input data-filter="#my-list" data-empty="#my-empty" type="search"
         placeholder="Filter…" aria-label="Filter list">
  <span class="count"></span>
</div>
<div class="cards" id="my-list">                <!-- or a tbody id for tables -->
  <a class="card" href="x.html" data-tags="extra keywords">…</a>
</div>
<div class="empty-state" id="my-empty">Nothing matches.</div>
```
Filterable children match on their text content plus `data-tags`.

### Card grid item
```html
<a class="card" href="target.html" data-tags="keywords">
  <span class="k">Category</span><h3>Title</h3>
  <p>One-line description.</p>
  <div class="tags"><span>tag</span><span>tag</span></div>
</a>
```

### Reference table
```html
<table class="tbl">
  <thead><tr><th>Col</th><th>Col</th></tr></thead>
  <tbody id="optional-filter-target"><tr data-tags="…"><td>…</td><td>…</td></tr></tbody>
</table>
```

### Others
- Feature pillars: `<div class="pillars"><div class="pillar">…</div></div>`
- Flow diagram: `.flow` > `.node` / `.arrow` (see architecture.html)
- Scroll-reveal: add class `rv` to any block
- Prose: plain `<p>`, `<ul>`, `<ol>`, inline `<code>`

## 4. Registering in the ToC (parent + parts)

The registry is split so every file stays under **200 lines**
(`check` warns past that):

- `assets/toc.js` — parent: `site` metadata, the `product` block
  (name / `semver` / repo — one repo documents ONE product), and the
  `parts` list.
- `assets/toc/<part>.js` — one file per nav group; each calls
  `window.DOCS.nav.push({ group, pages: […] })` and ends its `pages` array
  with the `/* pustaka:insert */` marker, which `pustaka new` uses to
  auto-register pages. Never delete the marker.

Append your entry to the right part's `pages` array, above the marker, or move
the complete leaf into a recursive menu's `children` array.
`file` is the path from the docs root, forward slashes. **Key order is
mandatory** — `id, file, title, desc, tags, sections` — it is what makes
the registry machine-readable by the Go engine:

```js
{
  id:    "my-page",
  file:  "guide/my-page.html",
  title: "My page",
  desc:  "One-line description (shown in search + cards).",
  tags:  ["keyword", "keyword"],
  sections: [
    { anchor: "section-id", title: "Section title",
      text: "Searchable summary — only used in static mode; the server generates its own index from the HTML." }
  ]
}
```

`sections` may be a rough summary: under `pustaka serve` it is replaced by
the index generated from the real page text.

Menus can nest to any practical depth. They are not pages and use the exact
key order `id, title, children`; ids must be stable because the runtime uses
them to persist disclosure state:

```js
{
  id: "guide-components-menu",
  title: "Components",
  children: [
    { id: "guide-navigation-menu", title: "Navigation", children: [
      /* complete page leaves */
    ] }
  ]
}
```

Never duplicate a leaf at the root and inside a menu. The active ancestor path
opens automatically; search and previous/next flatten leaves in source order.

## 5. Changelog (one product, one history)

`changelog.html` at the docs root is the product's release history. Every
release adds a `.rel` section at the TOP of `.clog`, and updates
`product.semver` in `assets/toc.js` — treat the two as one change. Version
h3 ids use the pattern `v0-3-0` so releases are searchable and linkable.

```html
<div class="clog">
  <section class="rel rv">
    <h3 id="v1-2-0">v1.2.0 <time datetime="2026-08-01">2026-08-01</time>
      <span class="bump minor">minor</span></h3>       <!-- major | minor | patch -->
    <ul>
      <li><span class="chg add">Added</span> …</li>     <!-- add | change | fix | remove -->
      <li><span class="chg change">Changed</span> …</li>
      <li><span class="chg fix">Fixed</span> …</li>
      <li><span class="chg remove">Removed</span> …</li>
    </ul>
  </section>
  <!-- older releases below -->
</div>
```

Bump rules (SemVer): **major** = forks/pages must change to keep working;
**minor** = new backwards-compatible capability; **patch** = fixes only.

## 6. Scaffolding with `pustaka new`

Prefer scaffolding over hand-writing boilerplate:

```bash
pustaka new ./docs guide/deploy/production.html \
  --title "Production deployment" --part guide --group "Guide"
```

It stamps `_template.html` with the right ids and the correct asset prefix
for the folder depth, and registers the page at the part's insert marker.
Then replace the TODO desc/tags and author the content. The command inserts a
flat leaf; move that complete leaf into `children` for nested placement.

## 7. Interactive charts (Apache ECharts)

**Never hard-code `<script src="…echarts…">` in a page.** Ask the runtime:
it loads the vendored copy at `assets/vendor/echarts.min.js` first, falls back
to the CDN, dedupes concurrent calls, and survives partial swaps.

```html
<script>
(function () {
  const boot = (H) => H.echarts().then((echarts) => {
    /* dispose instances from a previous visit — swaps re-run this block */
    (window.__pustakaCharts || []).forEach(x => { try { x.c.dispose(); } catch (e) {} });
    const charts = window.__pustakaCharts = [];
    /* …build options from CSS tokens, lazy-init via IntersectionObserver… */
    document.addEventListener("pustaka:theme", () =>
      charts.forEach(({ c, d }) => { if (!c.isDisposed()) c.setOption(d.opt()); }));
  }).catch((err) => {
    /* never fail silently: an empty box is undiagnosable */
    document.querySelectorAll(".chart").forEach(n => H.chartFailed(n, err.message));
  });
  if (window.pustaka) boot(window.pustaka);
  else document.addEventListener("pustaka:ready", e => boot(e.detail), { once: true });
})();
</script>
```

Rules: every chart box is `<div class="chart" id="…">` (the stylesheet gives it
height — never rely on content to size it); read colours from CSS tokens so the
theme toggle re-skins charts; always guard `isDisposed()` before touching an
instance after a swap.

### 7b. Landing layout (home page only)

`<body data-layout="landing">` replaces the docs chrome with a top bar: nav
groups become top-level links, and the sidebar, page-toc rail and prev/next are
suppressed. Use `.lp-hero` / `.lp-full` for full-bleed sections; everything else
is width-limited automatically. Exactly one page per site should use it. The
partial endpoint reports `layout`, so the runtime restores the docs chrome when
you navigate away — do not set the body class by hand.

### 7c. Chart markup

```html
<div class="chart-frame rv">
  <p class="cap">caption · lower is better</p>
  <div class="chart" id="chart-unique-id"></div>
</div>

<script>
(function () {
  if (!window.echarts) return;
  const css = v => getComputedStyle(document.documentElement).getPropertyValue(v).trim();
  (window.__myPageCharts || []).forEach(x => { try { x.c.dispose(); } catch (e) {} });
  const charts = window.__myPageCharts = [];
  const build = () => ({ /* option; take colors from css('--accent'), css('--ink'),
                            css('--muted'), css('--line'), css('--panel') */ });
  const n = document.getElementById('chart-unique-id');
  const c = echarts.init(n); c.setOption(build()); charts.push({ c, build });
  document.addEventListener('pustaka:theme', () =>
    charts.forEach(({ c, build }) => { if (!c.isDisposed()) c.setOption(build(), { notMerge: true }); }));
  addEventListener('resize', () => charts.forEach(({ c }) => { if (!c.isDisposed()) c.resize(); }), { passive: true });
})();
</script>
```

## 8. Script rules (critical for HTMX partial navigation)

**Load order.** Page scripts sit inside `<main class="doc">`, so on a normal
page load they execute BEFORE `assets/site.js` (which is the last element of
`<body>`); on a partial swap they execute AFTER it. Any script that uses the
runtime API must therefore handle both:

```html
<script>
(function () {
  const start = (H) => {           /* H === window.pustaka */
    /* … use H.search(q, limit), H.highlight(text, terms), H.url(path) … */
  };
  if (window.pustaka) start(window.pustaka);
  else document.addEventListener("pustaka:ready", e => start(e.detail), { once: true });
})();
</script>
```

`window.pustaka` exposes: `search(q, limit)` → `{ terms, results }`,
`highlight(text, terms)`, `url(relPath)`, `records`, `pages`, `version`,
`product`, `serverMode`, `auth`, `onIndex(fn)` (fires when the server-generated
index replaces the static one) and `onAuth(fn)`. Never re-implement search in a
page — embed this API so behaviour stays identical to the overlay.

`onAuth(fn)` reports the optional login layer once the server has answered,
which is *after* `pustaka:ready`; late callers are replayed immediately, so a
script that re-runs on a partial swap still gets its answer. The callback
receives `null` when the site is not gated (including static `file://` mode),
otherwise `{ enabled, authenticated, user?, loginUrl, logoutUrl? }`. Use it to
show something only to signed-out visitors, and always handle `null`:

```js
H.onAuth(auth => {
  if (!auth || !auth.enabled || auth.authenticated) return;
  /* … render a sign-in affordance … */
});
```

### Other script rules


- Page-specific `<script>` tags live **inside** `<main class="doc">`, as its
  last children. Scripts outside `main` never run after a partial swap.
- Scripts **re-execute on every visit** to the page. Make them idempotent:
  dispose/replace previous instances via a `window.__<page>…` slot (pattern
  in §5), and never assume first-run.
- Theme changes dispatch `pustaka:theme` on `document`; read colors from CSS
  variables at render time, never hard-code hex values.

## 9. CSS rules that bite

- **Only use `var()` tokens defined in `:root`.** An undefined custom property
  makes the whole declaration invalid at computed-value time, so the browser
  drops it *silently* — no console warning. The defined set is: `--paper`,
  `--panel`, `--ink`, `--muted`, `--faint`, `--line`, `--accent`,
  `--accent-strong`, `--accent-soft`, `--mark`, `--mark-ink`, `--code-bg`,
  `--code-ink`, `--shadow`, `--r-s`, `--r-m`, `--r-l`, `--ease`, `--header-h`.
- **`.hide` is a state utility** applied by the filter runtime to *any* element
  type. It is defined once as `display:none !important` — never re-define
  hiding per component, or filters will apply the class without hiding
  anything.
- **Reserved shell class names** (`.pn`, `.foot`, `.pagetoc`, `.sidebar`,
  `.hd`) are injected and cleaned up by the runtime. Do not reuse them for page
  components; pick a component-scoped name instead.
- Anything text-like below ~11px on a device mockup or badge is unreadable in
  practice. Prefer a real screenshot in `assets/img/` over a CSS miniature.

## 10. Writing style

- Sentence case headings; short ledes; concrete verbs ("Run it", not "Execution").
- `code` for identifiers, file names, commands.
- Explain before showing; one idea per section; keep sections indexable
  (2–8 sentences of body text under each h2).
