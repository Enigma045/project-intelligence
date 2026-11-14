package extract

import "testing"

func TestHTMLExtract(t *testing.T) {
  src := `<!DOCTYPE html>
<html>
<head>
<link rel="stylesheet" href="style.css">
<style>
.box { color: red; }
</style>
</head>
<body>
<form id="registrationForm" action="/api/register">
  <input id="email" name="email" type="text">
  <button id="submitBtn" class="primary-button big" onclick="submitForm()">Submit</button>
</form>
<script src="register.js"></script>
<script>
function inlineFn() {
  document.getElementById('email');
}
</script>
</body>
</html>
`
  f := Run("html", "web/register.html", []byte(src))
  btn := find(f, "button#submitBtn")
  if btn == nil || btn.Kind != "element" || btn.StartLine != 12 {
    t.Fatalf("bad button: %+v", btn)
  }
  form := find(f, "form#registrationForm")
  if form == nil || form.Kind != "form" || form.StartLine != 10 || form.EndLine != 13 {
    t.Fatalf("bad form: %+v", form)
  }
  fld := find(f, "email")
  if fld == nil || fld.Kind != "field" || fld.Parent != "form#registrationForm" || fld.Returns != "text" {
    t.Fatalf("bad field: %+v", fld)
  }
  fn := find(f, "inlineFn")
  if fn == nil || fn.StartLine != 16 || fn.EndLine != 18 {
    t.Fatalf("bad inline fn: %+v", fn)
  }
  rule := find(f, ".box")
  if rule == nil || rule.Kind != "css_rule" || rule.StartLine != 6 {
    t.Fatalf("bad inline css rule: %+v", rule)
  }
  for _, c := range []struct{ kind, name, target string }{
    {"script_src", "register.js", ""},
    {"style_href", "style.css", ""},
    {"html_handler", "onclick", "submitForm()"},
    {"html_class", "primary-button", "button"},
  } {
    if !hasRef(f, c.kind, c.name, c.target) {
      t.Errorf("missing ref %+v in %+v", c, f.Refs)
    }
  }
  found := false
  for _, r := range f.Refs {
    if r.Kind == "dom_id" && r.Name == "email" && r.Line == 17 && r.From == "inlineFn" {
      found = true
    }
  }
  if !found {
    t.Errorf("inline dom_id ref missing or wrong line: %+v", f.Refs)
  }
}

func TestCSSExtract(t *testing.T) {
  src := `/* c */
:root { --brand: #fff; }
.primary-button, #submitBtn { color: var(--brand); }
@media (max-width: 600px) {
  .primary-button { color: red; }
}
@keyframes spin { from { opacity: 0 } to { opacity: 1 } }
`
  f := Run("css", "style.css", []byte(src))
  if r := find(f, ".primary-button, #submitBtn"); r == nil || r.Kind != "css_rule" || r.StartLine != 3 {
    t.Fatalf("bad selector-list rule: %+v", r)
  }
  m := find(f, ".primary-button")
  if m == nil || m.Parent != "@media (max-width: 600px)" || m.StartLine != 5 {
    t.Fatalf("bad media rule: %+v", m)
  }
  if v := find(f, "--brand"); v == nil || v.Kind != "css_var" || v.StartLine != 2 {
    t.Fatalf("bad css var: %+v", v)
  }
  if k := find(f, "spin"); k == nil || k.Kind != "keyframes" || k.StartLine != 7 {
    t.Fatalf("bad keyframes: %+v", k)
  }
  for _, c := range []struct{ kind, name string }{
    {"css_id", "submitBtn"}, {"css_class", "primary-button"}, {"css_var_use", "--brand"},
  } {
    if !hasRef(f, c.kind, c.name, "") {
      t.Errorf("missing ref %+v in %+v", c, f.Refs)
    }
  }
}

func TestJSDOM(t *testing.T) {
  src := `const button = document.getElementById("submitBtn");
button.addEventListener("click", submitForm);
document.querySelector('#email').addEventListener('input', function () {});
button.classList.add('active');
function submitForm() {}
`
  f := Run("javascript", "dashboard.js", []byte(src))
  for _, c := range []struct{ kind, name, target string }{
    {"dom_listen", "#submitBtn", "click:submitForm"},
    {"dom_listen", "#email", "input:"},
    {"dom_class", "active", "add"},
    {"dom_id", "submitBtn", ""},
  } {
    if !hasRef(f, c.kind, c.name, c.target) {
      t.Errorf("missing ref %+v in %+v", c, f.Refs)
    }
  }
}
