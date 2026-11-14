package edit

import (
  "strings"
  "testing"

  "pidx/internal/model"
)

func envFor(files map[string]string, syms map[string]SymRange) Env {
  return Env{
    Read: func(rel string) ([]byte, bool, error) {
      s, ok := files[rel]
      return []byte(s), ok, nil
    },
    Lookup: func(q string) ([]SymRange, error) {
      if r, ok := syms[q]; ok {
        return []SymRange{r}, nil
      }
      return nil, nil
    },
  }
}

func TestSymbolAndTextOps(t *testing.T) {
  src := "a\nfunc f() {\n  1\n}\nb\nfunc g() {\n  2\n}\n"
  h := model.HashBytes([]byte(src))
  env := envFor(map[string]string{"x.go": src}, map[string]SymRange{
    "f": {File: "x.go", Start: 2, End: 4, FileHash: h},
    "g": {File: "x.go", Start: 6, End: 8, FileHash: h},
  })
  st, err := Plan([]Op{
    {Op: "replace_symbol", Symbol: "f", Text: "func f() {\n  10\n}"},
    {Op: "insert_after_symbol", Symbol: "g", Text: "\nfunc h() {}"},
    {File: "x.go", Op: "replace", Old: "\nb\n", New: "\nB\n"},
  }, env, false)
  if err != nil {
    t.Fatal(err)
  }
  want := "a\nfunc f() {\n  10\n}\nB\nfunc g() {\n  2\n}\n\nfunc h() {}\n"
  if len(st) != 1 || string(st[0].After) != want {
    t.Fatalf("got %+v", st)
  }
}

func TestDeleteSymbol(t *testing.T) {
  src := "a\n\nfunc f() {\n}\n\nb\n"
  h := model.HashBytes([]byte(src))
  env := envFor(map[string]string{"x.go": src}, map[string]SymRange{"f": {File: "x.go", Start: 3, End: 4, FileHash: h}})
  st, err := Plan([]Op{{Op: "delete_symbol", Symbol: "f"}}, env, false)
  if err != nil || len(st) != 1 || string(st[0].After) != "a\n\nb\n" {
    t.Fatalf("got %+v err=%v", st, err)
  }
}

func TestRejections(t *testing.T) {
  files := map[string]string{"x.txt": "aa\naa\n"}
  env := envFor(files, map[string]SymRange{"s": {File: "x.txt", Start: 1, End: 1, FileHash: "bogus"}})
  cases := []struct {
    op   Op
    want string
  }{
    {Op{File: "x.txt", Op: "replace", Old: "aa", New: "b"}, "matches 2 times"},
    {Op{File: "x.txt", Op: "replace", Old: "zz", New: "b"}, "not found"},
    {Op{File: "x.txt", Op: "replace", Old: "aa\naa", New: "b", BaseHash: "sha256:0"}, "base_hash mismatch"},
    {Op{Op: "replace_symbol", Symbol: "s", Text: "x"}, "stale"},
    {Op{Op: "replace_symbol", Symbol: "nope", Text: "x"}, "matched 0"},
    {Op{File: "x.txt", Op: "create", Text: "x"}, "already exists"},
    {Op{File: "../evil.txt", Op: "create", Text: "x"}, "escapes"},
    {Op{File: "x.txt", Op: "bogus"}, "unknown op"},
  }
  for _, c := range cases {
    _, err := Plan([]Op{c.op}, env, false)
    if err == nil || !strings.Contains(err.Error(), c.want) {
      t.Errorf("%+v: want error containing %q, got %v", c.op, c.want, err)
    }
  }
}

func TestCRLFPreserved(t *testing.T) {
  env := envFor(map[string]string{"x.txt": "a\r\nb\r\n"}, nil)
  st, err := Plan([]Op{{File: "x.txt", Op: "replace", Old: "a", New: "x"}}, env, false)
  if err != nil || len(st) != 1 || string(st[0].After) != "x\r\nb\r\n" {
    t.Fatalf("got %+v err=%v", st, err)
  }
}

func TestCreate(t *testing.T) {
  env := envFor(map[string]string{}, nil)
  st, err := Plan([]Op{{File: "new/a.txt", Op: "create", Text: "hi\n"}}, env, false)
  if err != nil || len(st) != 1 || st[0].Existed || string(st[0].After) != "hi\n" {
    t.Fatalf("got %+v err=%v", st, err)
  }
}

func TestSafePath(t *testing.T) {
  for _, p := range []string{"", "/abs", "../x", "a\\b", "C:/x", ".git/config", ".project-intelligence/index.db", "."} {
    if _, err := SafePath(p); err == nil {
      t.Errorf("SafePath(%q) should fail", p)
    }
  }
  if p, err := SafePath("a/./b.go"); err != nil || p != "a/b.go" {
    t.Errorf("SafePath normalise: %q %v", p, err)
  }
}

func TestSyntaxCheck(t *testing.T) {
  ok := []struct{ f, c string }{
    {"a.go", "package a\nfunc f() {}\n"},
    {"a.json", "{\"a\": 1}"},
    {"a.css", "a { color: red; }"},
  }
  for _, c := range ok {
    if checked, err := SyntaxCheck(c.f, []byte(c.c)); !checked || err != nil {
      t.Errorf("%s should pass: checked=%v err=%v", c.f, checked, err)
    }
  }
  bad := []struct{ f, c string }{
    {"a.go", "package a\nfunc f( {\n"},
    {"a.json", "{\"a\": }"},
    {"a.css", "a { color: red;"},
  }
  for _, c := range bad {
    if checked, err := SyntaxCheck(c.f, []byte(c.c)); !checked || err == nil {
      t.Errorf("%s should fail: checked=%v err=%v", c.f, checked, err)
    }
  }
  if checked, _ := SyntaxCheck("a.unknown", []byte("x")); checked {
    t.Error("unknown type must report checked=false")
  }
}
