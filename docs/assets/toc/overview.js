/* Pustaka ToC part — group "Overview".
   Loaded by assets/toc.js (parts list). Keep this file under 200 lines;
   when it grows past that, split the group or move pages to a new part.
   Entries MUST keep the key order: id, file, title, desc, tags, sections. */
window.DOCS.nav.push({
  group: "Overview",
  pages: [

        {
      id: "home",
      file: "index.html",
      title: "Home",
      desc: "Interactive landing page: responsive device screenshots, live search, the AI authoring loop and measured performance.",
      tags: ["home", "overview", "landing", "responsive", "search", "ai"],
      sections: [
        { anchor: "responsive", title: "One page, every screen",
          text: "Unretouched screenshots of the same page at 390 px portrait and 1280 px tablet landscape: navigation collapses into a sheet, charts re-flow, and from 1200 px sidebar, article and on-this-page rail sit side by side." },
        { anchor: "search-demo", title: "Answers before you finish typing",
          text: "Embedded live search running the real engine with AND semantics and prefix matching over titles, tags and section text." },
        { anchor: "ai-loop", title: "Docs an AI writes and a validator refuses to accept",
          text: "The authoring loop replayed: model emits pages, pustaka check rejects spec violations, the model fixes them until the gate passes." },
        { anchor: "numbers", title: "Measured, not claimed",
          text: "Median partial-swap response 0.8 ms, 2.9 kB gzipped per navigation, 31 ms index rebuild, zero dependencies." },
        { anchor: "why", title: "Why HTML-first",
          text: "Interactivity is ordinary markup, one registry file, machine-checkable structure, and no third-party requests because fonts and ECharts are vendored." },
        { anchor: "explore", title: "Explore the docs",
          text: "Filterable index of every page: Markdown conversion, page metadata, nested navigation, Sisflow ERD, AI authoring, charts, deployment, architecture, performance, FAQ, and changelog." }
      ]
    },
    /* pustaka:insert — `pustaka new` adds entries above this marker */
  ]
});
