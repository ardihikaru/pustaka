package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeCheckFixture(t *testing.T, meta string) string {
	t.Helper()
	root := t.TempDir()
	registry := `window.DOCS = { nav: [{ group: "Guide", pages: [
  { id: "menu", title: "Menu", children: [
    { id: "sample", file: "sample.html", title: "Sample", desc: "Sample page.", tags: ["sample"], sections: [] }
  ] }
] }] };`
	page := fmt.Sprintf(`<!DOCTYPE html><html><head>
<meta name="viewport" content="width=device-width">
<link rel="stylesheet" href="assets/site.css"></head>
<body data-page="sample"><main class="doc"><h1>Sample</h1>%s
<h2 id="section">Section</h2></main>
<script src="assets/toc.js"></script><script src="assets/site.js"></script></body></html>`, meta)
	if err := os.MkdirAll(filepath.Join(root, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "assets", "toc.js"), []byte(registry), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "sample.html"), []byte(page), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestCheckPageMetadata(t *testing.T) {
	valid := `<div class="page-meta" data-tags="guide, reference" data-hashtags="guide, api-docs" data-published="2026-07-31" data-updated="2026-08-01" data-version-from="0.6.0" data-version-to="0.8.x"></div>`
	tests := []struct {
		name string
		meta string
		want int
	}{
		{name: "omitted", want: 0},
		{name: "valid bounded", meta: valid, want: 0},
		{name: "valid open", meta: `<div class="page-meta" data-tags="guide" data-hashtags="guide" data-published="2026-07-31" data-updated="2026-07-31" data-version-from="0.6.0" data-version-to=""></div>`, want: 0},
		{name: "updated before published", meta: `<div class="page-meta" data-tags="guide" data-hashtags="guide" data-published="2026-08-02" data-updated="2026-08-01" data-version-from="0.6.0"></div>`, want: 1},
		{name: "bad hashtag", meta: `<div class="page-meta" data-tags="guide" data-hashtags="Not Valid" data-published="2026-07-31" data-updated="2026-07-31" data-version-from="0.6.0"></div>`, want: 1},
		{name: "reversed version", meta: `<div class="page-meta" data-tags="guide" data-hashtags="guide" data-published="2026-07-31" data-updated="2026-07-31" data-version-from="1.0.0" data-version-to="0.9.x"></div>`, want: 1},
		{name: "duplicate item", meta: `<div class="page-meta" data-tags="guide, Guide" data-hashtags="guide" data-published="2026-07-31" data-updated="2026-07-31" data-version-from="0.6.0"></div>`, want: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := check(writeCheckFixture(t, tt.meta))
			if got != tt.want {
				t.Fatalf("check() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestVersionBound(t *testing.T) {
	from, err := versionBound("0.6.0", false)
	if err != nil {
		t.Fatal(err)
	}
	to, err := versionBound("0.8.x", true)
	if err != nil {
		t.Fatal(err)
	}
	if versionAfter(from, to) {
		t.Fatal("0.6.0 must not be after 0.8.x")
	}
	if _, err := versionBound("0.x.1", true); err == nil {
		t.Fatal("numeric component after wildcard must fail")
	}
}

func TestPustakaIdentity(t *testing.T) {
	if productName != "pustaka" {
		t.Fatalf("productName = %q, want pustaka", productName)
	}
	if internalRoutePrefix != "/__pustaka" {
		t.Fatalf("internalRoutePrefix = %q, want /__pustaka", internalRoutePrefix)
	}
}

/* ============================== auth ================================ */

const (
	testUser   = "admin"
	testPass   = "s3cret-pass"
	testSecret = "fixed-test-secret-0123456789"
)

// writeServeFixture builds a servable docs root: a registry, a public landing
// page, one guarded page, assets, and a login template.
func writeServeFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mk := func(rel, body string) {
		full := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mk("assets/toc.js", `window.DOCS = {
  site: { name: "Fixture Docs", tagline: "t", version: "v9.9.9" },
  product: { name: "Other Product", semver: "9.9.9" },
  nav: [{ group: "Guide", pages: [
    { id: "home", file: "index.html", title: "Home", desc: "Landing.", tags: ["home"], sections: [] },
    { id: "x", file: "guide/x.html", title: "X", desc: "Guarded page.", tags: ["x"], sections: [] }
  ] }] };`)
	page := func(id, pre string) string {
		return `<!DOCTYPE html><html><head><meta name="viewport" content="width=device-width">
<link rel="stylesheet" href="` + pre + `assets/site.css"></head>
<body data-page="` + id + `"><main class="doc"><h1>` + id + `</h1><h2 id="s">Section</h2></main>
<script src="` + pre + `assets/toc.js"></script><script src="` + pre + `assets/site.js"></script></body></html>`
	}
	mk("index.html", page("home", ""))
	mk("guide/x.html", page("x", "../"))
	mk("assets/site.js", "/* runtime */")
	mk("assets/site.css", "/* css */")
	mk("assets/fonts.css", "@font-face{src:url('fonts/x.woff2')}")
	mk("assets/login.css", "/* login css */")
	mk("assets/fonts/x.woff2", "woff2")
	mk("assets/toc/guide.js", "/* toc part */")
	mk("assets/vendor/echarts.min.js", "/* vendor */")
	mk("sw.js", "/* kill switch */")
	mk("_template.html", page("tpl", ""))
	mk("_login.html", `<!DOCTYPE html><title>Sign in {{SITE_NAME}} {{SITE_VERSION}}</title>
<link rel="stylesheet" href="{{BASE}}assets/site.css">
<form method="post" action="{{ACTION}}"><input name="next" value="{{NEXT}}">
<p data-error="{{ERROR}}" data-retry="{{RETRY_AFTER}}">{{ERROR}}</p></form>
<a href="{{BASE}}"{{HOME_HIDDEN}}>home</a>`)
	return root
}

func testAuth(t *testing.T, root string, over map[string]string) *auth {
	t.Helper()
	env := map[string]string{
		"PUSTAKA_AUTH":        "1",
		"PUSTAKA_AUTH_USER":   testUser,
		"PUSTAKA_AUTH_PASS":   testPass,
		"PUSTAKA_AUTH_SECRET": testSecret,
	}
	for k, v := range over {
		if v == "" {
			delete(env, k)
			continue
		}
		env[k] = v
	}
	cfg, ephemeral, err := loadAuthConfig(func(k string) string { return env[k] })
	if err != nil {
		t.Fatalf("loadAuthConfig: %v", err)
	}
	if cfg == nil {
		t.Fatal("auth unexpectedly disabled")
	}
	return newAuth(cfg, root, true, ephemeral)
}

func testHandler(t *testing.T, root string, a *auth) http.Handler {
	t.Helper()
	site := &Site{root: root}
	if err := site.rebuild(); err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	return newHandler(site, root, true, a)
}

func TestLoadAuthConfig(t *testing.T) {
	base := map[string]string{
		"PUSTAKA_AUTH": "1", "PUSTAKA_AUTH_USER": testUser, "PUSTAKA_AUTH_PASS": testPass,
	}
	with := func(over map[string]string) func(string) string {
		env := map[string]string{}
		for k, v := range base {
			env[k] = v
		}
		for k, v := range over {
			if v == "" {
				delete(env, k)
				continue
			}
			env[k] = v
		}
		return func(k string) string { return env[k] }
	}

	if cfg, _, err := loadAuthConfig(func(string) string { return "" }); cfg != nil || err != nil {
		t.Fatalf("unset PUSTAKA_AUTH = (%v, %v), want (nil, nil)", cfg, err)
	}
	if cfg, _, err := loadAuthConfig(with(map[string]string{"PUSTAKA_AUTH": "false"})); cfg != nil || err != nil {
		t.Fatal("PUSTAKA_AUTH=false must disable the feature")
	}
	for _, missing := range []string{"PUSTAKA_AUTH_USER", "PUSTAKA_AUTH_PASS"} {
		if _, _, err := loadAuthConfig(with(map[string]string{missing: ""})); err == nil {
			t.Fatalf("enabled without %s must fail fast", missing)
		}
	}
	if _, _, err := loadAuthConfig(with(map[string]string{"PUSTAKA_AUTH_TTL": "yesterday"})); err == nil {
		t.Fatal("a malformed TTL must fail fast")
	}
	if _, _, err := loadAuthConfig(with(map[string]string{"PUSTAKA_AUTH_TTL": "-5m"})); err == nil {
		t.Fatal("a negative TTL must fail fast")
	}
	if _, _, err := loadAuthConfig(with(map[string]string{"PUSTAKA_AUTH_SECURE": "maybe"})); err == nil {
		t.Fatal("an unknown SECURE mode must fail fast")
	}
	if _, _, err := loadAuthConfig(with(map[string]string{"PUSTAKA_AUTH_PUBLIC_HOME": "maybe"})); err == nil {
		t.Fatal("an unknown PUBLIC_HOME value must fail fast")
	}
	for _, on := range []string{"1", "true", "yes", "on", "ON", " Yes "} {
		cfg, _, err := loadAuthConfig(with(map[string]string{"PUSTAKA_AUTH_PUBLIC_HOME": on}))
		if err != nil || !cfg.PublicHome {
			t.Fatalf("PUBLIC_HOME=%q did not enable public home (err=%v)", on, err)
		}
	}
	for _, off := range []string{"0", "false", "no", "off"} {
		cfg, _, err := loadAuthConfig(with(map[string]string{"PUSTAKA_AUTH_PUBLIC_HOME": off}))
		if err != nil || cfg.PublicHome {
			t.Fatalf("PUBLIC_HOME=%q did not gate the home (err=%v)", off, err)
		}
	}

	cfg, ephemeral, err := loadAuthConfig(with(nil))
	if err != nil {
		t.Fatal(err)
	}
	if !ephemeral {
		t.Fatal("an unset secret must be reported as ephemeral")
	}
	if len(cfg.Secret) != 32 {
		t.Fatalf("generated secret is %d bytes, want 32", len(cfg.Secret))
	}
	if cfg.TTL != 12*time.Hour || cfg.Secure != "auto" || cfg.CookieName != authCookieName {
		t.Fatalf("defaults wrong: ttl=%v secure=%q cookie=%q", cfg.TTL, cfg.Secure, cfg.CookieName)
	}
	if cfg.CredFP == "" {
		t.Fatal("credential fingerprint must be set")
	}
	if cfg.PublicHome {
		t.Fatal("the home must be gated by default")
	}
}

func TestSessionToken(t *testing.T) {
	root := writeServeFixture(t)
	a := testAuth(t, root, nil)
	good := a.sign(time.Now())

	if !a.verify(good) {
		t.Fatal("a freshly signed token must verify")
	}
	if a.verify(a.sign(time.Now().Add(-13 * time.Hour))) {
		t.Fatal("an expired token must not verify")
	}

	flip := func(s string, i int) string {
		b := []byte(s)
		if b[i] == 'a' {
			b[i] = 'b'
		} else {
			b[i] = 'a'
		}
		return string(b)
	}
	bad := map[string]string{
		"empty":            "",
		"no dot":           "notatoken",
		"tampered body":    flip(good, 4),
		"tampered sig":     flip(good, len(good)-2),
		"truncated sig":    good[:len(good)-4],
		"oversized":        strings.Repeat("a", 600),
		"trailing garbage": good + "x",
	}
	for name, v := range bad {
		if a.verify(v) {
			t.Errorf("%s must not verify", name)
		}
	}

	other := testAuth(t, root, map[string]string{"PUSTAKA_AUTH_SECRET": "a-completely-different-key"})
	if other.verify(good) {
		t.Error("a token signed with another secret must not verify")
	}
	// Rotating the credentials must invalidate outstanding cookies.
	rotated := testAuth(t, root, map[string]string{"PUSTAKA_AUTH_PASS": "new-password"})
	if rotated.verify(good) {
		t.Error("changing the password must invalidate existing sessions")
	}
}

func TestAuthCreds(t *testing.T) {
	a := testAuth(t, writeServeFixture(t), nil)
	if !a.creds(testUser, testPass) {
		t.Fatal("the configured pair must authenticate")
	}
	for _, c := range [][2]string{
		{testUser, "wrong"}, {"wrong", testPass}, {"", ""},
		{testUser, testPass + " "}, {strings.ToUpper(testUser), testPass},
	} {
		if a.creds(c[0], c[1]) {
			t.Errorf("creds(%q, %q) must fail", c[0], c[1])
		}
	}
}

func TestAuthCookieSecurity(t *testing.T) {
	root := writeServeFixture(t)
	for _, tt := range []struct {
		name   string
		over   map[string]string
		tls    bool
		secure bool
	}{
		{name: "forced secure", over: map[string]string{"PUSTAKA_AUTH_SECURE": "1"}, secure: true},
		{name: "forced insecure", over: map[string]string{"PUSTAKA_AUTH_SECURE": "0"}, tls: true},
		{name: "auto over TLS", tls: true, secure: true},
		{name: "auto over HTTP"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			a := testAuth(t, root, tt.over)
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.tls {
				r.TLS = &tls.ConnectionState{}
			}
			w := httptest.NewRecorder()
			a.setCookie(w, r)
			got := strings.Contains(w.Header().Get("Set-Cookie"), "; Secure")
			if got != tt.secure {
				t.Errorf("Secure=%v, want %v (%q)", got, tt.secure, w.Header().Get("Set-Cookie"))
			}
		})
	}
}

func TestSafeNext(t *testing.T) {
	tests := []struct{ in, want string }{
		{"", "/"},
		{"/", "/"},
		{"/guide/x.html", "/guide/x.html"},
		{"/guide/x.html?tag=a", "/guide/x.html?tag=a"},
		{"/guide/x.html?tag=a#sec", "/guide/x.html?tag=a#sec"},
		{"//evil.com", "/"},
		{"/\\evil.com", "/"},
		{"https://evil.com/x", "/"},
		{"javascript:alert(1)", "/"},
		{"guide/x.html", "/"},
		{"/x\r\nSet-Cookie: a=b", "/"},
		{"/x\x00y", "/"},
		{authRoutePrefix + "/login", "/"},
		{authRoutePrefix + "/logout", "/"},
		{"/" + strings.Repeat("a", 600), "/"},
		{"/a/../../etc/passwd", "/etc/passwd"}, // same-site; safeJoin guards the disk
	}
	for _, tt := range tests {
		if got := safeNext(tt.in); got != tt.want {
			t.Errorf("safeNext(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestAuthMiddlewareMatrix(t *testing.T) {
	root := writeServeFixture(t)
	a := testAuth(t, root, nil)
	h := testHandler(t, root, a)

	valid := &http.Cookie{Name: a.cfg.CookieName, Value: a.sign(time.Now())}
	expired := &http.Cookie{Name: a.cfg.CookieName, Value: a.sign(time.Now().Add(-13 * time.Hour))}

	doc := map[string]string{"Sec-Fetch-Dest": "document"}
	xhr := map[string]string{"Sec-Fetch-Dest": "empty"}

	tests := []struct {
		name     string
		method   string
		target   string
		headers  map[string]string
		cookie   *http.Cookie
		want     int
		location string
	}{
		{name: "home is gated", method: "GET", target: "/", headers: doc, want: 302, location: authRoutePrefix + "/login"},
		{name: "index is gated", method: "GET", target: "/index.html", headers: doc, want: 302, location: "next=%2Findex.html"},
		{name: "signed in home renders", method: "GET", target: "/", headers: doc, cookie: valid, want: 200},
		{name: "runtime is gated", method: "GET", target: "/assets/site.js", headers: xhr, want: 401},
		{name: "toc is gated", method: "GET", target: "/assets/toc.js", headers: xhr, want: 401},
		{name: "vendor is gated", method: "GET", target: "/assets/vendor/echarts.min.js", headers: xhr, want: 401},
		{name: "login stylesheet is public", method: "GET", target: "/assets/site.css", headers: xhr, want: 200},
		{name: "login css is public", method: "GET", target: "/assets/login.css", headers: xhr, want: 200},
		{name: "font css is public", method: "GET", target: "/assets/fonts.css", headers: xhr, want: 200},
		{name: "font is public", method: "GET", target: "/assets/fonts/x.woff2", headers: xhr, want: 200},
		{name: "service-worker kill switch is public", method: "GET", target: "/sw.js", headers: xhr, want: 200},
		{name: "font traversal fails", method: "GET", target: "/assets/fonts/../../guide/x.html", headers: doc, want: 302},
		{name: "guarded page redirects", method: "GET", target: "/guide/x.html", headers: doc,
			want: 302, location: authRoutePrefix + "/login?next=%2Fguide%2Fx.html"},
		{name: "query survives in next", method: "GET", target: "/guide/x.html?tag=a", headers: doc,
			want: 302, location: "next=%2Fguide%2Fx.html%3Ftag%3Da"},
		{name: "valid cookie passes", method: "GET", target: "/guide/x.html", headers: doc, cookie: valid, want: 200},
		{name: "expired cookie redirects", method: "GET", target: "/guide/x.html", headers: doc, cookie: expired, want: 302},
		{name: "post is never redirected", method: "POST", target: "/guide/x.html", headers: doc, want: 401},
		{name: "missing page still redirects", method: "GET", target: "/guide/nope.html", headers: doc, want: 302},
		{name: "partial is 401", method: "GET", target: internalRoutePrefix + "/partial/guide/x.html", headers: xhr, want: 401},
		{name: "index.json is 401", method: "GET", target: internalRoutePrefix + "/index.json", headers: xhr, want: 401},
		{name: "search is 401", method: "GET", target: internalRoutePrefix + "/search?q=a", headers: xhr, want: 401},
		{name: "login template never leaks", method: "GET", target: "/_login.html", headers: doc, want: 302},
		{name: "page template never leaks", method: "GET", target: "/_template.html", headers: doc, want: 302},
		{name: "traversal cannot ride the allowlist", method: "GET", target: "/assets/../guide/x.html", headers: doc, want: 302},
		{name: "curl gets json not a redirect", method: "GET", target: "/guide/x.html",
			headers: map[string]string{"Accept": "*/*"}, want: 401},
		{name: "old browser accept header redirects", method: "GET", target: "/guide/x.html",
			headers: map[string]string{"Accept": "text/html,application/xhtml+xml"}, want: 302},
		{name: "login page itself is reachable", method: "GET", target: authRoutePrefix + "/login", headers: doc, want: 200},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.method, tt.target, nil)
			for k, v := range tt.headers {
				r.Header.Set(k, v)
			}
			if tt.cookie != nil {
				r.AddCookie(tt.cookie)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != tt.want {
				t.Fatalf("%s %s = %d, want %d", tt.method, tt.target, w.Code, tt.want)
			}
			if tt.location != "" && !strings.Contains(w.Header().Get("Location"), tt.location) {
				t.Fatalf("Location = %q, want it to contain %q", w.Header().Get("Location"), tt.location)
			}
			if w.Code == 302 || w.Code == 401 {
				if w.Header().Get("Cache-Control") != "no-store" {
					t.Errorf("a denial must not be cacheable, got %q", w.Header().Get("Cache-Control"))
				}
			}
		})
	}

	// There is no useful return target beyond the gated root, so do not create
	// a login → home → login loop or leak a needless next query.
	t.Run("home redirect has no next", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("Sec-Fetch-Dest", "document")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if got := w.Header().Get("Location"); got != authRoutePrefix+"/login" {
			t.Errorf("home redirect = %q, want %q", got, authRoutePrefix+"/login")
		}
	})
}

func TestAuthInfoPayload(t *testing.T) {
	root := writeServeFixture(t)
	a := testAuth(t, root, nil)
	h := testHandler(t, root, a)

	get := func(cookie *http.Cookie) map[string]any {
		r := httptest.NewRequest("GET", internalRoutePrefix+"/info", nil)
		if cookie != nil {
			r.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 200 {
			t.Fatalf("/info = %d, want 200", w.Code)
		}
		var out map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		return out
	}

	anon := get(nil)
	if _, leaked := anon["records"]; leaked {
		t.Error("an anonymous caller must not learn the record count")
	}
	auth, _ := anon["auth"].(map[string]any)
	if auth == nil || auth["enabled"] != true || auth["authenticated"] != false {
		t.Fatalf("anonymous auth block wrong: %v", anon["auth"])
	}

	in := get(&http.Cookie{Name: a.cfg.CookieName, Value: a.sign(time.Now())})
	if _, ok := in["records"]; !ok {
		t.Error("an authenticated caller should get the full payload")
	}
	auth, _ = in["auth"].(map[string]any)
	if auth == nil || auth["authenticated"] != true || auth["user"] != testUser {
		t.Fatalf("authenticated auth block wrong: %v", in["auth"])
	}
}

func TestAuthPublicHomeMode(t *testing.T) {
	root := writeServeFixture(t)
	a := testAuth(t, root, map[string]string{"PUSTAKA_AUTH_PUBLIC_HOME": "1"})
	h := testHandler(t, root, a)
	get := func(target string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodGet, target, nil)
		r.Header.Set("Sec-Fetch-Dest", "document")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	for _, target := range []string{"/", "/index.html", "/assets/site.js", "/assets/toc.js", "/assets/vendor/echarts.min.js"} {
		if w := get(target); w.Code != http.StatusOK {
			t.Errorf("GET %s = %d, want 200", target, w.Code)
		}
	}
	if w := get("/guide/x.html"); w.Code != http.StatusFound {
		t.Errorf("private page = %d, want 302", w.Code)
	}

	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, authRoutePrefix+"/logout", nil))
	if got := w.Header().Get("Location"); got != "/" {
		t.Errorf("public-home logout = %q, want /", got)
	}
	if body := get(authRoutePrefix + "/login").Body.String(); !strings.Contains(body, `<a href="/">`) || strings.Contains(body, `hidden`) {
		t.Errorf("public-home login should expose a usable home link: %q", body)
	}
}

func TestLoginPageHidesHomeLinkWhenGated(t *testing.T) {
	root := writeServeFixture(t)
	h := testHandler(t, root, testAuth(t, root, nil))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, authRoutePrefix+"/login", nil))
	if body := w.Body.String(); !strings.Contains(body, `<a href="/" hidden>`) {
		t.Errorf("gated login must hide the home link: %q", body)
	}
	for name, src := range map[string][]byte{"embedded": embeddedLogin, "docs/_login.html": mustRead(t, filepath.Join("docs", "_login.html"))} {
		if !strings.Contains(string(src), "{{HOME_HIDDEN}}") {
			t.Errorf("%s is missing {{HOME_HIDDEN}}", name)
		}
	}
}

func mustRead(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestAuthDisabledPassthrough(t *testing.T) {
	root := writeServeFixture(t)
	h := testHandler(t, root, nil)

	for _, target := range []string{
		"/guide/x.html", "/index.html",
		internalRoutePrefix + "/index.json",
		internalRoutePrefix + "/partial/guide/x.html",
	} {
		r := httptest.NewRequest("GET", target, nil)
		r.Header.Set("Sec-Fetch-Dest", "document")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 200 {
			t.Errorf("with auth off, %s = %d, want 200", target, w.Code)
		}
	}

	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", authRoutePrefix+"/login", nil))
	if w.Code != 404 {
		t.Errorf("with auth off the login route must not exist, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", internalRoutePrefix+"/info", nil))
	var info map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &info); err != nil {
		t.Fatal(err)
	}
	if _, present := info["auth"]; present {
		t.Error("with auth off /info must not grow an auth key")
	}
}

func TestLoginHandlers(t *testing.T) {
	root := writeServeFixture(t)
	fresh := func() (http.Handler, *auth) {
		a := testAuth(t, root, nil)
		return testHandler(t, root, a), a
	}

	t.Run("get renders the page", func(t *testing.T) {
		h, _ := fresh()
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", authRoutePrefix+"/login", nil))
		body := w.Body.String()
		if w.Code != 200 {
			t.Fatalf("status %d", w.Code)
		}
		if strings.Contains(body, "{{") {
			t.Error("placeholders were left unsubstituted")
		}
		if !strings.Contains(body, "Fixture Docs") || !strings.Contains(body, `action="`+authRoutePrefix+`/login"`) {
			t.Errorf("page did not render as expected: %s", body)
		}
		if w.Header().Get("Cache-Control") != "no-store" {
			t.Error("the login page must not be cached")
		}
	})

	t.Run("get while signed in redirects", func(t *testing.T) {
		h, a := fresh()
		r := httptest.NewRequest("GET", authRoutePrefix+"/login?next=/guide/x.html", nil)
		r.AddCookie(&http.Cookie{Name: a.cfg.CookieName, Value: a.sign(time.Now())})
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 303 || w.Header().Get("Location") != "/guide/x.html" {
			t.Fatalf("got %d → %q", w.Code, w.Header().Get("Location"))
		}
	})

	post := func(h http.Handler, body, ctype string, hdr map[string]string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", authRoutePrefix+"/login", strings.NewReader(body))
		r.Header.Set("Content-Type", ctype)
		for k, v := range hdr {
			r.Header.Set(k, v)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}

	t.Run("form login succeeds", func(t *testing.T) {
		h, _ := fresh()
		w := post(h, "user="+testUser+"&pass="+testPass+"&next=/guide/x.html",
			"application/x-www-form-urlencoded", nil)
		if w.Code != 303 || w.Header().Get("Location") != "/guide/x.html" {
			t.Fatalf("got %d → %q", w.Code, w.Header().Get("Location"))
		}
		sc := w.Header().Get("Set-Cookie")
		for _, want := range []string{authCookieName + "=", "Path=/", "HttpOnly", "SameSite=Lax", "Max-Age="} {
			if !strings.Contains(sc, want) {
				t.Errorf("Set-Cookie %q is missing %q", sc, want)
			}
		}
	})

	t.Run("form login rejects a hostile next", func(t *testing.T) {
		h, _ := fresh()
		w := post(h, "user="+testUser+"&pass="+testPass+"&next=https%3A%2F%2Fevil.com",
			"application/x-www-form-urlencoded", nil)
		if got := w.Header().Get("Location"); got != "/" {
			t.Fatalf("Location = %q, want /", got)
		}
	})

	t.Run("form login rejects bad credentials", func(t *testing.T) {
		h, _ := fresh()
		w := post(h, "user="+testUser+"&pass=nope", "application/x-www-form-urlencoded", nil)
		if w.Code != 401 {
			t.Fatalf("status %d, want 401", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Incorrect username or password") {
			t.Error("the rendered page should carry the error")
		}
		if strings.Contains(w.Header().Get("Set-Cookie"), authCookieName) {
			t.Error("a failed login must not set a session cookie")
		}
	})

	t.Run("json login succeeds", func(t *testing.T) {
		h, _ := fresh()
		w := post(h, `{"user":"`+testUser+`","pass":"`+testPass+`","next":"/guide/x.html"}`,
			"application/json", nil)
		if w.Code != 200 {
			t.Fatalf("status %d", w.Code)
		}
		var body map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body["ok"] != true || body["next"] != "/guide/x.html" {
			t.Fatalf("body = %v", body)
		}
	})

	t.Run("json login reports attempts left", func(t *testing.T) {
		h, _ := fresh()
		w := post(h, `{"user":"`+testUser+`","pass":"nope"}`, "application/json", nil)
		if w.Code != 401 {
			t.Fatalf("status %d, want 401", w.Code)
		}
		var body map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body["ok"] != false {
			t.Fatalf("body = %v", body)
		}
		if left, _ := body["attemptsLeft"].(float64); left != float64(authMaxAttempts-1) {
			t.Fatalf("attemptsLeft = %v, want %d", body["attemptsLeft"], authMaxAttempts-1)
		}
	})

	t.Run("cross-site origin is rejected", func(t *testing.T) {
		h, _ := fresh()
		w := post(h, `{"user":"`+testUser+`","pass":"`+testPass+`"}`, "application/json",
			map[string]string{"Origin": "https://evil.example"})
		if w.Code != 403 {
			t.Fatalf("status %d, want 403", w.Code)
		}
	})

	t.Run("same-site origin is accepted", func(t *testing.T) {
		h, _ := fresh()
		w := post(h, `{"user":"`+testUser+`","pass":"`+testPass+`"}`, "application/json",
			map[string]string{"Origin": "http://example.com"})
		if w.Code != 200 {
			t.Fatalf("status %d, want 200", w.Code)
		}
	})

	t.Run("logout clears the cookie", func(t *testing.T) {
		h, _ := fresh()
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", authRoutePrefix+"/logout", nil))
		if w.Code != 405 {
			t.Fatalf("GET logout = %d, want 405", w.Code)
		}
		w = httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("POST", authRoutePrefix+"/logout", nil))
		if w.Code != 303 {
			t.Fatalf("POST logout = %d, want 303", w.Code)
		}
		if got := w.Header().Get("Location"); got != authRoutePrefix+"/login" {
			t.Errorf("gated logout Location = %q, want login", got)
		}
		if sc := w.Header().Get("Set-Cookie"); !strings.Contains(sc, "Max-Age=0") {
			t.Errorf("logout must expire the cookie, got %q", sc)
		}
	})

	t.Run("lockout after repeated failures", func(t *testing.T) {
		h, _ := fresh()
		var last *httptest.ResponseRecorder
		for i := 0; i < authMaxAttempts+1; i++ {
			last = post(h, `{"user":"`+testUser+`","pass":"nope"}`, "application/json", nil)
		}
		if last.Code != http.StatusTooManyRequests {
			t.Fatalf("status %d, want 429", last.Code)
		}
		if last.Header().Get("Retry-After") == "" {
			t.Error("a lockout must tell the client how long to wait")
		}
	})
}

func TestThrottleLockout(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	th := newThrottle(3, time.Minute)
	th.now = func() time.Time { return now }

	if left, _ := th.fail("ip"); left != 2 {
		t.Fatalf("first failure left %d, want 2", left)
	}
	if left, _ := th.fail("ip"); left != 1 {
		t.Fatalf("second failure left %d, want 1", left)
	}
	left, wait := th.fail("ip")
	if left != 0 || wait != time.Minute {
		t.Fatalf("third failure = (%d, %v), want (0, 1m)", left, wait)
	}
	if _, ok := th.allow("ip"); ok {
		t.Fatal("a locked address must not be allowed")
	}
	if _, ok := th.allow("other"); !ok {
		t.Fatal("the lockout must be per address")
	}

	now = now.Add(61 * time.Second)
	if _, ok := th.allow("ip"); !ok {
		t.Fatal("the lockout must expire")
	}

	th.fail("ip")
	th.reset("ip")
	if _, ok := th.allow("ip"); !ok {
		t.Fatal("reset must clear the record")
	}
}

func TestThrottleSweep(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	th := newThrottle(3, time.Minute)
	th.now = func() time.Time { return now }
	for i := 0; i < authThrottleSweep; i++ {
		th.fail(fmt.Sprintf("10.0.%d.%d", i/256, i%256))
	}
	now = now.Add(10 * time.Minute)
	th.fail("192.0.2.9")
	if len(th.m) != 1 {
		t.Fatalf("sweep left %d entries, want 1 — the map must not grow without bound", len(th.m))
	}
}

func TestLoginPageFallbackAndEscaping(t *testing.T) {
	t.Run("embedded fallback", func(t *testing.T) {
		bare := t.TempDir()
		a := testAuth(t, bare, nil)
		if !bytes.Contains(a.template(), []byte("lg-stage")) {
			t.Fatal("a docs root without _login.html must fall back to the embedded page")
		}
	})

	t.Run("on-disk page wins", func(t *testing.T) {
		a := testAuth(t, writeServeFixture(t), nil)
		if !bytes.Contains(a.template(), []byte("{{SITE_NAME}}")) {
			t.Fatal("the on-disk template should be preferred")
		}
	})

	t.Run("every placeholder is escaped", func(t *testing.T) {
		a := testAuth(t, writeServeFixture(t), nil)
		w := httptest.NewRecorder()
		a.renderLogin(w, 401, `/x"onload=alert(1)`, `<script>alert(1)</script>`, 42)
		body := w.Body.String()
		if strings.Contains(body, "<script>alert(1)</script>") {
			t.Error("the error message must be HTML-escaped")
		}
		if !strings.Contains(body, "&lt;script&gt;") {
			t.Errorf("expected an escaped error, got: %s", body)
		}
		if strings.Contains(body, `value="/x"onload=`) {
			t.Error("next must not break out of its attribute")
		}
		if !strings.Contains(body, `data-retry="42"`) {
			t.Error("the retry countdown should reach the page")
		}
	})
}

func TestSiteMeta(t *testing.T) {
	name, version := siteMeta(writeServeFixture(t))
	if name != "Fixture Docs" {
		t.Errorf("name = %q, want Fixture Docs (product.name must not win)", name)
	}
	if version != "v9.9.9" {
		t.Errorf("version = %q, want v9.9.9", version)
	}
	if name, version := siteMeta(t.TempDir()); name != productName || version != "" {
		t.Errorf("missing registry = (%q, %q), want (%q, \"\")", name, version, productName)
	}
}

// The whole auth design leans on "_-prefixed files are invisible to the
// engine", so pin that contract: a deliberately invalid login page must not
// make `pustaka check` fail.
func TestCheckIgnoresLoginPage(t *testing.T) {
	root := writeCheckFixture(t, "")
	bogus := `<html><body>no doctype, no data-page, no main, not registered</body></html>`
	if err := os.WriteFile(filepath.Join(root, "_login.html"), []byte(bogus), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := check(root); got != 0 {
		t.Fatalf("check() = %d, want 0 — _login.html must stay invisible to the checker", got)
	}
}

func TestAuthGzipOrdering(t *testing.T) {
	root := writeServeFixture(t)
	a := testAuth(t, root, nil)
	h := testHandler(t, root, a)

	r := httptest.NewRequest("GET", "/guide/x.html", nil)
	r.Header.Set("Accept-Encoding", "gzip")
	r.Header.Set("Sec-Fetch-Dest", "document")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 302 {
		t.Fatalf("status %d, want 302", w.Code)
	}
	if enc := w.Header().Get("Content-Encoding"); enc != "" {
		t.Errorf("a denial must not be gzip-framed, got %q", enc)
	}

	r = httptest.NewRequest("GET", "/guide/x.html", nil)
	r.Header.Set("Accept-Encoding", "gzip")
	r.Header.Set("Sec-Fetch-Dest", "document")
	r.AddCookie(&http.Cookie{Name: a.cfg.CookieName, Value: a.sign(time.Now())})
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("status %d, want 200", w.Code)
	}
	if enc := w.Header().Get("Content-Encoding"); enc != "gzip" {
		t.Errorf("an allowed page should still be compressed, got %q", enc)
	}
}
