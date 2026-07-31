/* Pustaka ToC part — group "Concept".
   Loaded by assets/toc.js (parts list). Keep this file under 200 lines;
   when it grows past that, split the group or move pages to a new part.
   Entries MUST keep the key order: id, file, title, desc, tags, sections. */
window.DOCS.nav.push({
  group: "Concept",
  pages: [

        {
          id: "architecture",
          file: "concept/architecture.html",
          title: "Go + HTMX architecture",
          desc: "How the production engine serves pages, builds the search index, and swaps content with HTMX.",
          tags: ["go", "htmx", "server", "performance", "architecture", "search index"],
          sections: [
            { anchor: "request-flow", title: "Request flow",
              text: "Browser asks, Go answers. First visit returns the full shell; every next click is an hx-get that swaps only the article region, no full page reload, scroll and theme preserved." },
            { anchor: "search-index", title: "Search index at startup",
              text: "On boot the Go server walks the docs folder, parses each HTML file, extracts headings and text, and builds an in-memory inverted index. Queries answer in microseconds." },
            { anchor: "performance", title: "Performance budget",
              text: "Single static binary, zero runtime dependencies, pages precompressed, sub-millisecond routing. Targets: first paint under 1s on 3G, search results under 16ms." }
          ]
        },
        {
          id: "faq",
          file: "concept/faq.html",
          title: "FAQ",
          desc: "Short answers on migration, search, AI authoring, deployment.",
          tags: ["faq", "questions", "migration", "deploy"],
          sections: [
            { anchor: "questions", title: "Common questions",
              text: "Markdown migration, offline search, AI authoring loop, Go HTMX rationale, charts offline, deployment." }
          ]
        },
        {
      id: "performance",
      file: "concept/performance.html",
      title: "Performance",
      desc: "Measured payloads, partial-swap latency and index build cost, with commands to reproduce.",
      tags: ["performance", "benchmark", "latency", "payload", "gzip", "speed"],
      sections: [
        { anchor: "headline", title: "Headline figures",
          text: "Median partial response under a millisecond on loopback, 2.9 kB gzipped article per navigation, 31 ms full index rebuild, 8.8 MB dependency-free binary." },
        { anchor: "payloads", title: "Payload sizes",
          text: "Gzip versus plain for landing HTML, full pages, partial fragments, site.css, site.js and index.json. About 16 kB for the first page of a session, then 2.9 kB per navigation." },
        { anchor: "latency", title: "Latency",
          text: "Twenty sequential partial requests: min 0.5 ms, median 0.8 ms, p95 20.6 ms caused by the dev-mode debounced rebuild, which prod mode removes." },
        { anchor: "index-build", title: "Index build cost",
          text: "The v0.5 ten-page baseline parsed 44 records in 31 ms; cost is linear in page bytes and happens at startup, not per request." },
        { anchor: "client-side", title: "What the browser does",
          text: "No build output or bundle, lazy IntersectionObserver chart init, swaps replacing only main, and static search over ToC metadata with no extra request." },
        { anchor: "reproduce", title: "Reproduce it yourself",
          text: "Curl commands for payload sizes, dev versus prod latency comparison, and timing the index build." },
        { anchor: "limits", title: "Where it would slow down",
          text: "Whole-index rebuilds past a few thousand pages, shipping index.json to clients on large sites, ECharts CDN weight on cold visits, and dev mode under real load." }
      ]
    },
    /* pustaka:insert — `pustaka new` adds entries above this marker */
  ]
});
