package index

import (
  "testing"

  "pidx/internal/model"
)

func one(got []string, want string) bool { return len(got) == 1 && got[0] == want }

func TestDiffSymbols(t *testing.T) {
  old := []model.Symbol{
    {ID: "function:a.go#A", SigHash: "s1", BodyHash: "b1"},
    {ID: "function:a.go#B", SigHash: "s2", BodyHash: "b2"},
    {ID: "function:a.go#C", SigHash: "s3", BodyHash: "b3"},
    {ID: "function:a.go#D", SigHash: "s4", BodyHash: "b4"},
  }
  cur := []model.Symbol{
    {ID: "function:a.go#A", SigHash: "s1", BodyHash: "b1"},
    {ID: "function:a.go#B", SigHash: "s2x", BodyHash: "b2"},
    {ID: "function:a.go#C", SigHash: "s3", BodyHash: "b3x"},
    {ID: "function:a.go#E", SigHash: "s5", BodyHash: "b5"},
  }
  d := DiffSymbols(old, cur)
  if !one(d.Added, "function:a.go#E") || !one(d.Removed, "function:a.go#D") || !one(d.SigChanged, "function:a.go#B") || !one(d.BodyChanged, "function:a.go#C") {
    t.Fatalf("bad diff: %+v", d)
  }
  if d.Empty() {
    t.Fatal("diff should not be empty")
  }
}

func TestIDFile(t *testing.T) {
  if idFile("function:cmd/main.go#main") != "cmd/main.go" || idFile("file:web/api.js") != "web/api.js" {
    t.Fatal("idFile wrong")
  }
}
