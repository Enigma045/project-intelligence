package resolve

import (
  "testing"

  "pidx/internal/model"
)

func TestLocalTargets(t *testing.T) {
  idMain := "function:cmd/main.go#main"
  idInsert := "method:store/store.go#Store.Insert"
  in := Input{
    Files: []model.FileRow{
      {Path: "cmd/main.go", Language: "go", Imports: []string{"testing", "mod/store"}},
      {Path: "store/store.go", Language: "go"},
    },
    Symbols: []model.Symbol{
      {ID: idMain, Kind: "function", Name: "main", File: "cmd/main.go", StartLine: 1},
      {ID: idInsert, Kind: "method", Name: "Insert", Parent: "Store", File: "store/store.go", StartLine: 1},
      {ID: "var:cmd/main.go#reX", Kind: "var", Name: "reX", File: "cmd/main.go", StartLine: 2},
    },
    Refs: []model.RawRef{
      {File: "cmd/main.go", Kind: "call", Name: "st.Insert", Target: "local:*store.Store", Line: 5, From: "main"},
      {File: "cmd/main.go", Kind: "call", Name: "t.Fatal", Target: "local:*testing.T", Line: 6, From: "main"},
      {File: "cmd/main.go", Kind: "call", Name: "need", Target: "local", Line: 7, From: "main"},
      {File: "cmd/main.go", Kind: "call", Name: "fs.Parse", Target: "local", Line: 8, From: "main"},
      {File: "cmd/main.go", Kind: "call", Name: "reX.Find", Line: 9, From: "main"},
      {File: "cmd/main.go", Kind: "call", Name: "x.Unknown", Line: 10, From: "main"},
      {File: "cmd/main.go", Kind: "call", Name: "err.Error", Target: "local:error", Line: 11, From: "main"},
      {File: "cmd/main.go", Kind: "call", Name: "s.Missing", Target: "local:*store.Store", Line: 12, From: "main"},
    },
  }
  es := Resolve(in)
  check := func(name, status, dst string) {
    t.Helper()
    e := find(es, "calls", idMain, name)
    if e == nil || e.Status != status || e.Dst != dst {
      t.Fatalf("%s: want %s -> %q, got %+v", name, status, dst, e)
    }
  }
  check("st.Insert", "resolved", idInsert)
  check("t.Fatal", "external", "")
  check("need", "value", "")
  check("fs.Parse", "value", "")
  check("reX.Find", "value", "")
  check("x.Unknown", "dynamic", "")
  check("err.Error", "external", "")
  check("s.Missing", "dynamic", "")
}
