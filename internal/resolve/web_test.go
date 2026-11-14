package resolve

import (
  "testing"

  "pidx/internal/model"
)

func TestWebChain(t *testing.T) {
  html, js, css := "web/register.html", "web/register.js", "web/style.css"
  idForm := "form:web/register.html#form#registrationForm"
  idBtn := "element:web/register.html#button#submitBtn"
  idFn := "function:web/register.js#submitForm"
  idRule := "css_rule:web/style.css#.primary-button"
  in := Input{
    Files: []model.FileRow{
      {Path: html, Language: "html"},
      {Path: js, Language: "javascript"},
      {Path: css, Language: "css"},
    },
    Symbols: []model.Symbol{
      {ID: idForm, Kind: "form", Name: "form#registrationForm", File: html, StartLine: 10},
      {ID: idBtn, Kind: "element", Name: "button#submitBtn", File: html, StartLine: 12},
      {ID: idFn, Kind: "function", Name: "submitForm", File: js, StartLine: 5},
      {ID: idRule, Kind: "css_rule", Name: ".primary-button", File: css, StartLine: 3},
    },
    Refs: []model.RawRef{
      {File: html, Kind: "script_src", Name: "register.js", Line: 14},
      {File: html, Kind: "style_href", Name: "style.css", Line: 4},
      {File: html, Kind: "script_src", Name: "https://cdn.example.com/lib.js", Line: 15},
      {File: html, Kind: "html_class", Name: "primary-button", Target: "button", From: "button#submitBtn", Line: 12},
      {File: html, Kind: "html_handler", Name: "onclick", Target: "submitForm()", From: "button#submitBtn", Line: 12},
      {File: js, Kind: "dom_id", Name: "submitBtn", From: "submitForm", Line: 2},
      {File: js, Kind: "dom_id", Name: "missing", From: "submitForm", Line: 9},
      {File: js, Kind: "dom_listen", Name: "#submitBtn", Target: "click:submitForm", Line: 3},
      {File: css, Kind: "css_id", Name: "submitBtn", Line: 3},
      {File: css, Kind: "css_class", Name: "primary-button", Line: 3},
    },
  }
  es := ResolveWeb(in)
  check := func(typ, src, name, status, dst string) {
    t.Helper()
    e := find(es, typ, src, name)
    if e == nil || e.Status != status || e.Dst != dst {
      t.Fatalf("%s %s -> %s: want %s %q, got %+v", typ, src, name, status, dst, e)
    }
  }
  check("loads_script", "file:"+html, "register.js", "resolved", "file:"+js)
  check("loads_style", "file:"+html, "style.css", "resolved", "file:"+css)
  check("loads_script", "file:"+html, "https://cdn.example.com/lib.js", "external", "")
  check("references_dom", idFn, "#submitBtn", "resolved", idBtn)
  check("references_dom", idFn, "#missing", "unresolved", "")
  check("listens_dom", idBtn, "submitForm", "resolved", idFn)
  check("styles", idRule, "#submitBtn", "resolved", idBtn)
  check("styles", idRule, ".primary-button", "inferred", idBtn)
}

func TestWebNoHTMLNoEdges(t *testing.T) {
  in := Input{Files: []model.FileRow{{Path: "a.go", Language: "go"}}}
  if es := ResolveWeb(in); len(es) != 0 {
    t.Fatalf("expected no edges, got %d", len(es))
  }
}
