package extract

import "testing"

func TestGoLocalTargets(t *testing.T) {
  src := "package main\n\nimport (\n  \"bytes\"\n  \"testing\"\n)\n\ntype S struct{}\n\nfunc (s *S) a() {}\n\nfunc (s *S) b(t *testing.T) {\n  var buf bytes.Buffer\n  buf.String()\n  s.a()\n  fn := func() {}\n  fn()\n  t.Fatal(\"x\")\n  bytes.NewReader(nil)\n  g := &S{}\n  g.a()\n  other.Do()\n}\n"
  f := Run("go", "a.go", []byte(src))
  if f.ParseStatus != "ok" {
    t.Fatalf("status %s %v", f.ParseStatus, f.ParseErrors)
  }
  cases := []struct{ name, target string }{
    {"buf.String", "local:bytes.Buffer"},
    {"s.a", "local:*S"},
    {"fn", "local"},
    {"t.Fatal", "local:*testing.T"},
    {"bytes.NewReader", ""},
    {"g.a", "local:S"},
    {"other.Do", ""},
  }
  for _, c := range cases {
    if !hasRef(f, "call", c.name, c.target) {
      t.Errorf("missing call ref %s target=%q in %+v", c.name, c.target, f.Refs)
    }
  }
}
