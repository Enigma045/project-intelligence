package query

import (
  "fmt"
  "strings"
  "testing"

  "pidx/internal/model"
)

func TestGenericWordsDoNotSeed(t *testing.T) {
  var syms []model.Symbol
  for i := 0; i < 20; i++ {
    n := fmt.Sprintf("M%d", i)
    syms = append(syms, model.Symbol{ID: "method:store/x.go#Store." + n, Kind: "method", Name: n, Parent: "Store", File: "store/x.go"})
  }
  users := model.Symbol{ID: "table:db/schema.sql#users", Kind: "table", Name: "users", File: "db/schema.sql"}
  syms = append(syms, users)
  got := Seeds("Registration should store the email in the users table", syms, 6)
  foundUsers := false
  for _, s := range got {
    if strings.HasPrefix(s.Sym.ID, "method:") {
      t.Fatalf("generic word seeded a Store method: %+v", s)
    }
    if s.Sym.ID == users.ID {
      foundUsers = true
    }
  }
  if !foundUsers {
    t.Fatalf("table users must seed: %+v", got)
  }
}
