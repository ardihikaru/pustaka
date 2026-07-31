/* Pustaka ToC part — group "Guide". Menu nodes use the exact key order
   id, title, children; page leaves keep id, file, title, desc, tags, sections. */
window.DOCS.nav.push({
  group: "Guide",
  pages: [
    {
      id: "getting-started", file: "guide/getting-started.html", title: "Getting started",
      desc: "Folder layout, adding your first page, registering it in the ToC, running the Go server.",
      tags: ["install", "setup", "tutorial", "go", "quickstart"],
      sections: [
        { anchor: "layout", title: "Project layout", text: "A Pustaka site is HTML files plus a registry, with no build step." },
        { anchor: "first-page", title: "Add your first page", text: "Create a page from the template and author plain HTML." },
        { anchor: "register", title: "Register it in the ToC", text: "Register the page so navigation and search discover it." },
        { anchor: "run", title: "Run it", text: "Open statically or serve through the Go engine." }
      ]
    },
    {
      id: "guide-authoring-menu", title: "Authoring", children: [
        {
          id: "authoring", file: "guide/authoring.html", title: "Writing pages",
          desc: "The component library: metadata, admonitions, tabs, code blocks, tables, and filtering.",
          tags: ["components", "metadata", "admonition", "tabs", "code", "filter", "reference"],
          sections: [
            { anchor: "structure", title: "Page structure", text: "Authored content lives in main.doc; the runtime injects the shell." },
            { anchor: "admonitions", title: "Admonitions", text: "Note, tip and warning callouts use simple classes." },
            { anchor: "tabs-code", title: "Tabs & code blocks", text: "Tabs and copyable code blocks are runtime wired." },
            { anchor: "page-filter", title: "Filtering page content", text: "data-filter makes page content filterable." }
          ]
        },
        {
          id: "markdown-to-html", file: "guide/markdown-to-html.html", title: "Markdown to HTML",
          desc: "Convert Markdown to a clean Pustaka page, then validate and register it.",
          tags: ["markdown", "html", "pandoc", "conversion", "migration", "tutorial"],
          sections: [
            { anchor: "convert", title: "Convert the source", text: "Use Pandoc to produce an HTML fragment from GitHub-Flavored Markdown." },
            { anchor: "fit-the-template", title: "Fit the Pustaka template", text: "Place the fragment inside main.doc and restore the page contract." },
            { anchor: "repair-semantics", title: "Repair semantics", text: "Add heading ids, translate extensions, and sanitize untrusted output." },
            { anchor: "register-and-check", title: "Register and check", text: "Register the page and use pustaka check as the acceptance gate." },
            { anchor: "ai-recipe", title: "AI conversion recipe", text: "A copyable prompt gives an AI deterministic conversion inputs and outputs." }
          ]
        },
        {
          id: "page-metadata", file: "guide/page-metadata.html", title: "Page metadata",
          desc: "Add optional tags, hashtags, publication dates, update dates, and version applicability to a page header.",
          tags: ["metadata", "tags", "hashtags", "published", "updated", "semver", "version"],
          sections: [
            { anchor: "contract", title: "The metadata contract", text: "One declarative element renders the complete header metadata panel." },
            { anchor: "version-ranges", title: "Version ranges", text: "A required lower SemVer bound and optional upper bound describe applicability." },
            { anchor: "validation", title: "Validation", text: "The checker rejects malformed dates, tokens, duplicates, and reversed ranges." },
            { anchor: "ai-prompt", title: "AI prompt", text: "A copyable prompt produces metadata without inventing values." }
          ]
        },
        {
          id: "ai-tutorial", file: "guide/ai-tutorial.html", title: "AI authoring tutorial",
          desc: "Build the prompt, attach the right context, and run the check-loop that validates AI-generated pages.",
          tags: ["ai", "prompt", "generation", "llm", "automation", "tutorial"],
          sections: [
            { anchor: "four-inputs", title: "The four inputs of a good prompt", text: "Role, standard, registry state and task brief." },
            { anchor: "prompt-template", title: "A prompt template you can copy", text: "A complete page-authoring task template." },
            { anchor: "the-loop", title: "The correction loop", text: "Generate, check, correct and repeat." },
            { anchor: "failure-modes", title: "Failure modes and their fixes", text: "Common structural and lifecycle errors." },
            { anchor: "ci-gate", title: "Make the gate automatic", text: "Run the validator in continuous integration." }
          ]
        }
      ]
    },
    {
      id: "guide-components-menu", title: "Components", children: [
        {
          id: "charts", file: "guide/charts.html", title: "Interactive charts",
          desc: "Apache ECharts embedded in doc pages: zoomable, theme-aware, lazy-initialized.",
          tags: ["echarts", "visualization", "charts", "data", "interactive"],
          sections: [
            { anchor: "why-echarts", title: "Why ECharts", text: "Interactive visualization in an HTML-first page." },
            { anchor: "demo-line", title: "Demo: request latency", text: "Theme-aware line chart with zoom and tooltip." },
            { anchor: "demo-bar", title: "Demo: engine comparison", text: "A comparative performance bar chart." }
          ]
        },
        {
          id: "guide-navigation-menu", title: "Navigation", children: [
            {
              id: "nested-sidebar", file: "guide/nested-sidebar.html", title: "Nested sidebar",
              desc: "Implement recursive sidebar sub-menus with active paths, persistence, keyboard controls, and ARIA.",
              tags: ["sidebar", "navigation", "submenu", "recursive", "accessibility", "toc"],
              sections: [
                { anchor: "schema", title: "Recursive schema", text: "Menu nodes contain children while page leaves keep the established contract." },
                { anchor: "runtime-behavior", title: "Runtime behavior", text: "Active ancestors open automatically and other branches persist." },
                { anchor: "migration", title: "Migrate a flat group", text: "Wrap existing leaves without changing their page records." },
                { anchor: "ai-recipe", title: "AI authoring recipe", text: "Rules for safely editing a recursive ToC." }
              ]
            }
          ]
        },
        {
          id: "guide-diagrams-menu", title: "Diagrams", children: [
            {
              id: "guide-sisflow-menu", title: "Sisflow", children: [
                {
                  id: "sisflow-erd", file: "guide/sisflow/erd.html", title: "Entity-relationship diagrams",
                  desc: "Editable and read-only Sisflow ERD variants with native fullscreen, fixed data, and runnable code.",
                  tags: ["sisflow", "erd", "diagram", "entity", "svg", "fullscreen", "readonly"],
                  sections: [
                    { anchor: "live-workbench", title: "Live ERD workbench", text: "Editable boxed ERD with entity visibility and freeze controls." },
                    { anchor: "live-viewer", title: "Live ERD viewer", text: "Unboxed ERD created read-only without an entity sidebar." },
                    { anchor: "runnable-sample", title: "Complete runnable sample", text: "Copyable HTML, model, setup, export and lifecycle code." },
                    { anchor: "fullscreen", title: "Canvas and viewer fullscreen", text: "Native fullscreen recipes for the canvas and complete viewer." },
                    { anchor: "provenance", title: "Source and provenance", text: "Pinned Sisflow commit and ZRender attribution." }
                  ]
                },
                {
                  id: "sisflow-system-architecture", file: "guide/sisflow/system-architecture.html", title: "System architecture",
                  desc: "Sisflow architecture workbench and read-only viewer with boundaries, tagged styles, and scoped layout.",
                  tags: ["sisflow", "architecture", "systems", "boundary", "fullscreen", "readonly"],
                  sections: [
                    { anchor: "live-workbench", title: "Live architecture workbench", text: "Editable boxed systems map." },
                    { anchor: "live-viewer", title: "Live architecture viewer", text: "Unboxed fixed and read-only systems map." },
                    { anchor: "runnable-sample", title: "Complete runnable sample", text: "Boundary, styles, layout, fullscreen, export and disposal." },
                    { anchor: "fullscreen", title: "Canvas and viewer fullscreen", text: "Choose diagram-only or complete-viewer fullscreen." }
                  ]
                },
                {
                  id: "sisflow-block-diagram", file: "guide/sisflow/block-diagram.html", title: "Block diagrams",
                  desc: "Editable and read-only Sisflow block diagrams for control systems and signal paths.",
                  tags: ["sisflow", "block", "control", "signal", "fullscreen", "readonly"],
                  sections: [
                    { anchor: "live-workbench", title: "Live block-diagram workbench", text: "Editable boxed block diagram." },
                    { anchor: "live-viewer", title: "Live block-diagram viewer", text: "Unboxed fixed and read-only block diagram." },
                    { anchor: "runnable-sample", title: "Complete runnable sample", text: "Shape catalog, model, fullscreen, export and disposal." },
                    { anchor: "fullscreen", title: "Canvas and viewer fullscreen", text: "Choose diagram-only or complete-viewer fullscreen." }
                  ]
                },
                {
                  id: "sisflow-class-diagram", file: "guide/sisflow/class-diagram.html", title: "Class diagrams",
                  desc: "Editable and read-only Sisflow UML class diagrams with compartments and relationship markers.",
                  tags: ["sisflow", "class", "uml", "inheritance", "fullscreen", "readonly"],
                  sections: [
                    { anchor: "live-workbench", title: "Live class-diagram workbench", text: "Editable boxed UML class diagram." },
                    { anchor: "live-viewer", title: "Live class-diagram viewer", text: "Unboxed fixed and read-only UML viewer." },
                    { anchor: "runnable-sample", title: "Complete runnable sample", text: "Class model, markers, fullscreen, export and disposal." },
                    { anchor: "fullscreen", title: "Canvas and viewer fullscreen", text: "Choose diagram-only or complete-viewer fullscreen." }
                  ]
                },
                {
                  id: "sisflow-flowchart", file: "guide/sisflow/flowchart.html", title: "Flowcharts",
                  desc: "Editable and read-only Sisflow flowcharts using the extended ISO 5807 shape catalog.",
                  tags: ["sisflow", "flowchart", "iso-5807", "process", "fullscreen", "readonly"],
                  sections: [
                    { anchor: "live-workbench", title: "Live flowchart workbench", text: "Editable boxed process flow." },
                    { anchor: "live-viewer", title: "Live flowchart viewer", text: "Unboxed fixed and read-only process flow." },
                    { anchor: "runnable-sample", title: "Complete runnable sample", text: "Extended shapes, model, fullscreen, export and disposal." },
                    { anchor: "fullscreen", title: "Canvas and viewer fullscreen", text: "Choose diagram-only or complete-viewer fullscreen." }
                  ]
                }
              ]
            }
          ]
        }
      ]
    },
    {
      id: "guide-deployment-menu", title: "Deployment", children: [
        {
          id: "production", file: "guide/deploy/production.html", title: "Production deployment",
          desc: "Ship docs as a static folder or as the pustaka binary behind a proxy, with a systemd unit.",
          tags: ["deploy", "production", "systemd", "nginx", "hosting"],
          sections: [
            { anchor: "two-modes", title: "Two ways to ship", text: "Static hosting or the Go binary behind a proxy." },
            { anchor: "systemd", title: "Run it as a service", text: "A systemd unit with restart-on-failure." }
          ]
        }
      ]
    },
    /* pustaka:insert — `pustaka new` adds flat entries above this marker */
  ]
});
