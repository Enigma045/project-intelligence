package extract

import (
  "go/ast"
  "go/parser"
  "go/token"
  "regexp"
  "strconv"
  "strings"

  "pidx/internal/model"
)

var (
  reSQLHint = regexp.MustCompile(`(?i)create\s+(?:temp\w*\s+)?(?:unique\s+)?(?:table|view|index|trigger)\s`)
  reUseIns  = regexp.MustCompile(`(?is)^\s*insert\s+(?:or\s+\w+\s+)?into\s+[\x22\x60]?([A-Za-z_]\w*)`)
  reUseUpd  = regexp.MustCompile(`(?is)^\s*update\s+(?:or\s+\w+\s+)?[\x22\x60]?([A-Za-z_]\w*)[\x22\x60]?\s+set\b`)
  reUseDel  = regexp.MustCompile(`(?is)^\s*delete\s+from\s+[\x22\x60]?([A-Za-z_]\w*)`)
  reUseSel  = regexp.MustCompile(`(?is)^\s*select\b.*?\bfrom\s+[\x22\x60]?([A-Za-z_]\w*)`)
  reUseFrom = regexp.MustCompile(`^\s*(?:FROM|JOIN)\s+[\x22\x60]?([A-Za-z_]\w*)`)
  reUseJoin = regexp.MustCompile(`(?is)\bjoin\s+[\x22\x60]?([A-Za-z_]\w*)[\x22\x60]?\s+(?:on|using|as)\b`)
)

// routeNames maps registration method names to an HTTP method ("" = taken from the pattern).
var routeNames = map[string]string{
  "HandleFunc": "", "Handle": "",
  "GET": "GET", "POST": "POST", "PUT": "PUT", "DELETE": "DELETE", "PATCH": "PATCH", "HEAD": "HEAD", "OPTIONS": "OPTIONS", "Any": "ANY", "ANY": "ANY",
  "Get": "GET", "Post": "POST", "Put": "PUT", "Delete": "DELETE", "Patch": "PATCH", "Head": "HEAD", "Options": "OPTIONS",
}

// sqlUses returns (table, operation) pairs found in one string literal.
func sqlUses(v string) [][2]string {
  var out [][2]string
  if m := reUseIns.FindStringSubmatch(v); m != nil {
    out = append(out, [2]string{m[1], "insert"})
  }
  if m := reUseUpd.FindStringSubmatch(v); m != nil {
    out = append(out, [2]string{m[1], "update"})
  }
  if m := reUseDel.FindStringSubmatch(v); m != nil {
    out = append(out, [2]string{m[1], "delete"})
  }
  m := reUseSel.FindStringSubmatch(v)
  if m == nil {
    m = reUseFrom.FindStringSubmatch(v)
  }
  if m != nil {
    out = append(out, [2]string{m[1], "select"})
    for _, j := range reUseJoin.FindAllStringSubmatch(v, -1) {
      out = append(out, [2]string{j[1], "select"})
    }
  }
  return out
}

func funcQ(fset *token.FileSet, d *ast.FuncDecl) string {
  if d.Recv != nil && len(d.Recv.List) > 0 {
    r := strings.TrimPrefix(typeStr(fset, d.Recv.List[0].Type), "*")
    if i := strings.Index(r, "["); i >= 0 {
      r = r[:i]
    }
    return r + "." + d.Name.Name
  }
  return d.Name.Name
}

func routeHandlerName(e ast.Expr) string {
  switch x := e.(type) {
  case *ast.Ident:
    return x.Name
  case *ast.SelectorExpr:
    if n, ok := chain(x); ok {
      return n
    }
  case *ast.CallExpr:
    if len(x.Args) > 0 {
      return routeHandlerName(x.Args[len(x.Args)-1])
    }
  }
  return ""
}

func routeCall(fset *token.FileSet, c *ast.CallExpr, q string, facts *model.FileFacts) {
  sel, ok := c.Fun.(*ast.SelectorExpr)
  if !ok || len(c.Args) < 2 {
    return
  }
  method, ok := routeNames[sel.Sel.Name]
  if !ok {
    return
  }
  lit, ok := c.Args[0].(*ast.BasicLit)
  if !ok || lit.Kind != token.STRING {
    return
  }
  pat, err := strconv.Unquote(lit.Value)
  if err != nil {
    return
  }
  pat = strings.TrimSpace(pat)
  if method == "" {
    method = "ANY"
    if i := strings.IndexByte(pat, ' '); i > 0 {
      method = strings.ToUpper(pat[:i])
      pat = strings.TrimSpace(pat[i+1:])
    }
  }
  if !strings.HasPrefix(pat, "/") {
    return
  }
  handler := routeHandlerName(c.Args[len(c.Args)-1])
  line := fset.Position(c.Pos()).Line
  name := method + " " + pat
  facts.Symbols = append(facts.Symbols, model.Symbol{
    Kind: "route", Name: name, StartLine: line, EndLine: fset.Position(c.End()).Line, Visibility: "public",
    BodyHash: model.HashString(name + "|" + handler), Provenance: "inferred",
  })
  facts.Refs = append(facts.Refs, model.RawRef{Kind: "route_handler", Name: name, Target: handler, Line: line, From: q})
}

// goBackend finds HTTP route registrations, embedded DDL and SQL usage in a Go file.
// Test files are skipped so fixtures do not pollute the index.
func goBackend(path string, src []byte) ([]model.Symbol, []model.RawRef) {
  if strings.HasSuffix(path, "_test.go") {
    return nil, nil
  }
  fset := token.NewFileSet()
  f, _ := parser.ParseFile(fset, path, src, parser.SkipObjectResolution)
  if f == nil {
    return nil, nil
  }
  var facts model.FileFacts
  visit := func(q string) func(ast.Node) bool {
    return func(n ast.Node) bool {
      switch x := n.(type) {
      case *ast.BasicLit:
        if x.Kind != token.STRING {
          return true
        }
        v, err := strconv.Unquote(x.Value)
        if err != nil || len(v) < 8 || len(v) > 20000 {
          return true
        }
        line := fset.Position(x.Pos()).Line
        if reSQLHint.MatchString(v) {
          sqlFacts(v, line, &facts)
        }
        for _, u := range sqlUses(v) {
          facts.Refs = append(facts.Refs, model.RawRef{Kind: "sql_use", Name: u[0], Target: u[1], Line: line, From: q})
        }
      case *ast.CallExpr:
        routeCall(fset, x, q, &facts)
      }
      return true
    }
  }
  for _, decl := range f.Decls {
    q := ""
    if fd, ok := decl.(*ast.FuncDecl); ok {
      q = funcQ(fset, fd)
    }
    ast.Inspect(decl, visit(q))
  }
  return facts.Symbols, facts.Refs
}
