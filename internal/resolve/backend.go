package resolve

import (
  "path"
  "sort"
  "strings"

  "pidx/internal/model"
)

type beCtx struct {
  routes     []*model.Symbol
  routeByKey map[string]*model.Symbol
  tables     map[string][]*model.Symbol
  fnByKey    map[string]*model.Symbol
  fnByName   map[string][]*model.Symbol
  varByName  map[string][]*model.Symbol
  consts     map[string]string
  imports    map[string]map[string]*importInfo
  out        []model.Edge
  idx        map[string]int
}

func (b *beCtx) add(e model.Edge) {
  k := e.Src + "|" + e.Dst + "|" + e.DstName + "|" + e.Type + "|" + e.Status + "|" + e.Evidence
  if i, ok := b.idx[k]; ok {
    b.out[i].Count++
    if e.Line < b.out[i].Line {
      b.out[i].Line = e.Line
    }
    return
  }
  e.Count = 1
  b.idx[k] = len(b.out)
  b.out = append(b.out, e)
}

func (b *beCtx) srcFor(file, from string) string {
  if from != "" {
    if s := b.fnByKey[file+"|"+from]; s != nil {
      return s.ID
    }
  }
  return "file:" + file
}

func splitRoute(name string) (string, string) {
  if i := strings.IndexByte(name, ' '); i > 0 {
    return name[:i], name[i+1:]
  }
  return "ANY", name
}

func isParamSeg(s string) bool {
  return strings.HasPrefix(s, "{") || strings.HasPrefix(s, ":") || s == "*"
}

// routeMatch matches a request path against a route pattern: {id}, :id and * match one segment,
// a ${expr} segment in the request matches any pattern segment, and a trailing slash means a subtree.
func routeMatch(pat, p string) bool {
  if pat == p {
    return true
  }
  if pat == "/" || p == "/" {
    return false
  }
  subtree := strings.HasSuffix(pat, "/")
  ps := strings.Split(strings.Trim(pat, "/"), "/")
  qs := strings.Split(strings.Trim(p, "/"), "/")
  if subtree {
    if len(qs) < len(ps) {
      return false
    }
  } else if len(ps) != len(qs) {
    return false
  }
  for i := range ps {
    if isParamSeg(ps[i]) || strings.HasPrefix(qs[i], "${") {
      continue
    }
    if ps[i] != qs[i] {
      return false
    }
  }
  return true
}

// pickNear prefers candidates in the same file, then the same directory, then anywhere.
func pickNear(cands []*model.Symbol, file string) []*model.Symbol {
  var same, dir []*model.Symbol
  for _, s := range cands {
    if s.File == file {
      same = append(same, s)
    } else if path.Dir(s.File) == path.Dir(file) {
      dir = append(dir, s)
    }
  }
  if len(same) > 0 {
    return same
  }
  if len(dir) > 0 {
    return dir
  }
  return cands
}

func (b *beCtx) emit(src string, ts []*model.Symbol, typ, name string, r model.RawRef, ev, single string, conf float64) {
  if len(ts) > 5 {
    ts = ts[:5]
  }
  switch len(ts) {
  case 0:
    b.add(model.Edge{Src: src, DstName: name, Type: typ, Status: "unresolved", Line: r.Line, File: r.File, Evidence: ev})
  case 1:
    b.add(model.Edge{Src: src, Dst: ts[0].ID, DstName: name, Type: typ, Status: single, Confidence: conf, Line: r.Line, File: r.File, Evidence: ev})
  default:
    c := 1 / float64(len(ts))
    for _, t := range ts {
      b.add(model.Edge{Src: src, Dst: t.ID, DstName: name, Type: typ, Status: "ambiguous", Confidence: c, Line: r.Line, File: r.File, Evidence: ev})
    }
  }
}

// buildImports maps each Go file to the packages it imports (qualifier -> project files).
func buildImports(in Input) map[string]map[string]*importInfo {
  fileSet := map[string]bool{}
  dirFiles := map[string][]string{}
  for _, f := range in.Files {
    fileSet[f.Path] = true
    if f.Language == "go" && !strings.HasSuffix(f.Path, "_test.go") {
      d := path.Dir(f.Path)
      dirFiles[d] = append(dirFiles[d], f.Path)
    }
  }
  var goDirs []string
  for d := range dirFiles {
    sort.Strings(dirFiles[d])
    if d != "." {
      goDirs = append(goDirs, d)
    }
  }
  sort.Slice(goDirs, func(i, j int) bool {
    if len(goDirs[i]) != len(goDirs[j]) {
      return len(goDirs[i]) > len(goDirs[j])
    }
    return goDirs[i] < goDirs[j]
  })
  out := map[string]map[string]*importInfo{}
  gn := newGoNames(in.Refs)
  for _, f := range in.Files {
    if f.Language != "go" || len(f.Imports) == 0 {
      continue
    }
    m := map[string]*importInfo{}
    for _, imp := range f.Imports {
      files, internal := resolveImport(f, imp, fileSet, dirFiles, goDirs)
      for _, base := range append(importBases(imp), gn.extra(f.Path, imp, files)...) {
        info := m[base]
        if info == nil {
          info = &importInfo{}
          m[base] = info
        }
        info.files = append(info.files, files...)
        info.internal = info.internal || internal
      }
    }
    out[f.Path] = m
  }
  return out
}

func (b *beCtx) pkgHandler(rt *model.Symbol, r model.RawRef, info *importInfo, last string) {
  if !info.internal {
    b.add(model.Edge{Src: rt.ID, DstName: r.Target, Type: "handles_route", Status: "external", Confidence: 0.9, Line: r.Line, File: r.File})
    return
  }
  in := map[string]bool{}
  for _, f := range info.files {
    in[f] = true
  }
  var fns, vars []*model.Symbol
  for _, s := range b.fnByName[last] {
    if s.Kind == "function" && in[s.File] {
      fns = append(fns, s)
    }
  }
  for _, s := range b.varByName[last] {
    if in[s.File] {
      vars = append(vars, s)
    }
  }
  switch {
  case len(fns) > 0:
    b.emit(rt.ID, fns, "handles_route", r.Target, r, "", "resolved", 0.95)
  case len(vars) > 0:
    b.emit(rt.ID, vars, "handles_route", r.Target, r, "", "inferred", 0.6)
  default:
    b.add(model.Edge{Src: rt.ID, DstName: r.Target, Type: "handles_route", Status: "unresolved", Line: r.Line, File: r.File})
  }
}

func (b *beCtx) handler(r model.RawRef) {
  rt := b.routeByKey[r.File+"|"+r.Name]
  if rt == nil || r.Target == "" {
    return
  }
  segs := strings.Split(r.Target, ".")
  last := segs[len(segs)-1]
  if len(segs) == 2 {
    if info := b.imports[r.File][segs[0]]; info != nil {
      b.pkgHandler(rt, r, info, last)
      return
    }
  }
  dotted := len(segs) > 1
  dir := path.Dir(r.File)
  var cands []*model.Symbol
  for _, s := range b.fnByName[last] {
    if path.Dir(s.File) != dir {
      continue
    }
    if dotted && s.Kind == "method" {
      cands = append(cands, s)
    } else if !dotted && s.Kind == "function" {
      cands = append(cands, s)
    }
  }
  if len(cands) == 0 && dotted {
    for _, s := range b.fnByName[last] {
      if s.Kind == "function" {
        cands = append(cands, s)
      }
    }
    if len(cands) == 1 {
      b.emit(rt.ID, cands, "handles_route", r.Target, r, "", "inferred", 0.5)
      return
    }
    cands = nil
  }
  if len(cands) == 0 {
    b.add(model.Edge{Src: rt.ID, DstName: r.Target, Type: "handles_route", Status: "dynamic", Confidence: 0.3, Line: r.Line, File: r.File})
    return
  }
  status, conf := "resolved", 0.95
  if dotted {
    status, conf = "inferred", 0.8
  }
  b.emit(rt.ID, cands, "handles_route", r.Target, r, "", status, conf)
}

func (b *beCtx) api(r model.RawRef) {
  raw := strings.TrimSpace(r.Name)
  if raw == "" {
    return
  }
  src := b.srcFor(r.File, r.From)
  ev := "http:" + r.Target
  if strings.Contains(raw, "://") || strings.HasPrefix(raw, "//") {
    b.add(model.Edge{Src: src, DstName: raw, Type: "requests_api", Status: "external", Confidence: 0.9, Line: r.Line, File: r.File, Evidence: ev})
    return
  }
  p, subbed := b.subst(r.File, raw)
  if i := strings.IndexAny(p, "?#"); i >= 0 {
    p = p[:i]
  }
  if p == "" || strings.HasPrefix(p, "${") {
    b.add(model.Edge{Src: src, DstName: raw, Type: "requests_api", Status: "dynamic", Confidence: 0.3, Line: r.Line, File: r.File, Evidence: ev})
    return
  }
  dyn := strings.Contains(p, "${")
  leading := strings.HasPrefix(p, "/")
  if !leading {
    p = "/" + p
  }
  if len(p) > 1 {
    p = strings.TrimRight(p, "/")
    if p == "" {
      p = "/"
    }
  }
  var hits, cands []*model.Symbol
  for _, rt := range b.routes {
    m, pat := splitRoute(rt.Name)
    if !routeMatch(pat, p) {
      continue
    }
    hits = append(hits, rt)
    if r.Target == "" || m == "ANY" || m == r.Target {
      cands = append(cands, rt)
    }
  }
  if len(cands) == 0 {
    if len(hits) > 0 {
      b.add(model.Edge{Src: src, DstName: raw, Type: "requests_api", Status: "unresolved", Line: r.Line, File: r.File, Evidence: "method mismatch " + ev})
    } else if dyn {
      b.add(model.Edge{Src: src, DstName: raw, Type: "requests_api", Status: "dynamic", Confidence: 0.3, Line: r.Line, File: r.File, Evidence: ev})
    } else if leading {
      b.add(model.Edge{Src: src, DstName: raw, Type: "requests_api", Status: "unresolved", Line: r.Line, File: r.File, Evidence: "no route " + ev})
    }
    return
  }
  want := r.Target
  if want == "" {
    want = "GET"
  }
  var exact []*model.Symbol
  for _, rt := range cands {
    if m, _ := splitRoute(rt.Name); m == want {
      exact = append(exact, rt)
    }
  }
  if len(exact) > 0 {
    cands = exact
  }
  status, conf := "inferred", 0.75
  if len(cands) == 1 {
    m, pat := splitRoute(cands[0].Name)
    if len(exact) == 1 {
      status, conf = "resolved", 0.95
    } else if m == "ANY" && pat == p {
      status, conf = "resolved", 0.9
    }
  }
  if (dyn || subbed) && status == "resolved" {
    status, conf = "inferred", 0.6
  }
  b.emit(src, cands, "requests_api", raw, r, ev, status, conf)
}

// use links a SQL statement to its table. A table the indexed project never declares (schema lives
// in a migration tool or the database itself) is external, not unresolved.
func (b *beCtx) use(r model.RawRef) {
  ts := pickNear(b.tables[strings.ToLower(r.Name)], r.File)
  if len(ts) == 0 {
    b.add(model.Edge{Src: b.srcFor(r.File, r.From), DstName: r.Name, Type: "queries_table", Status: "external", Confidence: 0.5, Line: r.Line, File: r.File, Evidence: "sql:" + r.Target})
    return
  }
  b.emit(b.srcFor(r.File, r.From), ts, "queries_table", r.Name, r, "sql:"+r.Target, "resolved", 0.9)
}

func (b *beCtx) fk(r model.RawRef) {
  src := ""
  for _, t := range b.tables[strings.ToLower(r.From)] {
    if t.File == r.File {
      src = t.ID
      break
    }
  }
  if src == "" {
    return
  }
  ts := pickNear(b.tables[strings.ToLower(r.Name)], r.File)
  if len(ts) == 0 {
    b.add(model.Edge{Src: src, DstName: r.Name, Type: "foreign_key", Status: "external", Confidence: 0.5, Line: r.Line, File: r.File, Evidence: "col:" + r.Target})
    return
  }
  b.emit(src, ts, "foreign_key", r.Name, r, "col:"+r.Target, "resolved", 0.95)
}

// ResolveBackend links web clients to HTTP routes, routes to handlers, handlers to tables and tables to
// tables: requests_api, handles_route, queries_table, foreign_key. Pure function: Input -> edges.
func ResolveBackend(in Input) []model.Edge {
  b := &beCtx{
    routeByKey: map[string]*model.Symbol{}, tables: map[string][]*model.Symbol{},
    fnByKey: map[string]*model.Symbol{}, fnByName: map[string][]*model.Symbol{},
    varByName: map[string][]*model.Symbol{}, idx: map[string]int{}, consts: map[string]string{},
  }
  for i := range in.Symbols {
    s := &in.Symbols[i]
    switch s.Kind {
    case "route":
      b.routes = append(b.routes, s)
      if _, ok := b.routeByKey[s.File+"|"+s.Name]; !ok {
        b.routeByKey[s.File+"|"+s.Name] = s
      }
    case "table":
      k := strings.ToLower(s.Name)
      b.tables[k] = append(b.tables[k], s)
    case "function", "method":
      if _, ok := b.fnByKey[s.File+"|"+s.Qualified()]; !ok {
        b.fnByKey[s.File+"|"+s.Qualified()] = s
      }
      b.fnByName[s.Name] = append(b.fnByName[s.Name], s)
    case "var", "const":
      b.varByName[s.Name] = append(b.varByName[s.Name], s)
    }
  }
  hasRoutes, hasTables := len(b.routes) > 0, len(b.tables) > 0
  if !hasRoutes && !hasTables {
    return nil
  }
  if hasRoutes {
    b.imports = buildImports(in)
  }
  for _, r := range in.Refs {
    switch r.Kind {
    case "js_const":
      b.consts[r.File+"|"+r.Name] = r.Target
    case "route_handler":
      b.handler(r)
    case "api_request":
      if hasRoutes {
        b.api(r)
      }
    case "sql_use":
      b.use(r)
    case "sql_fk":
      b.fk(r)
    }
  }
  sort.SliceStable(b.out, func(i, j int) bool {
    x, y := b.out[i], b.out[j]
    if x.Src != y.Src {
      return x.Src < y.Src
    }
    if x.Type != y.Type {
      return x.Type < y.Type
    }
    if x.Line != y.Line {
      return x.Line < y.Line
    }
    return x.Dst < y.Dst
  })
  return b.out
}
