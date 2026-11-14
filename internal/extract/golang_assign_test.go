package extract

import "testing"

func TestGoAssignRefs(t *testing.T) {
  src := "package main\n\nfunc main() {\n  st, err := store.Open(\"x\")\n  _ = err\n  st.Get()\n  a, b := f(), g()\n  var v = h()\n  _, _, _ = a, b, v\n}\n"
  f := Run("go", "a.go", []byte(src))
  if f.ParseStatus != "ok" {
    t.Fatalf("status %s %v", f.ParseStatus, f.ParseErrors)
  }
  for _, c := range []struct{ name, target string }{
    {"st", "store.Open#0"}, {"err", "store.Open#1"}, {"a", "f#0"}, {"b", "g#0"}, {"v", "h#0"},
  } {
    if !hasRef(f, "go_assign", c.name, c.target) {
      t.Errorf("missing go_assign %s -> %s in %+v", c.name, c.target, f.Refs)
    }
  }
  if !hasRef(f, "call", "st.Get", "local") {
    t.Errorf("st.Get must stay a local-value call for the resolver: %+v", f.Refs)
  }
}
