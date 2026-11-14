package resolve

import (
  "strconv"
  "strings"

  "pidx/internal/model"
)

// buildAssigns maps file|function|local to "callName#resultIndex" for locals assigned from a call
// (x, err := pkg.Func(...)). A local assigned from two different calls is marked "!" (conflict) and not used.
func buildAssigns(refs []model.RawRef) map[string]string {
  m := map[string]string{}
  for _, r := range refs {
    if r.Kind != "go_assign" {
      continue
    }
    k := r.File + "|" + r.From + "|" + r.Name
    if old, ok := m[k]; ok && old != r.Target {
      m[k] = "!"
      continue
    }
    m[k] = r.Target
  }
  return m
}

func splitComma(s string) []string {
  var out []string
  depth, start := 0, 0
  for i := 0; i < len(s); i++ {
    switch s[i] {
    case '(', '[', '{':
      depth++
    case ')', ']', '}':
      depth--
    case ',':
      if depth == 0 {
        out = append(out, s[start:i])
        start = i + 1
      }
    }
  }
  return append(out, s[start:])
}

// retType returns the idx-th result type of a Go signature's result list, or "" when it is not a simple type.
func retType(returns string, idx int) string {
  s := strings.TrimSpace(returns)
  if s == "" {
    return ""
  }
  parts := []string{s}
  if strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")") {
    parts = splitComma(s[1 : len(s)-1])
  }
  if idx < 0 || idx >= len(parts) {
    return ""
  }
  p := strings.TrimSpace(parts[idx])
  for _, bad := range []string{"func", "map[", "chan", "[]", "interface", "struct"} {
    if strings.HasPrefix(p, bad) {
      return ""
    }
  }
  if i := strings.IndexByte(p, ' '); i > 0 {
    p = strings.TrimSpace(p[i+1:])
  }
  return p
}

// viaLocalInferred resolves x.Method() where x has no declared type but was assigned from a call whose
// signature gives the type: x, err := store.Open(...). The type is named in the callee's file, so it is
// resolved with the callee file's imports and package. Anything it cannot prove falls back to viaLocal.
func (r *resolver) viaLocalInferred(c *fileCtx, from *model.Symbol, name, typ string) []hit {
  if typ != "" || from == nil {
    return r.viaLocal(c, name, typ)
  }
  segs := strings.Split(name, ".")
  if len(segs) != 2 {
    return r.viaLocal(c, name, typ)
  }
  a := r.assigns[c.path+"|"+from.Qualified()+"|"+segs[0]]
  i := strings.LastIndex(a, "#")
  if a == "" || a == "!" || i < 0 {
    return r.viaLocal(c, name, typ)
  }
  idx, err := strconv.Atoi(a[i+1:])
  if err != nil {
    return r.viaLocal(c, name, typ)
  }
  hs := r.targetStatic(c, from, a[:i])
  if len(hs) != 1 || hs[0].dst == "" || (hs[0].status != "resolved" && hs[0].status != "inferred") {
    return r.viaLocal(c, name, typ)
  }
  sym := r.byID[hs[0].dst]
  if sym == nil || (sym.Kind != "function" && sym.Kind != "method") {
    return r.viaLocal(c, name, typ)
  }
  t := retType(sym.Returns, idx)
  cc := r.ctxs[sym.File]
  if t == "" || cc == nil {
    return r.viaLocal(c, name, typ)
  }
  out := r.viaLocal(cc, name, t)
  if hs[0].status != "resolved" {
    for k := range out {
      if out[k].status == "resolved" {
        out[k].status = "inferred"
      }
    }
  }
  return out
}
