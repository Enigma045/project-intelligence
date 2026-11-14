package resolve

import "pidx/internal/model"

// goNames knows each Go file's package name and the aliases its imports use. In Go the qualifier in
// pkg.Func() is the package name (or the import alias), not the folder name.
type goNames struct {
  pkg   map[string]string
  alias map[string]string
}

func newGoNames(refs []model.RawRef) *goNames {
  g := &goNames{pkg: map[string]string{}, alias: map[string]string{}}
  for _, r := range refs {
    switch r.Kind {
    case "go_package":
      g.pkg[r.File] = r.Name
    case "go_import":
      g.alias[r.File+"|"+r.Target] = r.Name
    }
  }
  return g
}

// extra returns additional qualifiers for one import: its alias and the package names of the files it points to.
func (g *goNames) extra(file, imp string, files []string) []string {
  var out []string
  if a := g.alias[file+"|"+imp]; a != "" {
    out = append(out, a)
  }
  seen := map[string]bool{}
  for _, f := range files {
    if p := g.pkg[f]; p != "" && !seen[p] {
      seen[p] = true
      out = append(out, p)
    }
  }
  return out
}
