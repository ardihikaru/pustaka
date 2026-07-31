/* Pustaka ToC part — sample pages that sit beside the core guide groups. */
window.DOCS.nav.push({
  group: "Samples",
  pages: [
    {
      id: "guide-samples-menu", title: "Samples", children: [
        {
          id: "sisflow-request-workflow", file: "guide/sisflow/request-workflow.html", title: "Request workflow",
          desc: "Sisflow-derived request workflow with editable labels, stage progression, and read-only viewer mode.",
          tags: ["sisflow", "workflow", "request", "tree", "readonly"],
          sections: [
            { anchor: "live-workbench", title: "Live request-workflow workbench", text: "Editable workflow route with active-stage focus." },
            { anchor: "live-viewer", title: "Live request-workflow viewer", text: "Read-only request progression with no editing." },
            { anchor: "runnable-sample", title: "Complete runnable sample", text: "DOM bootstrap with a canonical request model." },
            { anchor: "fullscreen", title: "Fullscreen and embedding", text: "Fullscreen the iframe or its wrapper." }
          ]
        },
        {
          id: "sisflow-org-chart-editor", file: "guide/sisflow/org-chart-editor.html", title: "Org chart editor",
          desc: "Sisflow-derived tree editor for organization charts with inline labels and branch controls.",
          tags: ["sisflow", "org-chart", "tree", "editor", "readonly"],
          sections: [
            { anchor: "live-workbench", title: "Live org-chart-editor workbench", text: "Editable hierarchy with add-child and collapse controls." },
            { anchor: "live-viewer", title: "Live org-chart-editor viewer", text: "Read-only org chart for inspection." },
            { anchor: "tree-editor-workflow", title: "Tree-editor workflow", text: "The chart is edited branch-by-branch, not freely laid out." },
            { anchor: "runnable-sample", title: "Complete runnable sample", text: "Recursive tree renderer with a canonical source model." }
          ]
        },
        {
          id: "kanban-board", file: "guide/kanban-board.html", title: "Kanban board",
          desc: "DOM-first Kanban board sample with drag-and-drop cards, provider state, and JSON export.",
          tags: ["kanban", "board", "dom", "drag-and-drop", "export"],
          sections: [
            { anchor: "live-board", title: "Live kanban board", text: "DOM lane board with draggable cards." },
            { anchor: "sample-behavior", title: "Sample behavior", text: "Card movement updates the provider snapshot." },
            { anchor: "provider-export", title: "Provider and export notes", text: "Plain JSON provider state and export contract." },
            { anchor: "embedding", title: "Embedding", text: "Use any HTML shell with an explicit height." }
          ]
        }
      ]
    }
  ]
});

/* pustaka:insert — sample pages are maintained here directly */
