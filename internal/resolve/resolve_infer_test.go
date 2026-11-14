package resolve

import (
  "testing"

  "pidx/internal/model"
)

func TestReturnTypeInference(t *testing.T) {
  idMain := "function:cmd/main.go#main"
  idGet := "method:store/store.go#Store.Get"
  in := Input{
    Files: []model.FileRow{
      {Path: "cmd/main.go", Language: "go", Imports: []string{"mod/store", "net/http"}},
      {Path: "store/store.go", Language: "go", Imports: []string{"database/sql"}},
    },
    Symbols: []model.Symbol{
      {ID: idMain, Kind: "function", Name: "main", File: "cmd/main.go", StartLine: 1},
      {ID: "function:store/store.go#Open", Kind: "function", Name: "Open", File: "store/store.go", StartLine: 1, Returns: "(*Store, error)"},
      {ID: idGet, Kind: "method", Name: "Get", Parent: "Store", File: "store/store.go", StartLine: 5},
      {ID: "function:store/store.go#Raw", Kind: "function", Name: "Raw", File: "store/store.go", StartLine: 9, Returns: "*sql.DB"},
    },
    Refs: []model.RawRef{
      {File: "cmd/main.go", Kind: "go_assign", Name: "st", Target: "store.Open#0", Line: 3, From: "main"},
      {File: "cmd/main.go", Kind: "go_assign", Name: "db", Target: "store.Raw#0", Line: 4, From: "main"},
      {File: "cmd/main.go", Kind: "go_assign", Name: "dup", Target: "store.Open#0", Line: 5, From: "main"},
      {File: "cmd/main.go", Kind: "go_assign", Name: "dup", Target: "store.Raw#0", Line: 6, From: "main"},
      {File: "cmd/main.go", Kind: "call", Name: "st.Get", Target: "local", Line: 7, From: "main"},
      {File: "cmd/main.go", Kind: "call", Name: "db.Close", Target: "local", Line: 8, From: "main"},
      {File: "cmd/main.go", Kind: "call", Name: "dup.Get", Target: "local", Line: 9, From: "main"},
    },
  }
  es := Resolve(in)
  if e := find(es, "calls", idMain, "st.Get"); e == nil || e.Dst != idGet || e.Status != "resolved" {
    t.Fatalf("st.Get must resolve through store.Open's return type: %+v", e)
  }
  if e := find(es, "calls", idMain, "db.Close"); e == nil || e.Status != "external" {
    t.Fatalf("db.Close is on *sql.DB and must be external: %+v", e)
  }
  if e := find(es, "calls", idMain, "dup.Get"); e == nil || e.Dst != "" || e.Status != "value" {
    t.Fatalf("conflicting assignments must stay value: %+v", e)
  }
}

func TestRetType(t *testing.T) {
  cases := []struct {
    ret  string
    idx  int
    want string
  }{
    {"(*Store, error)", 0, "*Store"},
    {"(*Store, error)", 1, "error"},
    {"(res *Store, err error)", 0, "*Store"},
    {"*sql.DB", 0, "*sql.DB"},
    {"*sql.DB", 1, ""},
    {"([]int, error)", 0, ""},
    {"(map[string]int, error)", 0, ""},
    {"", 0, ""},
  }
  for _, c := range cases {
    if got := retType(c.ret, c.idx); got != c.want {
      t.Errorf("retType(%q,%d) = %q, want %q", c.ret, c.idx, got, c.want)
    }
  }
}
