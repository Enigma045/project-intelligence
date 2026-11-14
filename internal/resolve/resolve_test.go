package resolve

import (
  "testing"

  "pidx/internal/model"
)

func find(es []model.Edge, typ, src, name string) *model.Edge {
  for i := range es {
    if es[i].Type == typ && es[i].Src == src && es[i].DstName == name {
      return &es[i]
    }
  }
  return nil
}

func TestJSChain(t *testing.T) {
  idSubmit := "function:web/dashboard.js#submitForm"
  idPost := "function:web/api.js#postRegistration"
  in := Input{
    Files: []model.FileRow{
      {Path: "web/dashboard.js", Language: "javascript", Imports: []string{"./api"}},
      {Path: "web/api.js", Language: "javascript"},
    },
    Symbols: []model.Symbol{
      {ID: idSubmit, Kind: "function", Name: "submitForm", File: "web/dashboard.js", StartLine: 3},
      {ID: idPost, Kind: "function", Name: "postRegistration", File: "web/api.js", StartLine: 1},
    },
    Refs: []model.RawRef{
      {File: "web/dashboard.js", Kind: "call", Name: "api.postRegistration", Line: 4, From: "submitForm"},
      {File: "web/dashboard.js", Kind: "call", Name: "console.log", Line: 5, From: "submitForm"},
      {File: "web/dashboard.js", Kind: "call", Name: "missing", Line: 6, From: "submitForm"},
      {File: "web/dashboard.js", Kind: "event", Name: "click", Target: "submitForm", Line: 2},
    },
  }
  es := Resolve(in)
  if e := find(es, "calls", idSubmit, "api.postRegistration"); e == nil || e.Dst != idPost || e.Status != "resolved" {
    t.Fatalf("qualified call not resolved: %+v", e)
  }
  if e := find(es, "listens", "file:web/dashboard.js", "submitForm"); e == nil || e.Dst != idSubmit || e.Status != "resolved" {
    t.Fatalf("listener not resolved: %+v", e)
  }
  if e := find(es, "calls", idSubmit, "console.log"); e == nil || e.Status != "external" {
    t.Fatalf("console.log should be external: %+v", e)
  }
  if e := find(es, "calls", idSubmit, "missing"); e == nil || e.Status != "unresolved" {
    t.Fatalf("missing should be unresolved: %+v", e)
  }
  if e := find(es, "imports", "file:web/dashboard.js", "./api"); e == nil || e.Dst != "file:web/api.js" {
    t.Fatalf("import edge missing: %+v", e)
  }
}

func TestGoPackages(t *testing.T) {
  idMain := "function:cmd/main.go#main"
  idDo := "function:internal/svc/svc.go#Do"
  idHelper := "function:internal/svc/util.go#helper"
  in := Input{
    Files: []model.FileRow{
      {Path: "cmd/main.go", Language: "go", Imports: []string{"fmt", "mod/internal/svc"}},
      {Path: "internal/svc/svc.go", Language: "go"},
      {Path: "internal/svc/util.go", Language: "go"},
    },
    Symbols: []model.Symbol{
      {ID: idMain, Kind: "function", Name: "main", File: "cmd/main.go", StartLine: 1},
      {ID: idDo, Kind: "function", Name: "Do", File: "internal/svc/svc.go", StartLine: 1},
      {ID: idHelper, Kind: "function", Name: "helper", File: "internal/svc/util.go", StartLine: 1},
    },
    Refs: []model.RawRef{
      {File: "cmd/main.go", Kind: "call", Name: "svc.Do", Line: 5, From: "main"},
      {File: "cmd/main.go", Kind: "call", Name: "fmt.Println", Line: 6, From: "main"},
      {File: "internal/svc/svc.go", Kind: "call", Name: "helper", Line: 3, From: "Do"},
    },
  }
  es := Resolve(in)
  if e := find(es, "calls", idMain, "svc.Do"); e == nil || e.Dst != idDo || e.Status != "resolved" {
    t.Fatalf("svc.Do not resolved: %+v", e)
  }
  if e := find(es, "calls", idMain, "fmt.Println"); e == nil || e.Status != "external" {
    t.Fatalf("fmt.Println should be external: %+v", e)
  }
  if e := find(es, "calls", idDo, "helper"); e == nil || e.Dst != idHelper || e.Status != "resolved" {
    t.Fatalf("same-package call not resolved: %+v", e)
  }
  if e := find(es, "imports", "file:cmd/main.go", "mod/internal/svc"); e == nil {
    t.Fatal("go import edge missing")
  }
}
