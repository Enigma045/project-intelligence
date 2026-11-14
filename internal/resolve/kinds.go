package resolve

// dottedKinds lists which symbol kinds an `x.name()` call can resolve to when x is not a known package or
// typed local. In Go that can only be a method; plain functions are reached through a package qualifier.
func dottedKinds(lang string) []string {
  if lang == "go" {
    return []string{"method"}
  }
  return []string{"function", "method", "class"}
}
