package query

import (
  "fmt"
  "testing"

  "pidx/internal/model"
)

func TestSharedWordDoesNotSeedBesideStrongMatch(t *testing.T) {
  syms := []model.Symbol{
    {ID: "route:main.go#ANY /Authorize", Kind: "route", Name: "ANY /Authorize", File: "main.go"},
    {ID: "function:Routes/Login2.go#Login", Kind: "function", Name: "Login", File: "Routes/Login2.go"},
  }
  for i := 0; i < 6; i++ {
    n := fmt.Sprintf("el%d-users", i)
    syms = append(syms, model.Symbol{ID: "element:a.html#" + n, Kind: "element", Name: n, File: "a.html"})
  }
  got := Seeds("Modify login to authorize users and redirect by role", syms, 6)
  if len(got) != 2 {
    t.Fatalf("want exactly the route and Login, got %+v", got)
  }
  for _, s := range got {
    if s.Sym.Kind == "element" {
      t.Fatalf("shared word seeded an element: %+v", s)
    }
  }
}
