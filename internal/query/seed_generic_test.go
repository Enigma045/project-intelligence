package query

import (
  "testing"

  "pidx/internal/model"
)

func TestGenericVerbsDoNotSeed(t *testing.T) {
  syms := []model.Symbol{
    {ID: "function:a.js#set", Kind: "function", Name: "set", File: "a.js"},
    {ID: "function:r.go#ResetPasswordAPI", Kind: "function", Name: "ResetPasswordAPI", File: "r.go"},
  }
  got := Seeds("let the user set a new password", syms, 6)
  for _, s := range got {
    if s.Sym.Name == "set" {
      t.Fatalf("generic verb seeded a function named set: %+v", s)
    }
  }
  if len(got) != 1 || got[0].Sym.Name != "ResetPasswordAPI" {
    t.Fatalf("want only ResetPasswordAPI, got %+v", got)
  }
}
