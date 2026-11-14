package extract

import (
  "regexp"

  "pidx/internal/model"
)

var reJSConst = regexp.MustCompile(`^(?:export\s+)?const\s+([A-Za-z_$][\w$]*)\s*=\s*(?:'([^'\n]*)'|"([^"\n]*)")\s*;?\s*(?://.*)?$`)

// addJSConsts records top-level (column 0) string constants as js_const refs (Name=identifier, Target=value).
// The backend resolver uses them to resolve fetch(`${API_BASE_URL}/path`).
func addJSConsts(f *model.FileFacts, lines []string) {
  for i, l := range lines {
    m := reJSConst.FindStringSubmatch(l)
    if m == nil {
      continue
    }
    v := m[2]
    if v == "" {
      v = m[3]
    }
    if v == "" || len(v) > 200 {
      continue
    }
    f.Refs = append(f.Refs, model.RawRef{Kind: "js_const", Name: m[1], Target: v, Line: i + 1})
  }
}
