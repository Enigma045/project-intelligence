package resolve

import (
  "path"
  "sort"
  "strings"

  "pidx/internal/model"
)

// Input is everything the resolver needs. Resolve is a pure function: Input -> edges.
type Input struct {
  Files   []model.FileRow
  Symbols []model.Symbol
  Refs    []model.RawRef
}

type importInfo struct {
  files    []string
  internal bool
}

type fileCtx struct {
  path, dir, lang string
  bases           map[string]*importInfo
  imported        map[string]bool
}

type resolver struct {
  byName  map[string][]*model.Symbol
  byID    map[string]*model.Symbol
  byFileQ map[string]string
  vars    map[string][]*model.Symbol
  assigns map[string]string
  ctxs    map[string]*fileCtx
}

type hit struct {
  dst, status string
  conf        float64
}

func set(s string) map[string]bool {
  m := map[string]bool{}
  for _, w := range strings.Fields(s) {
    m[w] = true
  }
  return m
}

var builtins = set("make len cap append copy delete panic recover new print println close string int int64 int32 uint float64 bool byte rune error min max clear require parseInt parseFloat setTimeout setInterval clearTimeout clearInterval fetch alert confirm String Number Boolean Array Object Error Promise Date Symbol Map Set isNaN isFinite encodeURIComponent decodeURIComponent atob btoa range str float list dict tuple isinstance super open enumerate zip filter sorted sum any all type getattr setattr hasattr repr abs round")

var globals = set("console Math JSON Object Array Promise document window localStorage sessionStorage process Number String Date Error navigator location Reflect Intl Buffer history performance URL Boolean RegExp Symbol Map Set")

var plainConf = map[string]float64{"file": 0.95, "dir": 0.9, "imported": 0.85, "global": 0.4, "qualified": 0.95, "self": 0.9}
var dottedConf = map[string]float64{"file": 0.7, "dir": 0.65, "imported": 0.65, "global": 0.35}

var jsExts = []string{".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs"}
var knownExt = []string{".js", ".jsx", ".ts", ".tsx", ".mjs", ".cjs", ".py"}

func firstExisting(fileSet map[string]bool, cands []string) string {
  for _, c := range cands {
    if fileSet[c] {
      return c
    }
  }
  return ""
}

// resolveImport returns the project files an import points to, and whether the import is project-internal.
func resolveImport(f model.FileRow, imp string, fileSet map[string]bool, dirFiles map[string][]string, goDirs []string) ([]string, bool) {
  switch f.Language {
  case "go":
    for _, d := range goDirs {
      if imp == d || strings.HasSuffix(imp, "/"+d) {
        return dirFiles[d], true
      }
    }
    return nil, false
  case "javascript", "typescript":
    if !strings.HasPrefix(imp, ".") {
      return nil, false
    }
    base := path.Join(path.Dir(f.Path), imp)
    bases := []string{base}
    for _, e := range []string{".js", ".jsx", ".mjs", ".cjs"} {
      if strings.HasSuffix(base, e) {
        bases = append(bases, strings.TrimSuffix(base, e))
      }
    }
    var cands []string
    for _, b := range bases {
      cands = append(cands, b)
      for _, e := range jsExts {
        cands = append(cands, b+e, b+"/index"+e)
      }
    }
    if found := firstExisting(fileSet, cands); found != "" {
      return []string{found}, true
    }
    return nil, true
  case "python":
    n := 0
    for n < len(imp) && imp[n] == '.' {
      n++
    }
    rest := strings.ReplaceAll(imp[n:], ".", "/")
    var dirs []string
    if n > 0 {
      d := path.Dir(f.Path)
      for i := 1; i < n; i++ {
        d = path.Dir(d)
      }
      dirs = []string{d}
    } else {
      dirs = []string{".", path.Dir(f.Path)}
    }
    var cands []string
    for _, d := range dirs {
      if rest == "" {
        cands = append(cands, path.Join(d, "__init__.py"))
        continue
      }
      b := path.Join(d, rest)
      cands = append(cands, b+".py", b+"/__init__.py")
    }
    if found := firstExisting(fileSet, cands); found != "" {
      return []string{found}, true
    }
    return nil, n > 0
  }
  return nil, false
}

// importBases returns the identifiers a call qualifier may use to refer to this import.
func importBases(imp string) []string {
  t := strings.Trim(imp, "./")
  for _, e := range knownExt {
    t = strings.TrimSuffix(t, e)
  }
  if t == "" {
    return nil
  }
  parts := strings.Split(t, "/")
  last := parts[len(parts)-1]
  if len(parts) == 1 && strings.Contains(last, ".") {
    dp := strings.Split(last, ".")
    return []string{dp[0], dp[len(dp)-1]}
  }
  return []string{last}
}

func order(lang string, dotted bool) []string {
  switch {
  case lang == "go" && dotted:
    return []string{"file", "dir", "imported", "global"}
  case lang == "go":
    return []string{"file", "dir", "global"}
  }
  return []string{"file", "imported", "global"}
}

// pick returns candidates from the first non-empty scope tier.
func pick(cands []*model.Symbol, c *fileCtx, tiers []string) ([]*model.Symbol, string) {
  for _, t := range tiers {
    var out []*model.Symbol
    for _, s := range cands {
      ok := false
      switch t {
      case "file":
        ok = s.File == c.path
      case "dir":
        ok = s.File != c.path && path.Dir(s.File) == c.dir
      case "imported":
        ok = c.imported[s.File]
      case "global":
        ok = true
      }
      if ok {
        out = append(out, s)
      }
    }
    if len(out) > 0 {
      return out, t
    }
  }
  return nil, ""
}

func hitsFrom(c []*model.Symbol, tier string, dotted bool) []hit {
  if len(c) > 5 {
    c = c[:5]
  }
  base := plainConf[tier]
  if dotted {
    base = dottedConf[tier]
  }
  if len(c) == 1 {
    st := "resolved"
    if dotted || tier == "global" {
      st = "inferred"
    }
    return []hit{{c[0].ID, st, base}}
  }
  conf := base
  if f := 1 / float64(len(c)); f < conf {
    conf = f
  }
  out := make([]hit, 0, len(c))
  for _, s := range c {
    out = append(out, hit{s.ID, "ambiguous", conf})
  }
  return out
}

func ext(conf float64) []hit { return []hit{{"", "external", conf}} }
func unres() []hit          { return []hit{{"", "unresolved", 0}} }

func (r *resolver) filter(name string, kinds ...string) []*model.Symbol {
  var out []*model.Symbol
  for _, s := range r.byName[name] {
    for _, k := range kinds {
      if s.Kind == k {
        out = append(out, s)
        break
      }
    }
  }
  return out
}

func (r *resolver) target(c *fileCtx, from *model.Symbol, name, hint string) []hit {
  if strings.HasPrefix(hint, "local") {
    return r.viaLocalInferred(c, from, name, strings.TrimPrefix(strings.TrimPrefix(hint, "local"), ":"))
  }
  return r.sameLang(c, r.targetStatic(c, from, name))
}

func (r *resolver) targetStatic(c *fileCtx, from *model.Symbol, name string) []hit {
  segs := strings.Split(name, ".")
  last := segs[len(segs)-1]
  if last == "" {
    return unres()
  }
  if len(segs) == 1 {
    if cs, tier := pick(r.filter(last, "function", "class"), c, order(c.lang, false)); len(cs) > 0 {
      return hitsFrom(cs, tier, false)
    }
    if r.isVar(c, last) {
      return []hit{{"", "value", 0.85}}
    }
    if builtins[last] {
      return ext(0.9)
    }
    return unres()
  }
  first := segs[0]
  if first == "?" {
    return r.noCandidate(c)
  }
  if info, ok := c.bases[first]; ok {
    if !info.internal {
      return ext(0.95)
    }
    if len(segs) > 2 || r.isPkgVar(info, last) {
      return []hit{{"", "value", 0.8}}
    }
    in := map[string]bool{}
    for _, f := range info.files {
      in[f] = true
    }
    var cs []*model.Symbol
    for _, s := range r.filter(last, "function", "method", "class") {
      if in[s.File] {
        cs = append(cs, s)
      }
    }
    if len(cs) == 0 {
      return unres()
    }
    return hitsFrom(cs, "qualified", false)
  }
  if (first == "this" || first == "self") && len(segs) == 2 && from != nil && from.Parent != "" {
    var cs []*model.Symbol
    for _, s := range r.filter(last, "method") {
      if s.File == c.path && s.Parent == from.Parent {
        cs = append(cs, s)
      }
    }
    if len(cs) > 0 {
      return hitsFrom(cs, "self", false)
    }
  }
  if r.isVar(c, first) {
    return []hit{{"", "value", 0.85}}
  }
  if globals[first] {
    return ext(0.9)
  }
  cs, tier := pick(r.filter(last, dottedKinds(c.lang)...), c, order(c.lang, true))
  if len(cs) == 0 {
    return r.noCandidate(c)
  }
  return hitsFrom(cs, tier, true)
}

func importConf(lang string) float64 {
  if lang == "go" {
    return 1.0
  }
  return 0.9
}

// Resolve links raw references to symbols and files. Deterministic: same input, same output.
func Resolve(in Input) []model.Edge {
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

  r := &resolver{byName: map[string][]*model.Symbol{}, byID: map[string]*model.Symbol{}, byFileQ: map[string]string{}, vars: map[string][]*model.Symbol{}, assigns: buildAssigns(in.Refs)}
  for i := range in.Symbols {
    s := &in.Symbols[i]
    r.byID[s.ID] = s
    if s.Kind == "var" || s.Kind == "const" {
      r.vars[s.Name] = append(r.vars[s.Name], s)
    }
    switch s.Kind {
    case "function", "method":
      k := s.File + "|" + s.Qualified()
      if _, ok := r.byFileQ[k]; !ok {
        r.byFileQ[k] = s.ID
      }
      r.byName[s.Name] = append(r.byName[s.Name], s)
    case "class":
      r.byName[s.Name] = append(r.byName[s.Name], s)
    }
  }
  for _, l := range r.byName {
    l := l
    sort.Slice(l, func(i, j int) bool {
      if l[i].File != l[j].File {
        return l[i].File < l[j].File
      }
      return l[i].StartLine < l[j].StartLine
    })
  }

  var out []model.Edge
  idx := map[string]int{}
  add := func(e model.Edge) {
    k := e.Src + "|" + e.Dst + "|" + e.DstName + "|" + e.Type + "|" + e.Status + "|" + e.Evidence
    if i, ok := idx[k]; ok {
      out[i].Count++
      if e.Line < out[i].Line {
        out[i].Line = e.Line
      }
      return
    }
    e.Count = 1
    idx[k] = len(out)
    out = append(out, e)
  }

  ctxs := map[string]*fileCtx{}
  r.ctxs = ctxs
  gn := newGoNames(in.Refs)
  for _, f := range in.Files {
    c := &fileCtx{path: f.Path, dir: path.Dir(f.Path), lang: f.Language, bases: map[string]*importInfo{}, imported: map[string]bool{}}
    ctxs[f.Path] = c
    for _, imp := range f.Imports {
      files, internal := resolveImport(f, imp, fileSet, dirFiles, goDirs)
      for _, b := range append(importBases(imp), gn.extra(f.Path, imp, files)...) {
        info := c.bases[b]
        if info == nil {
          info = &importInfo{}
          c.bases[b] = info
        }
        info.files = append(info.files, files...)
        info.internal = info.internal || internal
      }
      for _, t := range files {
        c.imported[t] = true
        add(model.Edge{Src: "file:" + f.Path, Dst: "file:" + t, DstName: imp, Type: "imports", Status: "resolved", Confidence: importConf(f.Language), File: f.Path})
      }
      if len(files) == 0 && internal {
        add(model.Edge{Src: "file:" + f.Path, DstName: imp, Type: "imports", Status: "unresolved", File: f.Path})
      }
    }
  }

  for _, ref := range in.Refs {
    if ref.Kind != "call" && !(ref.Kind == "event" && ref.Target != "") {
      continue
    }
    c := ctxs[ref.File]
    if c == nil {
      continue
    }
    var from *model.Symbol
    src := "file:" + ref.File
    if ref.From != "" {
      if id, ok := r.byFileQ[ref.File+"|"+ref.From]; ok {
        from = r.byID[id]
        src = id
      }
    }
    name, typ, ev := ref.Name, "calls", ""
    if ref.Kind == "event" {
      name, typ, ev = ref.Target, "listens", "event:"+ref.Name
    }
    hint := ""
    if ref.Kind == "call" {
      hint = ref.Target
    }
    for _, h := range r.target(c, from, name, hint) {
      add(model.Edge{Src: src, Dst: h.dst, DstName: name, Type: typ, Status: h.status, Confidence: h.conf, Line: ref.Line, File: ref.File, Evidence: ev})
    }
  }

  sort.SliceStable(out, func(i, j int) bool {
    a, b := out[i], out[j]
    if a.Src != b.Src {
      return a.Src < b.Src
    }
    if a.Type != b.Type {
      return a.Type < b.Type
    }
    if a.Line != b.Line {
      return a.Line < b.Line
    }
    return a.Dst < b.Dst
  })
  return out
}
