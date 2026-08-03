/* ============================================================
   PUSTAKA registry — the mkdocs.yml equivalent.
   This parent file holds site + product metadata and the list of
   ToC PARTS. Navigation lives in the part files (assets/toc/*.js)
   so each file stays small (< 200 lines) and diffs stay readable.
   The runtime loads parts in order; the Go engine parses the same
   files. Part paths are relative to the docs root.
   ============================================================ */
window.DOCS = {
  site: {
    name: "Pustaka",
    tagline: "Open-source documentation for humans and AI agents",
    version: "v0.7.1"
  },
  /* One repo documents ONE product. Fork this repo per product and
     update this block — the changelog page renders against it. */
  product: {
    name: "Pustaka",
    semver: "0.7.1",
    repo: "https://example.com/your-org/pustaka"
  },
  parts: [
    "assets/toc/overview.js",
    "assets/toc/guide.js",
    "assets/toc/guide-samples.js",
    "assets/toc/concept.js",
    "assets/toc/project.js"
  ],
  nav: []  /* filled by the parts above */
};
