package query

import (
  "strings"
  "testing"

  "pidx/internal/model"
)

func TestTokens(t *testing.T) {
  got := strings.Join(Tokens("submitForm#registrationForm_x2"), " ")
  if got != "submit form registration form x2" {
    t.Fatalf("bad tokens: %q", got)
  }
}

func TestSeeds(t *testing.T) {
  syms := []model.Symbol{
    {ID: "function:web/register.js#submitForm", Kind: "function", Name: "submitForm", File: "web/register.js"},
    {ID: "function:web/register.js#isValidEmail", Kind: "function", Name: "isValidEmail", File: "web/register.js"},
    {ID: "function:web/other.js#renderChart", Kind: "function", Name: "renderChart", File: "web/other.js"},
  }
  got := Seeds("Modify the registration form to validate the user's email before submitting it", syms, 6)
  ids := map[string]bool{}
  for _, s := range got {
    ids[s.Sym.ID] = true
  }
  if !ids[syms[0].ID] || !ids[syms[1].ID] || ids[syms[2].ID] {
    t.Fatalf("bad seeds: %+v", got)
  }
}
