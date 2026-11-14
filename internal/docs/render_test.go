package docs

import (
  "strings"
  "testing"

  "pidx/internal/model"
)

func TestRender(t *testing.T) {
  in := Input{
    Files: []FileEntry{
      {Path: "cmd/main.go", Language: "go", Hash: "h1", Size: 100},
      {Path: "web/a.js", Language: "javascript", Hash: "h2", Size: 50},
    },
    Symbols: []model.Symbol{
      {ID: "function:cmd/main.go#main", Kind: "function", Name: "main", File: "cmd/main.go", StartLine: 1, EndLine: 3},
      {ID: "function:web/a.js#submit", Kind: "function", Name: "submit", File: "web/a.js", StartLine: 2, EndLine: 5,
        Params: []model.Param{{Name: "e", TypeSource: "unknown"}}},
    },
  }
  out := Render(in)
  if !strings.Contains(out.Root, "## Modules") || !strings.Contains(out.Root, "cmd/main.go:1") {
    t.Fatalf("bad root:\n%s", out.Root)
  }
  if s := out.Shards["web.md"]; !strings.Contains(s, "fn submit(e)") {
    t.Fatalf("bad web shard: %q", s)
  }
  if _, ok := out.Shards["cmd.md"]; !ok {
    t.Fatal("missing cmd shard")
  }
  if again := Render(in); again.Root != out.Root {
    t.Fatal("render is not deterministic")
  }
}
