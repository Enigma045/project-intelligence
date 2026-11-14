package extract

import "testing"

func TestGoPackageAndAliasRefs(t *testing.T) {
  src := "package user_logs\n\nimport ul \"mod/x\"\nimport _ \"mod/side\"\nimport \"mod/plain\"\n\nfunc F() { ul.Do() }\n"
  f := Run("go", "a/u.go", []byte(src))
  if f.ParseStatus != "ok" {
    t.Fatalf("status %s %v", f.ParseStatus, f.ParseErrors)
  }
  if !hasRef(f, "go_package", "user_logs", "") {
    t.Errorf("missing go_package ref: %+v", f.Refs)
  }
  if !hasRef(f, "go_import", "ul", "mod/x") {
    t.Errorf("missing go_import alias ref: %+v", f.Refs)
  }
  if hasRef(f, "go_import", "_", "mod/side") {
    t.Errorf("blank imports must not produce an alias ref")
  }
}
