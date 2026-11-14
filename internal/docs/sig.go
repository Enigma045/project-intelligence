package docs

import (
  "strings"

  "pidx/internal/model"
)

var kindAbbr = map[string]string{
  "function": "fn", "method": "method", "class": "class", "struct": "struct", "interface": "iface",
  "type": "type", "element": "el", "form": "form", "field": "field", "css_rule": "css",
  "css_var": "var", "keyframes": "anim", "const": "const", "var": "var",
}

// Sig renders a compact one-line signature, e.g. "fn submitForm(event)" or "method Store.Open(path: string) -> error".
func Sig(s model.Symbol) string {
  abbr := kindAbbr[s.Kind]
  if abbr == "" {
    abbr = s.Kind
  }
  n := s.Qualified()
  switch s.Kind {
  case "function", "method":
    ps := make([]string, 0, len(s.Params))
    for _, p := range s.Params {
      if p.Type == "" || p.TypeSource == "unknown" {
        ps = append(ps, p.Name)
      } else {
        ps = append(ps, p.Name+": "+p.Type)
      }
    }
    n += "(" + strings.Join(ps, ", ") + ")"
    if s.Returns != "" {
      n += " -> " + s.Returns
    }
  case "field", "column":
    if s.Returns != "" {
      n += " : " + s.Returns
    }
  }
  return abbr + " " + n
}
