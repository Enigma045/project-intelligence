package resolve

import (
  "path"
  "regexp"
  "sort"
  "strconv"
  "strings"

  "pidx/internal/model"
)

var (
  reHandlerCall = regexp.MustCompile(`^\s*(?:return\s+)?([A-Za-z_$][\w$]*)\s*\(`)
  reSelAttr     = regexp.MustCompile(`\[[^\]]*\]`)
  reSelID       = regexp.MustCompile(`#([A-Za-z_][\w-]*)`)
  reSelCls      = regexp.MustCompile(`\.([A-Za-z_][\w-]*)`)
)

var rank = map[string]int{"resolved": 0, "external": 0, "inferred": 1, "ambiguous": 2, "unresolved": 3, "dynamic": 3}

// combine merges the status of the element side and the handler side of a binding.
func combine(elem, h string) string {
  if h == "external" {
    return "external"
  }
  if rank[h] > rank[elem] {
    return h
  }
  return elem
}

func uniqAppend(l []string, s string) []string {
  for _, x := range l {
    if x == s {
      return l
    }
  }
  return append(l, s)
}

func lineKey(file string, line int) string { return file + "|" + strconv.Itoa(line) }

func classesIn(sel string) []string {
  var out []string
  for _, m := range reSelCls.FindAllStringSubmatch(reSelAttr.ReplaceAllString(sel, ""), -1) {
    out = append(out, m[1])
  }
  return out
}

type webCtx struct {
  files      map[string]string
  elems      map[string]map[string]string // html file -> dom id -> element symbol ID
  classes    map[string]map[string]bool   // html file -> classes present
  classEls   map[string]map[string][]string // html file -> class -> element IDs
  pagesOf    map[string][]string          // asset file -> html pages loading it
  loads      map[string][]string          // html page -> loaded asset files
  fnByName   map[string][]*model.Symbol
  fnByFileQ  map[string]*model.Symbol
  ruleByLine map[string]*model.Symbol
  cssClass   map[string][]*model.Symbol // class -> css rules mentioning it
  out        []model.Edge
  idx        map[string]int
}

func (w *webCtx) add(e model.Edge) {
  k := e.Src + "|" + e.Dst + "|" + e.DstName + "|" + e.Type + "|" + e.Status + "|" + e.Evidence
  if i, ok := w.idx[k]; ok {
    w.out[i].Count++
    if e.Line < w.out[i].Line {
      w.out[i].Line = e.Line
    }
    return
  }
  e.Count = 1
  w.idx[k] = len(w.out)
  w.out = append(w.out, e)
}

// pages returns the HTML pages a file belongs to: itself if HTML (inline code), else the pages that load it.
func (w *webCtx) pages(file string) []string {
  if w.files[file] == "html" {
    return []string{file}
  }
  return w.pagesOf[file]
}

func (w *webCtx) assetPath(from, ref string) (string, bool) {
  c := ref
  if i := strings.IndexAny(c, "?#"); i >= 0 {
    c = c[:i]
  }
  if c == "" {
    return "", false
  }
  if strings.Contains(c, "://") || strings.HasPrefix(c, "//") || strings.HasPrefix(c, "data:") {
    return "", true
  }
  var p string
  if strings.HasPrefix(c, "/") {
    p = path.Clean(strings.TrimPrefix(c, "/"))
  } else {
    p = path.Join(path.Dir(from), c)
  }
  if _, ok := w.files[p]; ok {
    return p, false
  }
  suf := "/" + strings.TrimLeft(path.Clean(c), "./")
  var found []string
  for f := range w.files {
    if strings.HasSuffix(f, suf) {
      found = append(found, f)
    }
  }
  if len(found) == 1 {
    return found[0], false
  }
  return "", false
}

func (w *webCtx) srcFor(r model.RawRef) string {
  if r.From != "" {
    if s := w.fnByFileQ[r.File+"|"+r.From]; s != nil {
      return s.ID
    }
  }
  return "file:" + r.File
}

// byID finds elements with the given DOM id: first on the pages that load/contain the file, then anywhere.
func (w *webCtx) byID(file, id string) ([]string, string, float64) {
  var ids []string
  for _, p := range w.pages(file) {
    if e, ok := w.elems[p][id]; ok {
      ids = append(ids, e)
    }
  }
  if len(ids) > 0 {
    return ids, "resolved", 0.95
  }
  var fb []string
  for _, m := range w.elems {
    if e, ok := m[id]; ok {
      fb = append(fb, e)
    }
  }
  sort.Strings(fb)
  switch {
  case len(fb) == 1:
    return fb, "inferred", 0.5
  case len(fb) > 1:
    if len(fb) > 5 {
      fb = fb[:5]
    }
    return fb, "ambiguous", 1 / float64(len(fb))
  }
  return nil, "unresolved", 0
}

// lookup resolves a selector containing an #id. ok=false means the selector has no id.
func (w *webCtx) lookup(file, sel string) ([]string, string, float64, bool) {
  m := reSelID.FindStringSubmatch(reSelAttr.ReplaceAllString(sel, ""))
  if m == nil {
    return nil, "", 0, false
  }
  ids, st, conf := w.byID(file, m[1])
  if strings.TrimSpace(sel) != "#"+m[1] && st == "resolved" {
    st, conf = "inferred", conf*0.8
  }
  return ids, st, conf, true
}

// findFn resolves a handler name: same file, then scripts on the same page(s), then anywhere.
func (w *webCtx) findFn(name, file string) []hit {
  segs := strings.Split(name, ".")
  last := segs[len(segs)-1]
  dotted := len(segs) > 1
  var cands []*model.Symbol
  for _, s := range w.fnByName[last] {
    if (s.Kind == "function" || (dotted && s.Kind == "method")) && langGroup(s.File) == "js" {
      cands = append(cands, s)
    }
  }
  if len(cands) == 0 {
    if dotted && globals[segs[0]] {
      return ext(0.9)
    }
    if !dotted && builtins[last] {
      return ext(0.9)
    }
    return []hit{{"", "unresolved", 0}}
  }
  sib := map[string]bool{file: true}
  for _, p := range w.pages(file) {
    sib[p] = true
    for _, a := range w.loads[p] {
      sib[a] = true
    }
  }
  var cs []*model.Symbol
  tier := ""
  for _, t := range []string{"file", "sib", "global"} {
    for _, s := range cands {
      if (t == "file" && s.File == file) || (t == "sib" && sib[s.File]) || t == "global" {
        cs = append(cs, s)
      }
    }
    if len(cs) > 0 {
      tier = t
      break
    }
  }
  if len(cs) > 5 {
    cs = cs[:5]
  }
  base := map[string]float64{"file": 0.95, "sib": 0.9, "global": 0.4}[tier]
  if dotted {
    base *= 0.7
  }
  if len(cs) == 1 {
    st := "resolved"
    if tier == "global" || dotted {
      st = "inferred"
    }
    return []hit{{cs[0].ID, st, base}}
  }
  conf := base
  if f := 1 / float64(len(cs)); f < conf {
    conf = f
  }
  out := make([]hit, 0, len(cs))
  for _, s := range cs {
    out = append(out, hit{s.ID, "ambiguous", conf})
  }
  return out
}

func (w *webCtx) emitListen(srcs []string, st string, conf float64, handler, ev string, r model.RawRef) {
  var hs []hit
  name := handler
  if handler == "" {
    hs = []hit{{w.srcFor(r), "inferred", 0.6}}
    name = "(anonymous)"
  } else {
    hs = w.findFn(handler, r.File)
  }
  for _, s := range srcs {
    for _, h := range hs {
      w.add(model.Edge{Src: s, Dst: h.dst, DstName: name, Type: "listens_dom", Status: combine(st, h.status),
        Confidence: conf * h.conf, Line: r.Line, File: r.File, Evidence: "event:" + ev})
    }
  }
}

// ResolveWeb links HTML, CSS and JS: loads_script, loads_style, references_dom, listens_dom, styles, toggles_class.
// Pure function: Input -> edges. Returns nil quickly when the project has no HTML/CSS.
func ResolveWeb(in Input) []model.Edge {
  w := &webCtx{
    files: map[string]string{}, elems: map[string]map[string]string{}, classes: map[string]map[string]bool{},
    pagesOf: map[string][]string{}, loads: map[string][]string{}, fnByName: map[string][]*model.Symbol{},
    fnByFileQ: map[string]*model.Symbol{}, ruleByLine: map[string]*model.Symbol{},
    cssClass: map[string][]*model.Symbol{}, idx: map[string]int{}, classEls: map[string]map[string][]string{},
  }
  hasWeb := false
  for _, f := range in.Files {
    w.files[f.Path] = f.Language
    if f.Language == "html" || f.Language == "css" {
      hasWeb = true
    }
  }
  if !hasWeb {
    return nil
  }
  for i := range in.Symbols {
    s := &in.Symbols[i]
    switch s.Kind {
    case "element", "form":
      if w.files[s.File] != "html" {
        continue
      }
      if k := strings.Index(s.Name, "#"); k >= 0 {
        m := w.elems[s.File]
        if m == nil {
          m = map[string]string{}
          w.elems[s.File] = m
        }
        if _, dup := m[s.Name[k+1:]]; !dup {
          m[s.Name[k+1:]] = s.ID
        }
      }
    case "function", "method":
      w.fnByName[s.Name] = append(w.fnByName[s.Name], s)
      if _, ok := w.fnByFileQ[s.File+"|"+s.Qualified()]; !ok {
        w.fnByFileQ[s.File+"|"+s.Qualified()] = s
      }
    case "css_rule":
      k := lineKey(s.File, s.StartLine)
      if _, ok := w.ruleByLine[k]; !ok {
        w.ruleByLine[k] = s
      }
    }
  }

  // Pass A: page structure (which page loads what, which classes exist).
  for _, r := range in.Refs {
    switch r.Kind {
    case "script_src", "style_href":
      typ := "loads_script"
      if r.Kind == "style_href" {
        typ = "loads_style"
      }
      src := "file:" + r.File
      target, external := w.assetPath(r.File, r.Name)
      switch {
      case external:
        w.add(model.Edge{Src: src, DstName: r.Name, Type: typ, Status: "external", Confidence: 0.95, Line: r.Line, File: r.File})
      case target == "":
        w.add(model.Edge{Src: src, DstName: r.Name, Type: typ, Status: "unresolved", Line: r.Line, File: r.File})
      default:
        w.add(model.Edge{Src: src, Dst: "file:" + target, DstName: r.Name, Type: typ, Status: "resolved", Confidence: 1.0, Line: r.Line, File: r.File})
        w.pagesOf[target] = uniqAppend(w.pagesOf[target], r.File)
        w.loads[r.File] = uniqAppend(w.loads[r.File], target)
      }
    case "html_class":
      if w.classes[r.File] == nil {
        w.classes[r.File] = map[string]bool{}
      }
      w.classes[r.File][r.Name] = true
      if i := strings.Index(r.From, "#"); i >= 0 {
        if id := w.elems[r.File][r.From[i+1:]]; id != "" {
          if w.classEls[r.File] == nil {
            w.classEls[r.File] = map[string][]string{}
          }
          w.classEls[r.File][r.Name] = uniqAppend(w.classEls[r.File][r.Name], id)
        }
      }
    case "css_class":
      if s := w.ruleByLine[lineKey(r.File, r.Line)]; s != nil {
        w.cssClass[r.Name] = append(w.cssClass[r.Name], s)
      }
    }
  }
  for k := range w.pagesOf {
    sort.Strings(w.pagesOf[k])
  }

  // Pass B: DOM, event and style relationships.
  for _, r := range in.Refs {
    switch r.Kind {
    case "dom_id", "dom_selector":
      sel := r.Name
      if r.Kind == "dom_id" {
        sel = "#" + r.Name
      }
      src := w.srcFor(r)
      ids, st, conf, ok := w.lookup(r.File, sel)
      if !ok {
        for _, p := range w.pages(r.File) {
          for _, c := range classesIn(sel) {
            if w.classes[p][c] {
              w.add(model.Edge{Src: src, Dst: "file:" + p, DstName: sel, Type: "references_dom", Status: "inferred",
                Confidence: 0.6, Line: r.Line, File: r.File, Evidence: "class ." + c})
            }
          }
        }
        continue
      }
      if len(ids) == 0 {
        w.add(model.Edge{Src: src, DstName: sel, Type: "references_dom", Status: "unresolved", Line: r.Line, File: r.File, Evidence: r.Kind})
        continue
      }
      for _, id := range ids {
        w.add(model.Edge{Src: src, Dst: id, DstName: sel, Type: "references_dom", Status: st, Confidence: conf, Line: r.Line, File: r.File, Evidence: r.Kind})
      }
    case "dom_listen":
      ev, handler := r.Target, ""
      if i := strings.Index(r.Target, ":"); i >= 0 {
        ev, handler = r.Target[:i], r.Target[i+1:]
      }
      ids, st, conf, ok := w.lookup(r.File, r.Name)
      srcs := ids
      if !ok || len(ids) == 0 {
        srcs = []string{"dom:" + r.Name}
        st, conf = "unresolved", 0
      }
      w.emitListen(srcs, st, conf, handler, ev, r)
    case "html_handler":
      ev := strings.TrimPrefix(r.Name, "on")
      src := "file:" + r.File
      if i := strings.Index(r.From, "#"); i >= 0 {
        if id := w.elems[r.File][r.From[i+1:]]; id != "" {
          src = id
        }
      }
      m := reHandlerCall.FindStringSubmatch(r.Target)
      if m == nil {
        t := r.Target
        if len(t) > 60 {
          t = t[:60]
        }
        w.add(model.Edge{Src: src, DstName: t, Type: "listens_dom", Status: "dynamic", Confidence: 0.3, Line: r.Line, File: r.File, Evidence: "event:" + ev})
        continue
      }
      for _, h := range w.findFn(m[1], r.File) {
        w.add(model.Edge{Src: src, Dst: h.dst, DstName: m[1], Type: "listens_dom", Status: h.status, Confidence: h.conf, Line: r.Line, File: r.File, Evidence: "event:" + ev})
      }
    case "css_id", "css_class":
      src := "file:" + r.File
      if s := w.ruleByLine[lineKey(r.File, r.Line)]; s != nil {
        src = s.ID
      }
      for _, p := range w.pages(r.File) {
        if r.Kind == "css_id" {
          if e, ok := w.elems[p][r.Name]; ok {
            w.add(model.Edge{Src: src, Dst: e, DstName: "#" + r.Name, Type: "styles", Status: "resolved", Confidence: 0.9, Line: r.Line, File: r.File, Evidence: "#" + r.Name})
          }
        } else if w.classes[p][r.Name] {
          if els := w.classEls[p][r.Name]; len(els) > 0 {
            for _, el := range els {
              w.add(model.Edge{Src: src, Dst: el, DstName: "." + r.Name, Type: "styles", Status: "inferred", Confidence: 0.7, Line: r.Line, File: r.File, Evidence: "." + r.Name})
            }
          } else {
            w.add(model.Edge{Src: src, Dst: "file:" + p, DstName: "." + r.Name, Type: "styles", Status: "inferred", Confidence: 0.7, Line: r.Line, File: r.File, Evidence: "." + r.Name})
          }
        }
      }
    case "dom_class":
      src := w.srcFor(r)
      for _, p := range w.pages(r.File) {
        assets := append([]string{p}, w.loads[p]...)
        for _, rule := range w.cssClass[r.Name] {
          for _, a := range assets {
            if rule.File == a {
              w.add(model.Edge{Src: src, Dst: rule.ID, DstName: "." + r.Name, Type: "toggles_class", Status: "inferred", Confidence: 0.6, Line: r.Line, File: r.File, Evidence: "classList." + r.Target})
            }
          }
        }
      }
    }
  }

  sort.SliceStable(w.out, func(i, j int) bool {
    a, b := w.out[i], w.out[j]
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
  return w.out
}
