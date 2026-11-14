package resolve

import "strings"

// subst replaces a leading ${CONST} in a request URL with the value of a top-level string constant defined
// in the same file (const API_BASE_URL = '/api'). Returns the new text and whether a substitution happened.
func (b *beCtx) subst(file, raw string) (string, bool) {
  if !strings.HasPrefix(raw, "${") {
    return raw, false
  }
  end := strings.Index(raw, "}")
  if end < 3 {
    return raw, false
  }
  v, ok := b.consts[file+"|"+raw[2:end]]
  if !ok {
    return raw, false
  }
  return v + raw[end+1:], true
}
