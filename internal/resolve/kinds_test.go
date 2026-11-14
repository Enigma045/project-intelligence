package resolve

import (
  "testing"

  "pidx/internal/model"
)

func TestGoValueCallNeverMatchesPlainFunction(t *testing.T) {
  idF := "function:a/x.go#f"
  in := Input{
    Files: []model.FileRow{
      {Path: "a/x.go", Language: "go"},
      {Path: "b/y.go", Language: "go"},
    },
    Symbols: []model.Symbol{
      {ID: idF, Kind: "function", Name: "f", File: "a/x.go", StartLine: 1},
      {ID: "function:b/y.go#Run", Kind: "function", Name: "Run", File: "b/y.go", StartLine: 1},
    },
    Refs: []model.RawRef{
      {File: "a/x.go", Kind: "call", Name: "?.Run", Line: 3, From: "f"},
    },
  }
  e := find(Resolve(in), "calls", idF, "?.Run")
  if e == nil || e.Dst != "" || e.Status != "dynamic" {
    t.Fatalf("value call must not match a plain function: %+v", e)
  }
}
