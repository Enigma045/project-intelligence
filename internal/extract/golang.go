package extract

import (
  "bytes"
  "go/ast"
  "go/parser"
  "go/printer"
  "go/token"
  "strconv"
  "strings"

  "pidx/internal/model"
)

func typeStr(fset *token.FileSet, e ast.Expr) string {
  var b bytes.Buffer
  _ = printer.Fprint(&b, fset, e)
  return b.String()
}

func srcSlice(src []byte, fset *token.FileSet, a, b token.Pos) []byte {
  s, e := fset.Position(a).Offset, fset.Position(b).Offset
  if s < 0 || e > len(src) || s > e {
    return nil
  }
  return src[s:e]
}

func goVis(name string) string {
  if ast.IsExported(name) {
    return "exported"
  }
  return "private"
}

func goParams(fset *token.FileSet, fl *ast.FieldList) []model.Param {
  if fl == nil {
    return nil
  }
  var out []model.Param
  for _, f := range fl.List {
    t := typeStr(fset, f.Type)
    if len(f.Names) == 0 {
      out = append(out, model.Param{Name: "_", Type: t, TypeSource: "declared"})
      continue
    }
    for _, n := range f.Names {
      out = append(out, model.Param{Name: n.Name, Type: t, TypeSource: "declared"})
    }
  }
  return out
}

func goResults(fset *token.FileSet, fl *ast.FieldList) string {
  if fl == nil || len(fl.List) == 0 {
    return ""
  }
  var parts []string
  named := false
  for _, f := range fl.List {
    t := typeStr(fset, f.Type)
    if len(f.Names) == 0 {
      parts = append(parts, t)
      continue
    }
    named = true
    for _, n := range f.Names {
      parts = append(parts, n.Name+" "+t)
    }
  }
  if len(parts) == 1 && !named {
    return parts[0]
  }
  return "(" + strings.Join(parts, ", ") + ")"
}

func extractGo(path string, src []byte) model.FileFacts {
  facts := model.FileFacts{Path: path, Language: "go", ParseStatus: "ok"}
  fset := token.NewFileSet()
  f, err := parser.ParseFile(fset, path, src, parser.AllErrors|parser.SkipObjectResolution)
  if err != nil {
    msg := err.Error()
    if len(msg) > 300 {
      msg = msg[:300]
    }
    facts.ParseErrors = append(facts.ParseErrors, msg)
    if f == nil {
      facts.ParseStatus = "error"
      return facts
    }
    facts.ParseStatus = "partial"
  }
  for _, im := range f.Imports {
    if p, uerr := strconv.Unquote(im.Path.Value); uerr == nil {
      facts.Imports = append(facts.Imports, p)
      if im.Name != nil && im.Name.Name != "_" && im.Name.Name != "." {
        facts.Refs = append(facts.Refs, model.RawRef{Kind: "go_import", Name: im.Name.Name, Target: p, Line: fset.Position(im.Pos()).Line})
      }
    }
  }
  imps := importNames(f)
  for _, decl := range f.Decls {
    switch d := decl.(type) {
    case *ast.FuncDecl:
      sy := model.Symbol{
        Kind: "function", Name: d.Name.Name,
        StartLine: fset.Position(d.Pos()).Line, EndLine: fset.Position(d.End()).Line,
        Params: goParams(fset, d.Type.Params), Returns: goResults(fset, d.Type.Results), ReturnsSource: "declared",
        Visibility: goVis(d.Name.Name), Provenance: "verified",
        BodyHash: model.HashBytes(srcSlice(src, fset, d.Pos(), d.End())),
      }
      if d.Recv != nil && len(d.Recv.List) > 0 {
        sy.Kind = "method"
        r := strings.TrimPrefix(typeStr(fset, d.Recv.List[0].Type), "*")
        if i := strings.Index(r, "["); i >= 0 {
          r = r[:i]
        }
        sy.Parent = r
      }
      q := sy.Qualified()
      if d.Body != nil {
        locals := collectLocals(fset, d)
        facts.Refs = append(facts.Refs, goAssignRefs(fset, d, q)...)
        ast.Inspect(d.Body, func(n ast.Node) bool {
          ce, ok := n.(*ast.CallExpr)
          if !ok {
            return true
          }
          name := ""
          switch fn := ce.Fun.(type) {
          case *ast.Ident:
            name = fn.Name
          case *ast.SelectorExpr:
            name = selName(fn)
          }
          if name != "" {
            facts.Refs = append(facts.Refs, model.RawRef{Kind: "call", Name: name, Target: callTarget(name, locals, imps), Line: fset.Position(ce.Pos()).Line, From: q})
          }
          return true
        })
      }
      facts.Symbols = append(facts.Symbols, sy)
    case *ast.GenDecl:
      for _, spec := range d.Specs {
        switch sp := spec.(type) {
        case *ast.TypeSpec:
          kind := "type"
          switch sp.Type.(type) {
          case *ast.StructType:
            kind = "struct"
          case *ast.InterfaceType:
            kind = "interface"
          }
          facts.Symbols = append(facts.Symbols, model.Symbol{
            Kind: kind, Name: sp.Name.Name,
            StartLine: fset.Position(sp.Pos()).Line, EndLine: fset.Position(sp.End()).Line,
            Visibility: goVis(sp.Name.Name), Provenance: "verified",
            BodyHash: model.HashBytes(srcSlice(src, fset, sp.Pos(), sp.End())),
          })
        case *ast.ValueSpec:
          kind := "var"
          if d.Tok == token.CONST {
            kind = "const"
          }
          for _, n := range sp.Names {
            if n.Name == "_" {
              continue
            }
            facts.Symbols = append(facts.Symbols, model.Symbol{
              Kind: kind, Name: n.Name,
              StartLine: fset.Position(sp.Pos()).Line, EndLine: fset.Position(sp.End()).Line,
              Visibility: goVis(n.Name), Provenance: "verified",
              BodyHash: model.HashBytes(srcSlice(src, fset, sp.Pos(), sp.End())),
            })
          }
        }
      }
    }
  }
  return facts
}
