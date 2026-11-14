package extract

import (
  "testing"

  "pidx/internal/model"
)

func find(f model.FileFacts, name string) *model.Symbol {
  for i := range f.Symbols {
    if f.Symbols[i].Name == name {
      return &f.Symbols[i]
    }
  }
  return nil
}

func hasRef(f model.FileFacts, kind, name, target string) bool {
  for _, r := range f.Refs {
    if r.Kind == kind && r.Name == name && r.Target == target {
      return true
    }
  }
  return false
}

func TestGo(t *testing.T) {
  src := "package main\n\nimport \"fmt\"\n\ntype Server struct{}\n\nfunc (s *Server) Handle(a int, b string) (int, error) {\n  fmt.Println(a)\n  return 0, nil\n}\n\nfunc helper() {}\n"
  f := Run("go", "main.go", []byte(src))
  if f.ParseStatus != "ok" {
    t.Fatalf("status %s %v", f.ParseStatus, f.ParseErrors)
  }
  h := find(f, "Handle")
  if h == nil || h.Kind != "method" || h.Parent != "Server" || len(h.Params) != 2 || h.Returns != "(int, error)" {
    t.Fatalf("bad Handle: %+v", h)
  }
  if find(f, "Server") == nil || find(f, "helper") == nil {
    t.Fatal("missing symbols")
  }
  if !hasRef(f, "call", "fmt.Println", "") {
    t.Fatal("missing call ref")
  }
}

func TestJS(t *testing.T) {
  src := "const button = document.getElementById(\"submitBtn\");\nbutton.addEventListener(\"click\", submitForm);\nfunction submitForm(event) {\n  console.log(\"Submitted\");\n}\n"
  f := Run("javascript", "dashboard.js", []byte(src))
  s := find(f, "submitForm")
  if s == nil || s.StartLine != 3 || s.EndLine != 5 || len(s.Params) != 1 || s.Params[0].Name != "event" {
    t.Fatalf("bad submitForm: %+v", s)
  }
  if s.Params[0].TypeSource != "unknown" {
    t.Fatal("untyped param must be unknown")
  }
  if !hasRef(f, "dom_id", "submitBtn", "") || !hasRef(f, "event", "click", "submitForm") {
    t.Fatalf("missing dom/event refs: %+v", f.Refs)
  }
  if !hasRef(f, "call", "console.log", "") {
    t.Fatal("missing console.log call")
  }
}

func TestPython(t *testing.T) {
  src := "class Greeter:\n    def hello(self, name: str = \"x\") -> str:\n        return format_name(name)\n\ndef main():\n    Greeter().hello(\"a\")\n"
  f := Run("python", "app.py", []byte(src))
  h := find(f, "hello")
  if h == nil || h.Kind != "method" || h.Parent != "Greeter" || h.Returns != "str" || len(h.Params) != 2 || h.Params[1].Type != "str" {
    t.Fatalf("bad hello: %+v", h)
  }
  if find(f, "main") == nil || find(f, "Greeter") == nil {
    t.Fatal("missing symbols")
  }
  if !hasRef(f, "call", "format_name", "") {
    t.Fatalf("missing call ref: %+v", f.Refs)
  }
}
