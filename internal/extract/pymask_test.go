package extract

import "testing"

func TestPythonTripleQuotedStringsAreNotCode(t *testing.T) {
  src := "JS = \"\"\"\nasync function f() {\n  go(1);\n  els.forEach(x => x);\n}\n\"\"\"\n\ndef run():\n    \"\"\"doc: ignored(1)\"\"\"\n    real(2)\n"
  f := Run("python", "p.py", []byte(src))
  for _, n := range []string{"go", "f", "forEach", "ignored"} {
    if hasRef(f, "call", n, "") {
      t.Errorf("call %q from inside a triple-quoted string must not be a ref: %+v", n, f.Refs)
    }
  }
  if !hasRef(f, "call", "real", "") {
    t.Errorf("a real call after the string must be kept: %+v", f.Refs)
  }
}
