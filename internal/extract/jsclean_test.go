package extract

import "testing"

func TestJSRefCleanup(t *testing.T) {
  src := "const data = items.filter(x => x).map(y => y);\nel.addEventListener('click', e => { go(); });\nconst r = (async () => { await run(); })();\nfunction go() {}\nfunction run() {}\n"
  f := Run("javascript", "a.js", []byte(src))
  if hasRef(f, "call", "map", "") || !hasRef(f, "call", "?.map", "") {
    t.Errorf("method call after a dot must become ?.map: %+v", f.Refs)
  }
  if hasRef(f, "call", "async", "") {
    t.Errorf("async is a keyword, not a call: %+v", f.Refs)
  }
  if hasRef(f, "event", "click", "e") || !hasRef(f, "event", "click", "") {
    t.Errorf("arrow param must not be a handler name: %+v", f.Refs)
  }
  if !hasRef(f, "call", "go", "") || !hasRef(f, "call", "run", "") {
    t.Errorf("plain calls must be kept: %+v", f.Refs)
  }
}
