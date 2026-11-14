package extract

import (
  "regexp"

  "pidx/internal/model"
)

var rePkgClause = regexp.MustCompile(`(?m)^package\s+([A-Za-z_]\w*)`)

// goPackageRefs records the file's package clause: pkg.Func() calls use the package name, not the folder name.
func goPackageRefs(src []byte) []model.RawRef {
  if m := rePkgClause.FindSubmatch(src); m != nil {
    return []model.RawRef{{Kind: "go_package", Name: string(m[1]), Line: 1}}
  }
  return nil
}
