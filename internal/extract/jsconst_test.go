package extract

import "testing"

func TestJSConstRefs(t *testing.T) {
  src := "const API_BASE_URL = '/api'; \nconst NAME = \"x\" // note\nfunction f() {\n  const INNER = '/no';\n}\n"
  f := Run("javascript", "a.js", []byte(src))
  if !hasRef(f, "js_const", "API_BASE_URL", "/api") {
    t.Errorf("missing API_BASE_URL const: %+v", f.Refs)
  }
  if !hasRef(f, "js_const", "NAME", "x") {
    t.Errorf("missing double-quoted const with trailing comment: %+v", f.Refs)
  }
  if hasRef(f, "js_const", "INNER", "/no") {
    t.Errorf("indented (non top-level) const must not be recorded")
  }
}
