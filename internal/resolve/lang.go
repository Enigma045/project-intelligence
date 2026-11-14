package resolve

import (
  "path"
  "strings"
)

// langGroup maps a file to a language family; calls never cross families.
func langGroup(file string) string {
  switch strings.ToLower(path.Ext(file)) {
  case ".go":
    return "go"
  case ".py":
    return "py"
  case ".js", ".jsx", ".mjs", ".cjs", ".ts", ".tsx", ".html", ".htm":
    return "js"
  }
  return ""
}

func groupOfLang(l string) string {
  switch l {
  case "go":
    return "go"
  case "python":
    return "py"
  case "javascript", "typescript", "html":
    return "js"
  }
  return ""
}

// sameLang drops candidates defined in a different language than the caller (JS cannot call Go).
// If every candidate was cross-language the call becomes dynamic instead of a false edge.
func (r *resolver) sameLang(c *fileCtx, hs []hit) []hit {
  g := groupOfLang(c.lang)
  if g == "" {
    return hs
  }
  var out []hit
  dropped := false
  for _, h := range hs {
    if h.dst == "" {
      out = append(out, h)
      continue
    }
    if s := r.byID[h.dst]; s != nil {
      if lg := langGroup(s.File); lg != "" && lg != g {
        dropped = true
        continue
      }
    }
    out = append(out, h)
  }
  if len(out) == 0 && dropped {
    return []hit{{"", "dynamic", 0.2}}
  }
  return out
}
