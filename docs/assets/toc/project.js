/* Pustaka ToC part — group "Project".
   Loaded by assets/toc.js (parts list). Keep this file under 200 lines.
   Entries MUST keep the key order: id, file, title, desc, tags, sections. */
window.DOCS.nav.push({
  group: "Project",
  pages: [
    {
      id: "changelog",
      file: "changelog.html",
      title: "Changelog",
      desc: "Release history for the documented product, kept in semver.",
      tags: ["changelog", "releases", "semver", "versions"],
      sections: [
        { anchor: "versioning-policy", title: "Versioning policy",
          text: "Semantic versioning: major for breaking changes, minor for new capability, patch for fixes. Current version lives in assets/toc.js product.semver." },
        { anchor: "releases", title: "Releases",
          text: "v0.6.0 optional page metadata, recursive sidebar menus, Markdown and navigation tutorials, five offline Sisflow diagram workbench/viewer guides, and partial-swap resource fix. v0.5.0 landing layout, screenshots and vendored assets. v0.4.0 interactive landing and AI authoring. v0.3.0 nested folders and multi-file ToC. v0.2.0 Go engine and AI kit. v0.1.0 static concept." }
      ]
    },
    /* pustaka:insert — `pustaka new` adds entries above this marker */
  ]
});
