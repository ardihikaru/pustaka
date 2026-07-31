package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
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
