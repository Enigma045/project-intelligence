package resolve

import (
  "path"
  "regexp"
  "strings"

  "pidx/internal/model"
)

var simpleType = regexp.MustCompile(`^\*?[A-Za-z_]\w*(\.[A-Za-z_]\w*)?$`)

// isVar reports whether name is a package-level var/const visible from file c.
func (r *resolver) isVar(c *fileCtx, name string) bool {
  for _, s := range r.vars[name] {
    if s.File == c.path || (c.lang == "go" && path.Dir(s.File) == c.dir) {
      return true
    }
  }
  return false
}

func typedHits(cs []*model.Symbol) []hit {
  if len(cs) == 0 {
    return []hit{{"", "dynamic", 0.3}}
  }
  return hitsFrom(cs, "qualified", false)
}

// viaLocal resolves a call whose root is a local value. typ is the declared type ("" = unknown).
func (r *resolver) viaLocal(c *fileCtx, name, typ string) []hit {
  if typ == "" || !simpleType.MatchString(typ) {
    return []hit{{"", "value", 0.9}}
  }
  segs := strings.Split(name, ".")
  method := segs[len(segs)-1]
  t := strings.TrimPrefix(typ, "*")
  if i := strings.Index(t, "."); i > 0 {
    pkg, tn := t[:i], t[i+1:]
    info, ok := c.bases[pkg]
    if !ok {
      return []hit{{"", "value", 0.5}}
    }
    if !info.internal {
      return ext(0.95)
    }
    in := map[string]bool{}
    for _, f := range info.files {
      in[f] = true
    }
    var cs []*model.Symbol
    for _, s := range r.filter(method, "method") {
      if s.Parent == tn && in[s.File] {
        cs = append(cs, s)
      }
    }
    return typedHits(cs)
  }
  if builtins[t] {
    return ext(0.9)
  }
  var cs []*model.Symbol
  for _, s := range r.filter(method, "method") {
    if s.Parent == t && path.Dir(s.File) == c.dir {
      cs = append(cs, s)
    }
  }
  return typedHits(cs)
}
