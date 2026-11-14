package resolve

import (
  "testing"

  "pidx/internal/model"
)

func goPkgInput(callName string, extra ...model.RawRef) Input {
  in := Input{
    Files: []model.FileRow{
      {Path: "Routes/f.go", Language: "go", Imports: []string{"net/http", "mod/Audit_logs"}},
      {Path: "Audit_logs/u.go", Language: "go"},
    },
    Symbols: []model.Symbol{
      {ID: "function:Routes/f.go#Fn", Kind: "function", Name: "Fn", File: "Routes/f.go", StartLine: 1},
      {ID: "function:Audit_logs/u.go#Create_user_log", Kind: "function", Name: "Create_user_log", File: "Audit_logs/u.go", StartLine: 1},
    },
    Refs: []model.RawRef{
      {File: "Audit_logs/u.go", Kind: "go_package", Name: "user_logs", Line: 1},
      {File: "Routes/f.go", Kind: "call", Name: callName, Line: 5, From: "Fn"},
    },
  }
  in.Refs = append(in.Refs, extra...)
  return in
}

func TestGoPackageNameDiffersFromFolder(t *testing.T) {
  e := find(Resolve(goPkgInput("user_logs.Create_user_log")), "calls", "function:Routes/f.go#Fn", "user_logs.Create_user_log")
  if e == nil || e.Dst != "function:Audit_logs/u.go#Create_user_log" || e.Status != "resolved" {
    t.Fatalf("package-name qualifier must resolve: %+v", e)
  }
}

func TestGoImportAlias(t *testing.T) {
  in := goPkgInput("ul.Create_user_log", model.RawRef{File: "Routes/f.go", Kind: "go_import", Name: "ul", Target: "mod/Audit_logs", Line: 3})
  e := find(Resolve(in), "calls", "function:Routes/f.go#Fn", "ul.Create_user_log")
  if e == nil || e.Dst != "function:Audit_logs/u.go#Create_user_log" || e.Status != "resolved" {
    t.Fatalf("import alias must resolve: %+v", e)
  }
}
