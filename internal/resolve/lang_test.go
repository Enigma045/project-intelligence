package resolve

import (
  "testing"

  "pidx/internal/model"
)

func TestNoCrossLanguageEdges(t *testing.T) {
  idRun := "function:web/a.js#run"
  in := Input{
    Files: []model.FileRow{
      {Path: "web/a.js", Language: "javascript"},
      {Path: "x/w.go", Language: "go"},
    },
    Symbols: []model.Symbol{
      {ID: idRun, Kind: "function", Name: "run", File: "web/a.js", StartLine: 1},
      {ID: "method:x/w.go#W.add", Kind: "method", Name: "add", Parent: "W", File: "x/w.go", StartLine: 1},
    },
    Refs: []model.RawRef{
      {File: "web/a.js", Kind: "call", Name: "el.classList.add", Line: 2, From: "run"},
    },
  }
  es := Resolve(in)
  e := find(es, "calls", idRun, "el.classList.add")
  if e == nil || e.Dst != "" || e.Status != "dynamic" {
    t.Fatalf("JS call must not resolve to a Go method: %+v", e)
  }
}
