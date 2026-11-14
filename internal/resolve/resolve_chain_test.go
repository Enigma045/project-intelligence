package resolve

import (
  "testing"

  "pidx/internal/model"
)

func TestChainedCallNeverResolvesToProjectMethod(t *testing.T) {
  idF := "function:Routes/l.go#Login"
  in := Input{
    Files: []model.FileRow{
      {Path: "Routes/l.go", Language: "go", Imports: []string{"mod/utils"}},
      {Path: "utils/t.go", Language: "go"},
    },
    Symbols: []model.Symbol{
      {ID: idF, Kind: "function", Name: "Login", File: "Routes/l.go", StartLine: 1},
      {ID: "method:utils/t.go#AutoTime.Scan", Kind: "method", Name: "Scan", Parent: "AutoTime", File: "utils/t.go", StartLine: 1},
    },
    Refs: []model.RawRef{
      {File: "Routes/l.go", Kind: "call", Name: "?.Scan", Line: 3, From: "Login"},
    },
  }
  e := find(Resolve(in), "calls", idF, "?.Scan")
  if e == nil || e.Dst != "" || e.Status != "dynamic" {
    t.Fatalf("call on an expression result must not resolve to a project method: %+v", e)
  }
}

func TestJSChainedCallIsValue(t *testing.T) {
  idF := "function:web/a.js#run"
  in := Input{
    Files: []model.FileRow{{Path: "web/a.js", Language: "javascript"}},
    Symbols: []model.Symbol{
      {ID: idF, Kind: "function", Name: "run", File: "web/a.js", StartLine: 1},
      {ID: "function:web/a.js#map", Kind: "function", Name: "map", File: "web/a.js", StartLine: 9},
    },
    Refs: []model.RawRef{
      {File: "web/a.js", Kind: "call", Name: "?.map", Line: 3, From: "run"},
    },
  }
  e := find(Resolve(in), "calls", idF, "?.map")
  if e == nil || e.Dst != "" || e.Status != "value" {
    t.Fatalf("JS method call after a dot must be a value edge: %+v", e)
  }
}
