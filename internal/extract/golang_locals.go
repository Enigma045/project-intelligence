package extract

import (
  "go/ast"
  "go/token"
  "strconv"
  "strings"
)

// chain returns "a.b.c" for a pure identifier selector chain.
func chain(e ast.Expr) (string, bool) {
  switch x := e.(type) {
  case *ast.Ident:
    return x.Name, true
  case *ast.SelectorExpr:
    if p, ok := chain(x.X); ok {
      return p + "." + x.Sel.Name, true
    }
  }
  return "", false
}

// selName names a selector call; non-identifier receivers become "?.Method".
func selName(s *ast.SelectorExpr) string {
  if n, ok := chain(s); ok {
    return n
  }
  return "?." + s.Sel.Name
}

func isMajorVersion(s string) bool {
  if len(s) < 2 || s[0] != 'v' {
    return false
  }
  for i := 1; i < len(s); i++ {
    if s[i] < '0' || s[i] > '9' {
      return false
    }
  }
  return true
}

// importNames returns the identifiers that refer to imported packages in this file.
func importNames(f *ast.File) map[string]bool {
  m := map[string]bool{}
  for _, im := range f.Imports {
    if im.Name != nil {
      m[im.Name.Name] = true
      continue
    }
    p, err := strconv.Unquote(im.Path.Value)
    if err != nil {
      continue
    }
    segs := strings.Split(p, "/")
    name := segs[len(segs)-1]
    if len(segs) > 1 && isMajorVersion(name) {
      name = segs[len(segs)-2]
    }
    if i := strings.Index(name, "."); i > 0 {
      name = name[:i]
    }
    m[name] = true
  }
  return m
}

// inferType returns a type name only when it is syntactically certain.
func inferType(fset *token.FileSet, e ast.Expr) string {
  switch x := e.(type) {
  case *ast.UnaryExpr:
    if x.Op == token.AND {
      return inferType(fset, x.X)
    }
  case *ast.CompositeLit:
    if x.Type != nil {
      return typeStr(fset, x.Type)
    }
  case *ast.CallExpr:
    if id, ok := x.Fun.(*ast.Ident); ok && id.Name == "new" && len(x.Args) == 1 {
      return typeStr(fset, x.Args[0])
    }
  }
  return ""
}

// collectLocals maps every receiver, parameter and local variable of d to its declared type ("" = unknown).
// Flow-insensitive; conflicting declarations of one name degrade to unknown.
func collectLocals(fset *token.FileSet, d *ast.FuncDecl) map[string]string {
  m := map[string]string{}
  put := func(name, typ string) {
    if name == "" || name == "_" {
      return
    }
    if old, ok := m[name]; ok {
      if old != typ {
        m[name] = ""
      }
      return
    }
    m[name] = typ
  }
  fields := func(fl *ast.FieldList) {
    if fl == nil {
      return
    }
    for _, fd := range fl.List {
      t := typeStr(fset, fd.Type)
      for _, n := range fd.Names {
        put(n.Name, t)
      }
    }
  }
  fields(d.Recv)
  fields(d.Type.Params)
  fields(d.Type.Results)
  if d.Body == nil {
    return m
  }
  ast.Inspect(d.Body, func(n ast.Node) bool {
    switch x := n.(type) {
    case *ast.FuncLit:
      fields(x.Type.Params)
      fields(x.Type.Results)
    case *ast.AssignStmt:
      if x.Tok == token.DEFINE {
        for i, l := range x.Lhs {
          id, ok := l.(*ast.Ident)
          if !ok {
            continue
          }
          t := ""
          if len(x.Lhs) == len(x.Rhs) {
            t = inferType(fset, x.Rhs[i])
          }
          put(id.Name, t)
        }
      }
    case *ast.ValueSpec:
      for i, id := range x.Names {
        t := ""
        if x.Type != nil {
          t = typeStr(fset, x.Type)
        } else if len(x.Names) == len(x.Values) {
          t = inferType(fset, x.Values[i])
        }
        put(id.Name, t)
      }
    case *ast.RangeStmt:
      if x.Tok == token.DEFINE {
        for _, e := range []ast.Expr{x.Key, x.Value} {
          if id, ok := e.(*ast.Ident); ok {
            put(id.Name, "")
          }
        }
      }
    }
    return true
  })
  return m
}

// callTarget returns "" (static), "local" (call on a local value) or "local:<type>" (typed receiver).
func callTarget(name string, locals map[string]string, imps map[string]bool) string {
  segs := strings.Split(name, ".")
  root := segs[0]
  if imps[root] {
    return ""
  }
  typ, ok := locals[root]
  if !ok {
    return ""
  }
  if len(segs) == 2 && typ != "" {
    return "local:" + typ
  }
  return "local"
}
