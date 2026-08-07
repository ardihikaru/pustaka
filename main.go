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
//
// serve can gate the site behind a single central credential taken from the
// environment (PUSTAKA_AUTH…, see the auth section). It is off unless
// PUSTAKA_AUTH is set, and when off the server behaves exactly as it always has.
package main

import (
	"compress/gzip"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	_ "embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"net/url"
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
	authRoutePrefix     = internalRoutePrefix + "/auth"
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

// authConfig is the resolved PUSTAKA_AUTH… environment. A nil *authConfig
// means the feature is off, which is the only state the rest of the engine
// has to know about.
type authConfig struct {
	User, Pass string
	Secret     []byte
	TTL        time.Duration
	CookieName string
	Secure     string // "auto" | "on" | "off"
	// PublicHome restores the wider legacy allowlist: the docs home and all
	// assets stay readable signed out. It is deliberately off by default.
	PublicHome bool
	// CredFP fingerprints the credential pair, so changing the user or the
	// password invalidates every outstanding cookie without server state.
	CredFP string
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
	// site: { … } — matched non-greedily so it stops before the product block,
	// whose own name: key would otherwise win.
	reSiteBlock   = regexp.MustCompile(`(?s)site:\s*\{(.*?)\}`)
	reSiteName    = regexp.MustCompile(`name:\s*"([^"]*)"`)
	reSiteVersion = regexp.MustCompile(`version:\s*"([^"]*)"`)
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

// siteMeta reads window.DOCS.site.{name,version} from assets/toc.js so the
// login page can name the site without loading the registry in the browser.
func siteMeta(root string) (name, version string) {
	name = productName
	b, err := os.ReadFile(filepath.Join(root, "assets", "toc.js"))
	if err != nil {
		return name, ""
	}
	block := reSiteBlock.FindStringSubmatch(string(b))
	if block == nil {
		return name, ""
	}
	if m := reSiteName.FindStringSubmatch(block[1]); m != nil && m[1] != "" {
		name = m[1]
	}
	if m := reSiteVersion.FindStringSubmatch(block[1]); m != nil {
		version = m[1]
	}
	return name, version
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

// jsonStatus writes v with an explicit status. Denials must use this: jsonOut
// commits a 200 the moment it writes a header, and site.js only checks r.ok.
func jsonStatus(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func jsonOut(w http.ResponseWriter, v any) { jsonStatus(w, http.StatusOK, v) }

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

// newHandler builds the complete route tree. Split out of serve so the
// middleware matrix is testable with httptest, without binding a port.
// A nil *auth means the login layer is absent, not merely disabled.
func newHandler(site *Site, root string, dev bool, a *auth) http.Handler {
	maybeRebuild := func() {
		if dev && time.Since(site.built) > 500*time.Millisecond {
			_ = site.rebuild()
		}
	}

	mux := http.NewServeMux()

	mux.HandleFunc(internalRoutePrefix+"/info", func(w http.ResponseWriter, r *http.Request) {
		// Public, but payload-aware: page and record counts describe the size
		// of a private site, so an anonymous caller does not get them.
		if a != nil {
			w.Header().Add("Vary", "Cookie")
			if !a.authed(r) {
				w.Header().Set("Cache-Control", "no-store")
				jsonOut(w, map[string]any{"name": productName, "auth": map[string]any{
					"enabled": true, "authenticated": false, "loginUrl": authRoutePrefix + "/login",
				}})
				return
			}
		}
		maybeRebuild()
		site.mu.RLock()
		defer site.mu.RUnlock()
		out := map[string]any{
			"name": productName, "pages": len(site.pages),
			"records": len(site.records), "built": site.built, "dev": dev,
		}
		if a != nil {
			out["auth"] = map[string]any{
				"enabled": true, "authenticated": true, "user": a.cfg.User,
				"loginUrl": authRoutePrefix + "/login", "logoutUrl": authRoutePrefix + "/logout",
			}
		}
		jsonOut(w, out)
	})

	if a != nil {
		mux.HandleFunc(authRoutePrefix+"/login", a.handleLogin)
		mux.HandleFunc(authRoutePrefix+"/logout", a.handleLogout)
	}

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
		switch {
		case strings.HasSuffix(full, ".html"):
			w.Header().Set("Cache-Control", "no-cache")
		case a != nil:
			// public would let a shared cache hand a guarded asset to an
			// anonymous client; Vary: Cookie alone does not prevent that.
			w.Header().Set("Cache-Control", "private, max-age=300")
		default:
			w.Header().Set("Cache-Control", "public, max-age=300")
		}
		http.ServeContent(w, r, full, fi.ModTime(), f)
	})

	var h http.Handler = gzipMiddleware(mux)
	if a != nil {
		// Auth wraps gzip so denials keep a real Content-Length and are not
		// gzip-framed; the login page itself is on the mux, so it still is.
		h = a.middleware(h)
	}
	return h
}

func serve(root, addr string, dev bool, a *auth) error {
	site := &Site{root: root}
	if err := site.rebuild(); err != nil {
		return err
	}
	h := newHandler(site, root, dev, a)
	site.mu.RLock()
	fmt.Printf("pustaka: serving %d pages, %d search records from %s\n", len(site.pages), len(site.records), root)
	site.mu.RUnlock()
	fmt.Printf("pustaka: http://localhost%s  (dev=%v)\n", addr, dev)
	if a != nil {
		scope := "every page, including the docs home, is gated"
		if a.cfg.PublicHome {
			scope = "everything but the docs home and assets/ is gated"
		}
		fmt.Printf("pustaka: auth ON — user %q, sessions last %s, %s\n", a.cfg.User, a.cfg.TTL, scope)
		if a.ephemeral {
			fmt.Println("pustaka: PUSTAKA_AUTH_SECRET is unset — a random key was generated, so sessions end when this process does")
		}
	}
	return http.ListenAndServe(addr, h)
}

/* ============================== auth ================================ */

// The login layer is optional and off unless PUSTAKA_AUTH is set. By default it
// gates everything, including the docs home. Set PUSTAKA_AUTH_PUBLIC_HOME=1 to
// restore the older shape where the home and all assets are public. Sessions are
// a signed cookie, so the server keeps no state.
//
// The engine serves docs/_login.html; the underscore already makes that file
// invisible to the registry, to `check`, and to the partial endpoint. The
// same file is embedded as a fallback for a docs root that lacks it.
//
//go:embed docs/_login.html
var embeddedLogin []byte

const (
	authCookieName    = "pustaka_session"
	authMaxAttempts   = 5
	authLockout       = time.Minute
	authTokenMaxLen   = 512
	authBodyMaxBytes  = 8 << 10
	authThrottleSweep = 1024
)

type auth struct {
	cfg       authConfig
	root      string
	dev       bool
	ephemeral bool // secret was generated, not configured
	thr       *throttle
	siteName  string
	siteVer   string

	mu   sync.RWMutex
	page []byte // cached login template, --prod only
}

// loadAuthConfig resolves the PUSTAKA_AUTH… environment. It returns
// (nil, nil) when the feature is off — the caller treats that as "absent".
// env is a parameter so tests never touch the process environment.
func loadAuthConfig(env func(string) string) (*authConfig, bool, error) {
	switch strings.ToLower(strings.TrimSpace(env("PUSTAKA_AUTH"))) {
	case "1", "true", "yes", "on":
	default:
		return nil, false, nil
	}
	user, pass := env("PUSTAKA_AUTH_USER"), env("PUSTAKA_AUTH_PASS")
	if user == "" {
		return nil, false, fmt.Errorf("PUSTAKA_AUTH is on but PUSTAKA_AUTH_USER is empty — set a username or unset PUSTAKA_AUTH")
	}
	if pass == "" {
		return nil, false, fmt.Errorf("PUSTAKA_AUTH is on but PUSTAKA_AUTH_PASS is empty — set a password or unset PUSTAKA_AUTH")
	}
	ttl := 12 * time.Hour
	if raw := strings.TrimSpace(env("PUSTAKA_AUTH_TTL")); raw != "" {
		d, err := time.ParseDuration(raw)
		if err != nil || d <= 0 {
			return nil, false, fmt.Errorf("PUSTAKA_AUTH_TTL %q is not a positive Go duration (try 12h, 30m, 168h)", raw)
		}
		ttl = d
	}
	secure := strings.ToLower(strings.TrimSpace(env("PUSTAKA_AUTH_SECURE")))
	switch secure {
	case "", "auto":
		secure = "auto"
	case "1", "true", "yes", "on":
		secure = "on"
	case "0", "false", "no", "off":
		secure = "off"
	default:
		return nil, false, fmt.Errorf("PUSTAKA_AUTH_SECURE %q must be auto, 1 or 0", secure)
	}
	publicHome, rawHome := false, strings.TrimSpace(env("PUSTAKA_AUTH_PUBLIC_HOME"))
	switch strings.ToLower(rawHome) {
	case "", "0", "false", "no", "off":
	case "1", "true", "yes", "on":
		publicHome = true
	default:
		return nil, false, fmt.Errorf("PUSTAKA_AUTH_PUBLIC_HOME %q must be 1 or 0", rawHome)
	}
	secret, ephemeral := []byte(env("PUSTAKA_AUTH_SECRET")), false
	if len(secret) == 0 {
		secret = make([]byte, 32)
		if _, err := rand.Read(secret); err != nil {
			return nil, false, fmt.Errorf("cannot generate a session key: %w", err)
		}
		ephemeral = true
	}
	fp := sha256.Sum256([]byte(user + "\x00" + pass))
	return &authConfig{
		User: user, Pass: pass, Secret: secret, TTL: ttl,
		CookieName: authCookieName, Secure: secure, PublicHome: publicHome,
		CredFP: hex.EncodeToString(fp[:])[:16],
	}, ephemeral, nil
}

func newAuth(cfg *authConfig, root string, dev, ephemeral bool) *auth {
	name, ver := siteMeta(root)
	return &auth{cfg: *cfg, root: root, dev: dev, ephemeral: ephemeral,
		thr: newThrottle(authMaxAttempts, authLockout), siteName: name, siteVer: ver}
}

/* ---------- session token ---------- */

// sign builds "v1.<expiry>.<credFP>.<hmac>". The payload is fixed-shape, so
// splitting on the last dot is unambiguous.
func (a *auth) sign(now time.Time) string {
	payload := "v1." + strconv.FormatInt(now.Add(a.cfg.TTL).Unix(), 10) + "." + a.cfg.CredFP
	return payload + "." + base64.RawURLEncoding.EncodeToString(a.mac(payload))
}

func (a *auth) mac(payload string) []byte {
	m := hmac.New(sha256.New, a.cfg.Secret)
	m.Write([]byte(payload))
	return m.Sum(nil)
}

// verify authenticates the signature BEFORE parsing anything, so no
// attacker-shaped bytes ever reach the field logic.
func (a *auth) verify(v string) bool {
	if v == "" || len(v) > authTokenMaxLen {
		return false
	}
	i := strings.LastIndexByte(v, '.')
	if i < 0 {
		return false
	}
	payload, raw := v[:i], v[i+1:]
	sig, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil || !hmac.Equal(a.mac(payload), sig) {
		return false
	}
	f := strings.Split(payload, ".")
	if len(f) != 3 || f[0] != "v1" {
		return false
	}
	exp, err := strconv.ParseInt(f[1], 10, 64)
	if err != nil || time.Now().Unix() >= exp {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(f[2]), []byte(a.cfg.CredFP)) == 1
}

// creds compares hashes, not the raw strings: fixed-length operands keep
// ConstantTimeCompare's length check from leaking the password length, and
// & rather than && avoids leaking which of the two fields was wrong.
func (a *auth) creds(user, pass string) bool {
	gu, wu := sha256.Sum256([]byte(user)), sha256.Sum256([]byte(a.cfg.User))
	gp, wp := sha256.Sum256([]byte(pass)), sha256.Sum256([]byte(a.cfg.Pass))
	return subtle.ConstantTimeCompare(gu[:], wu[:])&subtle.ConstantTimeCompare(gp[:], wp[:]) == 1
}

/* ---------- cookie ---------- */

func (a *auth) secure(r *http.Request) bool {
	switch a.cfg.Secure {
	case "on":
		return true
	case "off":
		return false
	}
	return r.TLS != nil
}

func (a *auth) cookie(r *http.Request, value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name: a.cfg.CookieName, Value: value, Path: "/",
		HttpOnly: true, SameSite: http.SameSiteLaxMode,
		Secure: a.secure(r), MaxAge: maxAge,
	}
}

func (a *auth) setCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, a.cookie(r, a.sign(time.Now()), int(a.cfg.TTL/time.Second)))
}

// clearCookie must mirror the set attributes or the browser keeps the old one.
func (a *auth) clearCookie(w http.ResponseWriter, r *http.Request) {
	c := a.cookie(r, "", -1)
	c.Expires = time.Unix(0, 0)
	http.SetCookie(w, c)
}

func (a *auth) authed(r *http.Request) bool {
	c, err := r.Cookie(a.cfg.CookieName)
	return err == nil && a.verify(c.Value)
}

/* ---------- middleware ---------- */

// normPath collapses .. and duplicate slashes. net/http has already
// percent-decoded the path, so /assets/%2e%2e/guide/x.html arrives here as
// /assets/../guide/x.html — classifying without this is an allowlist bypass.
func normPath(p string) string {
	if p == "" {
		return "/"
	}
	return path.Clean("/" + p)
}

// publicAsset is exactly the static surface the login template needs. Runtime,
// ToC and vendor files remain private: they disclose the protected site.
func publicAsset(p string) bool {
	switch p {
	case "/assets/fonts.css", "/assets/site.css", "/assets/login.css":
		return true
	}
	return strings.HasPrefix(p, "/assets/fonts/")
}

// isPublic is the minimal allowlist. Public-home mode deliberately makes the
// complete runtime/ToC asset tree readable because the anonymous home needs it.
func (a *auth) isPublic(p string) bool {
	switch p {
	case "/robots.txt", "/favicon.ico", "/sw.js", internalRoutePrefix + "/info":
		return true
	}
	if p == authRoutePrefix || strings.HasPrefix(p, authRoutePrefix+"/") {
		return true
	}
	if a.cfg.PublicHome {
		switch p {
		case "/", "/index.html":
			return true
		}
		return p == "/assets" || strings.HasPrefix(p, "/assets/")
	}
	return publicAsset(p)
}

// wantsHTMLNav distinguishes a browser navigation (redirect to the login page)
// from a fetch or subresource (401 JSON).
func wantsHTMLNav(r *http.Request) bool {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false // a 302 would silently downgrade the method
	}
	switch r.Header.Get("Sec-Fetch-Dest") {
	case "document", "iframe", "frame":
		return true
	case "":
		return strings.Contains(r.Header.Get("Accept"), "text/html")
	}
	return false
}

func (a *auth) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.isPublic(normPath(r.URL.Path)) {
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Add("Vary", "Cookie")
		if a.authed(r) {
			next.ServeHTTP(w, r)
			return
		}
		// The check runs before any os.Stat, so a guarded 404 and a guarded
		// 200 look identical to an anonymous client.
		w.Header().Set("Cache-Control", "no-store")
		if wantsHTMLNav(r) {
			http.Redirect(w, r, a.loginURL(r), http.StatusFound)
			return
		}
		// site.js treats any non-2xx from /partial/ as "do a full page load",
		// which this middleware then answers with the redirect above.
		jsonStatus(w, http.StatusUnauthorized, map[string]any{
			"error": "unauthorized", "login": authRoutePrefix + "/login",
		})
	})
}

func (a *auth) loginURL(r *http.Request) string {
	q := url.Values{}
	if next := safeNext(r.URL.RequestURI()); next != "/" {
		q.Set("next", next)
	}
	if len(q) == 0 {
		return authRoutePrefix + "/login"
	}
	return authRoutePrefix + "/login?" + q.Encode()
}

// safeNext reduces raw to a same-site absolute path, or "/" when it cannot.
// It is applied when the link is built, when the POST arrives, and again
// before redirecting — validating only at the first is the classic hole.
func safeNext(raw string) string {
	const fallback = "/"
	if raw == "" || len(raw) > 512 || !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") {
		return fallback
	}
	for _, c := range []byte(raw) {
		// backslash because browsers fold /\evil.com into //evil.com;
		// control bytes because they are Location header injection.
		if c == '\\' || c < 0x20 || c == 0x7f {
			return fallback
		}
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "" || u.Host != "" || u.User != nil || u.Opaque != "" {
		return fallback
	}
	p := path.Clean("/" + strings.TrimPrefix(u.Path, "/"))
	if p == authRoutePrefix || strings.HasPrefix(p, authRoutePrefix+"/") {
		return fallback // never bounce login → login
	}
	out := p
	if u.RawQuery != "" {
		out += "?" + u.RawQuery
	}
	if f := u.EscapedFragment(); f != "" {
		out += "#" + f
	}
	return out
}

/* ---------- login page ---------- */

// template prefers the on-disk page so it can be restyled without a rebuild,
// and mirrors the dev/--prod split the index already uses.
func (a *auth) template() []byte {
	read := func() []byte {
		if b, err := os.ReadFile(filepath.Join(a.root, "_login.html")); err == nil {
			return b
		}
		return embeddedLogin
	}
	if a.dev {
		return read()
	}
	a.mu.RLock()
	p := a.page
	a.mu.RUnlock()
	if p != nil {
		return p
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.page == nil {
		a.page = read()
	}
	return a.page
}

// renderLogin substitutes the sentinels. Every value is escaped: next and
// errMsg are attacker-influenced and land inside value="…" attributes.
func (a *auth) renderLogin(w http.ResponseWriter, code int, next, errMsg string, retry int) {
	homeHidden := " hidden"
	if a.cfg.PublicHome {
		homeHidden = ""
	}
	rep := strings.NewReplacer(
		"{{SITE_NAME}}", html.EscapeString(a.siteName),
		"{{SITE_VERSION}}", html.EscapeString(a.siteVer),
		// the page is served from a two-segment path, so relative asset
		// links would resolve under /__pustaka/auth/.
		"{{BASE}}", "/",
		"{{ACTION}}", authRoutePrefix+"/login",
		"{{NEXT}}", html.EscapeString(next),
		"{{ERROR}}", html.EscapeString(errMsg),
		"{{RETRY_AFTER}}", strconv.Itoa(retry),
		"{{HOME_HIDDEN}}", homeHidden,
	)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	_, _ = io.WriteString(w, rep.Replace(string(a.template())))
}

/* ---------- handlers ---------- */

func (a *auth) handleLogin(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		next := safeNext(r.URL.Query().Get("next"))
		if a.authed(r) {
			http.Redirect(w, r, next, http.StatusSeeOther)
			return
		}
		a.renderLogin(w, http.StatusOK, next, "", 0)
	case http.MethodPost:
		a.handleLoginPost(w, r)
	default:
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (a *auth) handleLoginPost(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, authBodyMaxBytes)
	asJSON := strings.HasPrefix(r.Header.Get("Content-Type"), "application/json")

	// Cheap, stateless login-CSRF mitigation alongside SameSite=Lax.
	if o := r.Header.Get("Origin"); o != "" && !sameOrigin(o, r.Host) {
		http.Error(w, "cross-site request rejected", http.StatusForbidden)
		return
	}

	var user, pass, next string
	if asJSON {
		var body struct{ User, Pass, Next string }
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			jsonStatus(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "Malformed request."})
			return
		}
		user, pass, next = body.User, body.Pass, body.Next
	} else {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "malformed form", http.StatusBadRequest)
			return
		}
		user, pass, next = r.PostFormValue("user"), r.PostFormValue("pass"), r.PostFormValue("next")
	}
	next = safeNext(next)

	ip := clientIP(r)
	if wait, ok := a.thr.allow(ip); !ok {
		secs := int(wait.Seconds()) + 1
		w.Header().Set("Retry-After", strconv.Itoa(secs))
		msg := fmt.Sprintf("Too many attempts. Try again in %d seconds.", secs)
		if asJSON {
			jsonStatus(w, http.StatusTooManyRequests,
				map[string]any{"ok": false, "error": msg, "retryAfter": secs})
			return
		}
		a.renderLogin(w, http.StatusTooManyRequests, next, msg, secs)
		return
	}

	if !a.creds(user, pass) {
		left, wait := a.thr.fail(ip)
		msg := "Incorrect username or password."
		secs := 0
		if left <= 0 {
			secs = int(wait.Seconds()) + 1
			msg = fmt.Sprintf("Too many attempts. Try again in %d seconds.", secs)
		}
		if asJSON {
			jsonStatus(w, http.StatusUnauthorized,
				map[string]any{"ok": false, "error": msg, "attemptsLeft": left, "retryAfter": secs})
			return
		}
		a.renderLogin(w, http.StatusUnauthorized, next, msg, secs)
		return
	}

	a.thr.reset(ip)
	a.setCookie(w, r)
	if asJSON {
		jsonOut(w, map[string]any{"ok": true, "next": next})
		return
	}
	http.Redirect(w, r, next, http.StatusSeeOther) // 303: POST → GET
}

func (a *auth) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	a.clearCookie(w, r)
	if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		jsonOut(w, map[string]any{"ok": true})
		return
	}
	// Do not send a signed-out visitor through a needless home → login loop.
	dest := "/"
	if !a.cfg.PublicHome {
		dest = authRoutePrefix + "/login"
	}
	http.Redirect(w, r, dest, http.StatusSeeOther)
}

func sameOrigin(origin, host string) bool {
	u, err := url.Parse(origin)
	return err == nil && u.Host == host
}

// clientIP deliberately ignores X-Forwarded-For: honouring it lets an attacker
// rotate the header to escape throttling and to lock out a victim's address.
func clientIP(r *http.Request) string {
	if h, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return h
	}
	return r.RemoteAddr
}

/* ---------- brute-force throttle ---------- */

type attempts struct {
	n     int
	until time.Time
	seen  time.Time
}

type throttle struct {
	mu      sync.Mutex
	max     int
	lockout time.Duration
	now     func() time.Time // injectable so lockout is testable
	m       map[string]*attempts
}

func newThrottle(max int, lockout time.Duration) *throttle {
	return &throttle{max: max, lockout: lockout, now: time.Now, m: map[string]*attempts{}}
}

// allow reports whether ip may try again, and how long is left if not.
func (t *throttle) allow(ip string) (time.Duration, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := t.now()
	e := t.m[ip]
	if e == nil {
		return 0, true
	}
	e.seen = now
	if now.Before(e.until) {
		return e.until.Sub(now), false
	}
	return 0, true
}

// fail records a failure and returns the attempts left before lockout.
func (t *throttle) fail(ip string) (left int, retryAfter time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := t.now()
	t.sweep(now)
	e := t.m[ip]
	if e == nil {
		e = &attempts{}
		t.m[ip] = e
	}
	e.seen = now
	e.n++
	if e.n >= t.max {
		e.n = 0
		e.until = now.Add(t.lockout)
		return 0, t.lockout
	}
	return t.max - e.n, 0
}

func (t *throttle) reset(ip string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.m, ip)
}

// sweep keeps the map from growing one permanent entry per source address.
func (t *throttle) sweep(now time.Time) {
	if len(t.m) < authThrottleSweep {
		return
	}
	cutoff := now.Add(-2 * t.lockout)
	for ip, e := range t.m {
		if e.seen.Before(cutoff) && e.until.Before(now) {
			delete(t.m, ip)
		}
	}
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
		cfg, ephemeral, err := loadAuthConfig(os.Getenv)
		if err != nil {
			fmt.Fprintln(os.Stderr, "pustaka:", err)
			os.Exit(1)
		}
		var a *auth
		if cfg != nil {
			a = newAuth(cfg, dir, !*prod, ephemeral)
		}
		if err := serve(dir, *addr, !*prod, a); err != nil {
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
