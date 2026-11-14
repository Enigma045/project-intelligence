package resolve

// isPkgVar reports whether name is a package-level var/const defined in one of the imported package's files.
func (r *resolver) isPkgVar(info *importInfo, name string) bool {
  in := map[string]bool{}
  for _, f := range info.files {
    in[f] = true
  }
  for _, s := range r.vars[name] {
    if in[s.File] {
      return true
    }
  }
  return false
}

// noCandidate classifies a dotted call with no project candidate. In Go the receiver is a chained call
// result (dynamic). In JS/Python it is almost always a DOM/built-in method on a value.
func (r *resolver) noCandidate(c *fileCtx) []hit {
  if c.lang == "go" {
    return []hit{{"", "dynamic", 0.2}}
  }
  return []hit{{"", "value", 0.5}}
}
