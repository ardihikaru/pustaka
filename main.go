// Pustaka — HTML-first documentation engine.
//
//	pustaka serve ./docs [--addr :8080] [--prod]
//	pustaka check ./docs                     validate against the authoring spec
//	pustaka index ./docs                     print the generated search index
//	pustaka new   ./docs guide/deploy.html --title "Deploying" [--part guide]
//
// Pages may live in nested folders (guide/…, concept/…). The registry is
// assets/toc.js plus the part files it lists (assets/toc/*.js), so each
// registry file stays small. Go stdlib only: one static binary, zero deps.
package main

import (
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	"html"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	productName         = "pustaka"
	internalRoutePrefix = "/__pustaka"
)

/* ============================== model ============================== */

type Page struct {
	ID    string   `json:"id"`
	File  string   `json:"file"` // path relative to docs root, forward slashes
	Title string   `json:"title"`
	Desc  string   `json:"desc"`
	Tags  []string `json:"tags"`
}

// Record is one searchable unit: a page summary (Top) or a section.
type Record struct {
	PageID    string `json:"pageId"`
	PageTitle string `json:"pageTitle"`
	Title     string `json:"title"`
	Text      string `json:"text"`
	Href      string `json:"href"`
	Top       bool   `json:"top,omitempty"`
}

type Site struct {
	mu      sync.RWMutex
	root    string
	pages   []Page
	records []Record
	inv     map[string][]int
	built   time.Time
}

/* ============================== parsing ============================ */

// The authoring spec mandates this key order in registry files —
// id, file, title, desc, tags — which is what makes them machine-readable
// without a JS engine.
var (
	rePageEntry  = regexp.MustCompile(`(?s)\{\s*id:\s*"([^"]+)"\s*,\s*file:\s*"([^"]+)"\s*,\s*title:\s*"([^"]+)"\s*,\s*desc:\s*"([^"]*)"\s*,\s*tags:\s*\[([^\]]*)\]`)
	reParts      = regexp.MustCompile(`(?s)parts:\s*\[([^\]]*)\]`)
	reQuoted     = regexp.MustCompile(`"([^"]+)"`)
	reMainOpen   = regexp.MustCompile(`<main\s+class="doc[^"]*"[^>]*>`)
	reMainClass  = regexp.MustCompile(`<main\s+class="doc\s*([^"]*)"`)
	reHead       = regexp.MustCompile(`(?s)<h([23])\s+[^>]*id="([^"]+)"[^>]*>(.*?)</h[23]>`)
	reLede       = regexp.MustCompile(`(?s)<p class="lede"[^>]*>(.*?)</p>`)
	reTitleTag   = regexp.MustCompile(`(?s)<title>(.*?)</title>`)
	reDataPage   = regexp.MustCompile(`data-page="([^"]+)"`)
	reDataLayout = regexp.MustCompile(`<body[^>]*\bdata-layout="([^"]*)"`)
	reScript     = regexp.MustCompile(`(?s)<script[^>]*>.*?</script>`)
	reStyle      = regexp.MustCompile(`(?s)<style[^>]*>.*?</style>`)
	reTags       = regexp.MustCompile(`<[^>]+>`)
	reSpace      = regexp.MustCompile(`\s+`)
	reToken      = regexp.MustCompile(`[a-z0-9]+`)
	reH2NoID     = regexp.MustCompile(`<h2(\s+[^>]*)?>`)
	reH2WithID   = regexp.MustCompile(`<h2\s+[^>]*id="[^"]+"`)
	reHref       = regexp.MustCompile(`href="([^"#]+\.html)(#[^"]*)?"`)
	rePageMeta   = regexp.MustCompile(`<div\b([^>]*)class="[^"]*\bpage-meta\b[^"]*"([^>]*)>`)
	reHashtag    = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
	reMenuEntry  = regexp.MustCompile(`\{\s*id:\s*"([^"]+)"\s*,\s*title:\s*"[^"]+"\s*,\s*children:\s*\[`)
	reEmptyMenu  = regexp.MustCompile(`children:\s*\[\s*\]`)
)

// registryFiles returns toc.js plus, in order, the part files it lists.
func registryFiles(root string) []string {
	files := []string{"assets/toc.js"}
	b, err := os.ReadFile(filepath.Join(root, "assets", "toc.js"))
	if err != nil {
		return files
	}
	if m := reParts.FindStringSubmatch(string(b)); m != nil {
		for _, q := range reQuoted.FindAllStringSubmatch(m[1], -1) {
			files = append(files, q[1])
		}
	}
	return files
}

func loadRegistry(root string) ([]Page, error) {
	var pages []Page
	for _, rf := range registryFiles(root) {
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rf)))
		if err != nil {
			return nil, fmt.Errorf("registry file %s: %w", rf, err)
		}
		for _, m := range rePageEntry.FindAllStringSubmatch(string(b), -1) {
			var tags []string
			for _, t := range reQuoted.FindAllStringSubmatch(m[5], -1) {
				tags = append(tags, t[1])
			}
			pages = append(pages, Page{ID: m[1], File: m[2], Title: m[3], Desc: m[4], Tags: tags})
		}
	}
	if len(pages) == 0 {
		return nil, fmt.Errorf("no pages found in the registry (assets/toc.js + parts). Do entries follow the key order id, file, title, desc, tags?")
	}
	return pages, nil
}

func stripText(s string) string {
	s = reScript.ReplaceAllString(s, " ")
	s = reStyle.ReplaceAllString(s, " ")
	s = reTags.ReplaceAllString(s, " ")
	s = html.UnescapeString(s)
	s = strings.ReplaceAll(s, "¶", "")
	return strings.TrimSpace(reSpace.ReplaceAllString(s, " "))
}

func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	cut := s[:n]
	if i := strings.LastIndex(cut, " "); i > n/2 {
		cut = cut[:i]
	}
	return cut + "…"
}

func htmlAttr(attrs, name string) (string, bool) {
	re := regexp.MustCompile(`\b` + regexp.QuoteMeta(name) + `="([^"]*)"`)
	m := re.FindStringSubmatch(attrs)
	if m == nil {
		return "", false
	}
	return html.UnescapeString(m[1]), true
}

func versionBound(raw string, wildcard bool) ([3]int, error) {
	var out [3]int
	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		return out, fmt.Errorf("must use MAJOR.MINOR.PATCH")
	}
	seenX := false
	for i, part := range parts {
		if part == "x" {
			if !wildcard || i == 0 {
				return out, fmt.Errorf("wildcard x is only allowed in an upper minor/patch bound")
			}
			seenX = true
			out[i] = 1_000_000
			continue
		}
		if seenX {
			return out, fmt.Errorf("numeric component cannot follow wildcard x")
		}
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 {
			return out, fmt.Errorf("components must be non-negative integers")
		}
		out[i] = n
	}
	return out, nil
}

func versionAfter(a, b [3]int) bool {
	for i := range a {
		if a[i] != b[i] {
			return a[i] > b[i]
		}
	}
	return false
}

func mainInner(page string) (inner, classes string, ok bool) {
	loc := reMainOpen.FindStringIndex(page)
	if loc == nil {
		return "", "", false
	}
	end := strings.LastIndex(page, "</main>")
	if end < loc[1] {
		return "", "", false
	}
	if m := reMainClass.FindStringSubmatch(page); m != nil {
		classes = strings.TrimSpace(m[1])
	}
	return page[loc[1]:end], classes, true
}

func extractRecords(p Page, raw string) []Record {
	inner, _, ok := mainInner(raw)
	if !ok {
		inner = raw
	}
	var recs []Record
	lede := ""
	if m := reLede.FindStringSubmatch(inner); m != nil {
		lede = stripText(m[1])
	}
	recs = append(recs, Record{
		PageID: p.ID, PageTitle: p.Title, Title: p.Title,
		Text: clip(strings.TrimSpace(p.Desc+" "+lede+" "+strings.Join(p.Tags, " ")), 400),
		Href: p.File, Top: true,
	})
	heads := reHead.FindAllStringSubmatchIndex(inner, -1)
	for i, h := range heads {
		id := inner[h[4]:h[5]]
		title := stripText(inner[h[6]:h[7]])
		bodyEnd := len(inner)
		if i+1 < len(heads) {
			bodyEnd = heads[i+1][0]
		}
		recs = append(recs, Record{
			PageID: p.ID, PageTitle: p.Title, Title: title,
			Text: clip(stripText(inner[h[1]:bodyEnd]), 420),
			Href: p.File + "#" + id,
		})
	}
	return recs
}

/* ============================== index ============================== */

func (s *Site) rebuild() error {
	pages, err := loadRegistry(s.root)
	if err != nil {
		return err
	}
	var records []Record
	for _, p := range pages {
		b, err := os.ReadFile(filepath.Join(s.root, filepath.FromSlash(p.File)))
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: %s registered but unreadable: %v\n", p.File, err)
			continue
		}
		records = append(records, extractRecords(p, string(b))...)
	}
	inv := make(map[string][]int)
	for i, r := range records {
		seen := map[string]bool{}
		for _, tok := range reToken.FindAllString(strings.ToLower(r.Title+" "+r.Text), -1) {
			if !seen[tok] {
				seen[tok] = true
				inv[tok] = append(inv[tok], i)
			}
		}
	}
	s.mu.Lock()
	s.pages, s.records, s.inv, s.built = pages, records, inv, time.Now()
	s.mu.Unlock()
	return nil
}

// search: AND of terms with prefix matching, ranked by title hits.
func (s *Site) search(q string, limit int) []Record {
	s.mu.RLock()
	defer s.mu.RUnlock()
	terms := reToken.FindAllString(strings.ToLower(q), -1)
	if len(terms) == 0 {
		return nil
	}
	var match map[int]int
	for _, term := range terms {
		hits := map[int]int{}
		for tok, ids := range s.inv {
			if strings.HasPrefix(tok, term) {
				for _, id := range ids {
					hits[id]++
				}
			}
		}
		if match == nil {
			match = hits
			continue
		}
		for id := range match {
			if _, ok := hits[id]; !ok {
				delete(match, id)
			}
		}
	}
	type sc struct{ i, s int }
	var out []sc
	for id, base := range match {
		r := s.records[id]
		score := base
		lt := strings.ToLower(r.Title)
		for _, term := range terms {
			if strings.HasPrefix(lt, term) {
				score += 6
			} else if strings.Contains(lt, term) {
				score += 4
			}
		}
		if r.Top {
			score++
		}
		out = append(out, sc{id, score})
	}
	sort.Slice(out, func(a, b int) bool { return out[a].s > out[b].s })
	if len(out) > limit {
		out = out[:limit]
	}
	res := make([]Record, len(out))
	for i, o := range out {
		res[i] = s.records[o.i]
	}
	return res
}

/* ============================== server ============================= */

type gzWriter struct {
	http.ResponseWriter
	gz *gzip.Writer
}

func (w gzWriter) Write(b []byte) (int, error) { return w.gz.Write(b) }
func (w gzWriter) WriteHeader(code int) {
	w.Header().Del("Content-Length")
	w.ResponseWriter.WriteHeader(code)
}

func gzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") || r.Header.Get("Range") != "" {
			next.ServeHTTP(w, r)
			return
		}
		switch strings.ToLower(filepath.Ext(r.URL.Path)) {
		case "", ".html", ".css", ".js", ".json", ".svg", ".md", ".txt":
			w.Header().Set("Content-Encoding", "gzip")
			w.Header().Add("Vary", "Accept-Encoding")
			gz := gzip.NewWriter(w)
			defer gz.Close()
			next.ServeHTTP(gzWriter{w, gz}, r)
		default:
			next.ServeHTTP(w, r)
		}
	})
}

func jsonOut(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(v)
}

// safeJoin resolves a slash path under root, rejecting escapes.
func safeJoin(root, rel string) (string, bool) {
	rel = path.Clean("/" + rel)[1:] // normalize, strip leading /
	if rel == "" || strings.HasPrefix(rel, "..") {
		return "", false
	}
	full, err := filepath.Abs(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return "", false
	}
	absRoot, _ := filepath.Abs(root)
	if full != absRoot && !strings.HasPrefix(full, absRoot+string(os.PathSeparator)) {
		return "", false
	}
	return full, true
}

func serve(root, addr string, dev bool) error {
	site := &Site{root: root}
	if err := site.rebuild(); err != nil {
		return err
	}
	maybeRebuild := func() {
		if dev && time.Since(site.built) > 500*time.Millisecond {
			_ = site.rebuild()
		}
	}

	mux := http.NewServeMux()

	mux.HandleFunc(internalRoutePrefix+"/info", func(w http.ResponseWriter, r *http.Request) {
		maybeRebuild()
		site.mu.RLock()
		defer site.mu.RUnlock()
		jsonOut(w, map[string]any{
			"name": productName, "pages": len(site.pages),
			"records": len(site.records), "built": site.built, "dev": dev,
		})
	})

	mux.HandleFunc(internalRoutePrefix+"/index.json", func(w http.ResponseWriter, r *http.Request) {
		maybeRebuild()
		site.mu.RLock()
		defer site.mu.RUnlock()
		jsonOut(w, site.records)
	})

	mux.HandleFunc(internalRoutePrefix+"/search", func(w http.ResponseWriter, r *http.Request) {
		maybeRebuild()
		jsonOut(w, map[string]any{"q": r.URL.Query().Get("q"),
			"results": site.search(r.URL.Query().Get("q"), 15)})
	})

	// HTMX partial for any page, including nested folders:
	// GET /__pustaka/partial/guide/deploy.html
	mux.HandleFunc(internalRoutePrefix+"/partial/", func(w http.ResponseWriter, r *http.Request) {
		maybeRebuild()
		rel := strings.TrimPrefix(r.URL.Path, internalRoutePrefix+"/partial/")
		if !strings.HasSuffix(rel, ".html") || strings.HasPrefix(path.Base(rel), "_") {
			http.Error(w, "not a page", http.StatusBadRequest)
			return
		}
		full, ok := safeJoin(root, rel)
		if !ok {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		b, err := os.ReadFile(full)
		if err != nil {
			http.Error(w, "page not found", http.StatusNotFound)
			return
		}
		raw := string(b)
		inner, classes, ok := mainInner(raw)
		if !ok {
			http.Error(w, `page has no <main class="doc">`, http.StatusUnprocessableEntity)
			return
		}
		id, title, layout := "", rel, ""
		if m := reDataPage.FindStringSubmatch(raw); m != nil {
			id = m[1]
		}
		if m := reTitleTag.FindStringSubmatch(raw); m != nil {
			title = stripText(m[1])
		}
		// Layout lives on <body>, which a partial swap does not replace — so the
		// client needs it explicitly, or a landing page's chrome would persist.
		if m := reDataLayout.FindStringSubmatch(raw); m != nil {
			layout = m[1]
		}
		jsonOut(w, map[string]string{"id": id, "title": title, "classes": classes,
			"layout": layout, "html": inner})
	})

	// Static files (nested paths, dirs resolve to index.html). Served
	// directly — http.FileServer's /index.html redirect fights SPA routing.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if strings.HasSuffix(p, "/") {
			p += "index.html"
		}
		full, ok := safeJoin(root, p)
		if !ok {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if fi, err := os.Stat(full); err == nil && fi.IsDir() {
			full = filepath.Join(full, "index.html")
		}
		f, err := os.Open(full)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer f.Close()
		fi, _ := f.Stat()
		if strings.HasSuffix(full, ".html") {
			w.Header().Set("Cache-Control", "no-cache")
		} else {
			w.Header().Set("Cache-Control", "public, max-age=300")
		}
		http.ServeContent(w, r, full, fi.ModTime(), f)
	})

	site.mu.RLock()
	fmt.Printf("pustaka: serving %d pages, %d search records from %s\n", len(site.pages), len(site.records), root)
	site.mu.RUnlock()
	fmt.Printf("pustaka: http://localhost%s  (dev=%v)\n", addr, dev)
	return http.ListenAndServe(addr, gzipMiddleware(mux))
}

/* ============================== check =============================== */

// prefixFor returns the relative asset prefix a page at rel must use:
// "" at root, "../" one folder deep, "../../" two deep, and so on.
func prefixFor(rel string) string {
	return strings.Repeat("../", strings.Count(rel, "/"))
}

// check validates every page against the authoring spec — the feedback
// signal in the AI authoring loop: model writes → check → fix → repeat.
func check(root string) int {
	fail, warns := 0, 0
	bad := func(f, msg string) { fail++; fmt.Printf("  ✗ %s — %s\n", f, msg) }
	warn := func(f, msg string) { warns++; fmt.Printf("  ⚠ %s — %s\n", f, msg) }

	pages, err := loadRegistry(root)
	if err != nil {
		fmt.Println("  ✗", err)
		return 1
	}
	registered := map[string]Page{}
	registryIDs := map[string]string{}
	for _, p := range pages {
		if owner, exists := registryIDs[p.ID]; exists {
			bad("registry", fmt.Sprintf("duplicates id %q (already used by %s)", p.ID, owner))
		}
		registryIDs[p.ID] = p.File
		if _, exists := registered[p.File]; exists {
			bad("registry", fmt.Sprintf("registers file %q more than once", p.File))
		}
		registered[p.File] = p
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(p.File))); err != nil {
			bad("registry", fmt.Sprintf("registers %q but the file does not exist", p.File))
		}
	}

	// Registry hygiene: parts exist, keep files small, keep the insert marker.
	for i, rf := range registryFiles(root) {
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rf)))
		if err != nil {
			bad(rf, "listed in parts but unreadable")
			continue
		}
		if n := len(strings.Split(string(b), "\n")); n > 200 {
			warn(rf, fmt.Sprintf("%d lines (> 200) — split this ToC part to keep files reviewable", n))
		}
		if i > 0 && !strings.Contains(string(b), "pustaka:insert") {
			warn(rf, "missing the `/* pustaka:insert */` marker — `pustaka new` cannot auto-register pages here")
		}
		for _, menu := range reMenuEntry.FindAllStringSubmatch(string(b), -1) {
			if owner, exists := registryIDs[menu[1]]; exists {
				bad(rf, fmt.Sprintf("menu id %q duplicates %s", menu[1], owner))
			}
			registryIDs[menu[1]] = "menu in " + rf
		}
		if reEmptyMenu.Match(b) {
			bad(rf, "menu children cannot be empty")
		}
	}

	// Walk every page, at any depth. assets/ and _-prefixed names are skipped.
	_ = filepath.WalkDir(root, func(fp string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		name := d.Name()
		if d.IsDir() {
			if name == "assets" || strings.HasPrefix(name, "_") || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(name, ".html") || strings.HasPrefix(name, "_") {
			return nil
		}
		rel := filepath.ToSlash(strings.TrimPrefix(fp, filepath.Clean(root)+string(os.PathSeparator)))
		b, err := os.ReadFile(fp)
		if err != nil {
			bad(rel, err.Error())
			return nil
		}
		s := string(b)
		p, isReg := registered[rel]
		if !isReg {
			bad(rel, "exists on disk but is not registered in the ToC (assets/toc.js or a part file)")
		}
		if !strings.Contains(strings.ToLower(s[:min(200, len(s))]), "<!doctype html>") {
			bad(rel, "missing <!DOCTYPE html>")
		}
		if !strings.Contains(s, `name="viewport"`) {
			bad(rel, "missing viewport meta (breaks phones)")
		}
		m := reDataPage.FindStringSubmatch(s)
		switch {
		case m == nil:
			bad(rel, `missing data-page="…" on <body>`)
		case isReg && m[1] != p.ID:
			bad(rel, fmt.Sprintf("data-page=%q but the registry id is %q", m[1], p.ID))
		}
		if _, _, ok := mainInner(s); !ok {
			bad(rel, `missing <main class="doc"> … </main>`)
		}
		// Optional declarative page metadata. When present, the complete contract
		// is validated so the runtime never has to guess at malformed values.
		metas := rePageMeta.FindAllStringSubmatch(s, -1)
		if len(metas) > 1 {
			bad(rel, "must contain at most one .page-meta component")
		}
		for _, meta := range metas {
			attrs := meta[1] + " " + meta[2]
			values := map[string]string{}
			for _, name := range []string{"data-tags", "data-hashtags", "data-published", "data-updated", "data-version-from"} {
				value, ok := htmlAttr(attrs, name)
				if !ok || strings.TrimSpace(value) == "" {
					bad(rel, fmt.Sprintf(".page-meta requires non-empty %s", name))
				}
				values[name] = strings.TrimSpace(value)
			}
			for _, name := range []string{"data-tags", "data-hashtags"} {
				seen := map[string]bool{}
				for _, item := range strings.Split(values[name], ",") {
					item = strings.TrimSpace(strings.TrimPrefix(item, "#"))
					if item == "" {
						bad(rel, fmt.Sprintf("%s contains an empty item", name))
						continue
					}
					key := strings.ToLower(item)
					if seen[key] {
						bad(rel, fmt.Sprintf("%s contains duplicate %q", name, item))
					}
					seen[key] = true
					if name == "data-hashtags" && !reHashtag.MatchString(item) {
						bad(rel, fmt.Sprintf("hashtag %q must be lowercase letters, digits, or hyphens", item))
					}
				}
			}
			published, pubErr := time.Parse(time.DateOnly, values["data-published"])
			updated, updErr := time.Parse(time.DateOnly, values["data-updated"])
			if pubErr != nil {
				bad(rel, "data-published must be an ISO date (YYYY-MM-DD)")
			}
			if updErr != nil {
				bad(rel, "data-updated must be an ISO date (YYYY-MM-DD)")
			}
			if pubErr == nil && updErr == nil && updated.Before(published) {
				bad(rel, "data-updated cannot be earlier than data-published")
			}
			from, fromErr := versionBound(values["data-version-from"], false)
			if fromErr != nil {
				bad(rel, "data-version-from "+fromErr.Error())
			}
			if toRaw, ok := htmlAttr(attrs, "data-version-to"); ok && strings.TrimSpace(toRaw) != "" {
				to, toErr := versionBound(strings.TrimSpace(toRaw), true)
				if toErr != nil {
					bad(rel, "data-version-to "+toErr.Error())
				} else if fromErr == nil && versionAfter(from, to) {
					bad(rel, "data-version-from cannot be later than data-version-to")
				}
			}
		}
		// Asset prefix must match the folder depth (guide/x.html → ../assets/…).
		pre := prefixFor(rel)
		for _, ref := range []string{"assets/site.css", "assets/toc.js", "assets/site.js"} {
			if !strings.Contains(s, `"`+pre+ref+`"`) {
				bad(rel, fmt.Sprintf("must reference %q (folder depth %d ⇒ prefix %q)", pre+ref, strings.Count(rel, "/"), pre))
			}
		}
		// Every internal link must resolve to a real file.
		dir := path.Dir(rel)
		for _, h := range reHref.FindAllStringSubmatch(s, -1) {
			href := h[1]
			if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
				continue
			}
			target := path.Clean(path.Join(dir, href))
			if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(target))); err != nil {
				bad(rel, fmt.Sprintf("link %q resolves to %q, which does not exist", href, target))
			}
		}
		// h2 = section: id required. h3 id optional (indexed when present).
		ids := map[string]bool{}
		for _, h := range reHead.FindAllStringSubmatch(s, -1) {
			if ids[h[2]] {
				bad(rel, fmt.Sprintf("duplicate heading id %q", h[2]))
			}
			ids[h[2]] = true
		}
		if withID, total := len(reH2WithID.FindAllString(s, -1)), len(reH2NoID.FindAllString(s, -1)); total > withID {
			bad(rel, fmt.Sprintf("%d of %d h2 headings missing an id (h2 = section: id required; h3 id optional)", total-withID, total))
		}
		return nil
	})

	if fail == 0 {
		fmt.Printf("  ✓ %d pages valid, registry consistent", len(pages))
		if warns > 0 {
			fmt.Printf(" (%d warning(s))", warns)
		}
		fmt.Println()
		return 0
	}
	fmt.Printf("  %d problem(s) found\n", fail)
	return 1
}

/* ============================== new ================================= */

// newPage stamps docs/_template.html into a (possibly nested) page and
// registers it in a ToC part at its `pustaka:insert` marker.
func newPage(root, rel, title, part, group string) error {
	rel = path.Clean(strings.TrimPrefix(rel, "/"))
	if !strings.HasSuffix(rel, ".html") {
		return fmt.Errorf("page path must end in .html, got %q", rel)
	}
	full, ok := safeJoin(root, rel)
	if !ok {
		return fmt.Errorf("path %q escapes the docs root", rel)
	}
	if _, err := os.Stat(full); err == nil {
		return fmt.Errorf("%s already exists", rel)
	}
	tpl, err := os.ReadFile(filepath.Join(root, "_template.html"))
	if err != nil {
		return fmt.Errorf("cannot read _template.html: %w", err)
	}
	id := strings.TrimSuffix(path.Base(rel), ".html")
	if title == "" {
		title = strings.ToUpper(id[:1]) + strings.ReplaceAll(id[1:], "-", " ")
	}
	if group == "" {
		group = strings.ToUpper(part[:1]) + part[1:]
	}
	s := string(tpl)
	s = strings.ReplaceAll(s, "PAGE TITLE", title)
	s = strings.ReplaceAll(s, "PAGE-ID", id)
	s = strings.ReplaceAll(s, "GROUP NAME", group)
	pre := prefixFor(rel)
	s = strings.ReplaceAll(s, `href="assets/`, `href="`+pre+`assets/`)
	s = strings.ReplaceAll(s, `src="assets/`, `src="`+pre+`assets/`)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(full, []byte(s), 0o644); err != nil {
		return err
	}

	partFile := filepath.Join(root, "assets", "toc", part+".js")
	pb, err := os.ReadFile(partFile)
	if err != nil {
		return fmt.Errorf("page written, but ToC part %s is unreadable: %w — register manually", partFile, err)
	}
	entry := fmt.Sprintf(`    {
      id: %q,
      file: %q,
      title: %q,
      desc: "TODO: one-line description for search and cards.",
      tags: [%q],
      sections: []
    },
    `, id, rel, title, id)
	marker := "/* pustaka:insert"
	i := strings.Index(string(pb), marker)
	if i < 0 {
		return fmt.Errorf("page written, but %s has no %q marker — add the entry manually", partFile, marker)
	}
	head := string(pb[:i])
	// The entry above the marker must end with a comma; add one if missing.
	if t := strings.TrimRight(head, " \t\n"); strings.HasSuffix(t, "}") {
		head = t + ",\n    "
	}
	out := head + entry + string(pb[i:])
	if err := os.WriteFile(partFile, []byte(out), 0o644); err != nil {
		return err
	}
	fmt.Printf("pustaka: created %s and registered it in assets/toc/%s.js\n", rel, part)
	fmt.Println("pustaka: fill in the desc/tags TODOs, then run: pustaka check", root)
	return nil
}

/* ============================== main ================================ */

func usage() {
	fmt.Println("usage: pustaka <serve|check|index|new> [dir] [args] [flags]")
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	if os.Args[1] == "help" || os.Args[1] == "--help" || os.Args[1] == "-h" {
		usage()
		return
	}
	cmd := os.Args[1]
	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	addr := fs.String("addr", ":8080", "listen address")
	prod := fs.Bool("prod", false, "cache pages and index in memory (no per-request rebuild)")
	title := fs.String("title", "", "new: page title")
	part := fs.String("part", "guide", "new: ToC part name (assets/toc/<part>.js)")
	group := fs.String("group", "", "new: eyebrow/group label (defaults to the part name)")

	args := os.Args[2:]
	dir := "./docs"
	var pos []string
	for len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		pos = append(pos, args[0])
		args = args[1:]
	}
	if len(pos) > 0 {
		dir = pos[0]
	}
	_ = fs.Parse(args)

	switch cmd {
	case "serve":
		if err := serve(dir, *addr, !*prod); err != nil {
			fmt.Fprintln(os.Stderr, "pustaka:", err)
			os.Exit(1)
		}
	case "check":
		fmt.Printf("pustaka check %s\n", dir)
		os.Exit(check(dir))
	case "index":
		site := &Site{root: dir}
		if err := site.rebuild(); err != nil {
			fmt.Fprintln(os.Stderr, "pustaka:", err)
			os.Exit(1)
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(site.records)
	case "new":
		if len(pos) < 2 {
			fmt.Println(`usage: pustaka new <dir> <path/to/page.html> --title "Title" [--part guide] [--group "Guide"]`)
			os.Exit(2)
		}
		if err := newPage(dir, pos[1], *title, *part, *group); err != nil {
			fmt.Fprintln(os.Stderr, "pustaka:", err)
			os.Exit(1)
		}
	default:
		fmt.Println("unknown command:", cmd)
		usage()
		os.Exit(2)
	}
}
