package extract

import (
  "go/ast"
  "go/token"
  "strconv"

  "pidx/internal/model"
)

// goAssignRefs records locals assigned from a call: x, err := pkg.Func(...) gives go_assign refs
// (Name=local, Target="pkg.Func#resultIndex"). The resolver turns them into receiver types using the
// callee's signature.
func goAssignRefs(fset *token.FileSet, d *ast.FuncDecl, q string) []model.RawRef {
  if d.Body == nil {
    return nil
  }
  var out []model.RawRef
  emit := func(name string, call *ast.CallExpr, idx int, pos token.Pos) {
    if name == "" || name == "_" {
      return
    }
    fn := ""
    switch f := call.Fun.(type) {
    case *ast.Ident:
      fn = f.Name
    case *ast.SelectorExpr:
      if n, ok := chain(f); ok {
        fn = n
      }
    }
    if fn == "" {
      return
    }
    out = append(out, model.RawRef{Kind: "go_assign", Name: name, Target: fn + "#" + strconv.Itoa(idx), Line: fset.Position(pos).Line, From: q})
  }
  ast.Inspect(d.Body, func(n ast.Node) bool {
    switch x := n.(type) {
    case *ast.AssignStmt:
      if x.Tok != token.DEFINE {
        return true
      }
      if len(x.Rhs) == 1 {
        if c, ok := x.Rhs[0].(*ast.CallExpr); ok {
          for i, l := range x.Lhs {
            if id, ok := l.(*ast.Ident); ok {
              emit(id.Name, c, i, x.Pos())
            }
          }
        }
      } else if len(x.Lhs) == len(x.Rhs) {
        for i, r := range x.Rhs {
          if c, ok := r.(*ast.CallExpr); ok {
            if id, ok := x.Lhs[i].(*ast.Ident); ok {
              emit(id.Name, c, 0, x.Pos())
            }
          }
        }
      }
    case *ast.ValueSpec:
      if len(x.Values) == 1 {
        if c, ok := x.Values[0].(*ast.CallExpr); ok {
          for i, id := range x.Names {
            emit(id.Name, c, i, x.Pos())
          }
        }
      }
    }
    return true
  })
  return out
}
