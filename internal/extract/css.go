package extract

import (
  "regexp"
  "strings"

  "pidx/internal/model"
)

var (
  reCSSVarDef = regexp.MustCompile(`(--[\w-]+)\s*:`)
  reCSSVarUse = regexp.MustCompile(`var\(\s*(--[\w-]+)`)
  reCSSImport = regexp.MustCompile(`@import\s+(?:url\(\s*)?['"]?([^'")\s;]+)`)
  reSelAttr   = regexp.MustCompile(`\[[^\]]*\]`)
  reSelID     = regexp.MustCompile(`#([A-Za-z_][\w-]*)`)
  reSelClass  = regexp.MustCompile(`\.([A-Za-z_][\w-]*)`)
)

// blankComments replaces /* ... */ with spaces, keeping newlines so offsets and line numbers stay valid.
func blankComments(s string) string {
  b := []byte(s)
  for i := 0; i+1 < len(b); i++ {
    if b[i] == '/' && b[i+1] == '*' {
      j := i + 2
      for j+1 < len(b) && !(b[j] == '*' && b[j+1] == '/') {
        j++
      }
      end := j + 2
      if end > len(b) {
        end = len(b)
      }
      for k := i; k < end; k++ {
        if b[k] != '\n' {
          b[k] = ' '
        }
      }
      i = end - 1
    }
  }
  return string(b)
}

type cssWalker struct {
  s      string
  starts []int
  f      *model.FileFacts
  ctx    string
}

func (w *cssWalker) matchBrace(open, end int) int {
  depth := 0
  var q byte
  for i := open; i < end; i++ {
    c := w.s[i]
    if q != 0 {
      if c == '\\' {
        i++
        continue
      }
      if c == q {
        q = 0
      }
      continue
    }
    switch c {
    case '"', '\'':
      q = c
    case '{':
      depth++
    case '}':
      depth--
      if depth == 0 {
        return i
      }
    }
  }
  return end
}

func (w *cssWalker) walk(start, end int) {
  i := start
  for i < end {
    for i < end && isWS(w.s[i]) {
      i++
    }
    if i >= end {
      return
    }
    p := i
    paren := 0
    var q byte
    stop := false
    for i < end && !stop {
      c := w.s[i]
      switch {
      case q != 0:
        if c == '\\' {
          i++
        } else if c == q {
          q = 0
        }
      case c == '"' || c == '\'':
        q = c
      case c == '(':
        paren++
      case c == ')':
        if paren > 0 {
          paren--
        }
      case paren == 0 && (c == '{' || c == ';' || c == '}'):
        stop = true
      }
      if !stop {
        i++
      }
    }
    if i >= end {
      return
    }
    prelude := strings.TrimSpace(w.s[p:i])
    switch w.s[i] {
    case '}':
      i++
    case ';':
      w.statement(prelude, p)
      i++
    default:
      closeIdx := w.matchBrace(i, end)
      w.block(prelude, p, i, closeIdx)
      i = closeIdx + 1
    }
  }
}

func (w *cssWalker) statement(prelude string, off int) {
  if !strings.HasPrefix(prelude, "@import") {
    return
  }
  if m := reCSSImport.FindStringSubmatch(prelude); m != nil {
    w.f.Imports = append(w.f.Imports, m[1])
    w.f.Refs = append(w.f.Refs, model.RawRef{Kind: "css_import", Name: m[1], Line: lineAt(w.starts, off)})
  }
}

func (w *cssWalker) block(prelude string, pstart, open, closeIdx int) {
  line := lineAt(w.starts, pstart)
  endOff := closeIdx
  if endOff >= len(w.s) {
    endOff = len(w.s) - 1
  }
  endLine := lineAt(w.starts, endOff)
  endSlice := closeIdx + 1
  if endSlice > len(w.s) {
    endSlice = len(w.s)
  }

  if strings.HasPrefix(prelude, "@") {
    name := strings.ToLower(prelude[1:])
    if k := strings.IndexAny(name, " \t\n\r("); k >= 0 {
      name = name[:k]
    }
    switch {
    case name == "media" || name == "supports" || name == "layer" || name == "container" || name == "scope" || name == "document":
      old := w.ctx
      w.ctx = strings.Join(strings.Fields(prelude), " ")
      w.walk(open+1, closeIdx)
      w.ctx = old
    case strings.HasSuffix(name, "keyframes"):
      fs := strings.Fields(prelude)
      if len(fs) > 1 {
        w.f.Symbols = append(w.f.Symbols, model.Symbol{
          Kind: "keyframes", Name: fs[1], Parent: w.ctx, StartLine: line, EndLine: endLine,
          Visibility: "public", BodyHash: model.HashString(w.s[pstart:endSlice]), Provenance: "verified",
        })
      }
    }
    return
  }

  sel := strings.Join(strings.Fields(prelude), " ")
  if sel == "" {
    return
  }
  if len(sel) > 200 {
    sel = sel[:200]
  }
  w.f.Symbols = append(w.f.Symbols, model.Symbol{
    Kind: "css_rule", Name: sel, Parent: w.ctx, StartLine: line, EndLine: endLine,
    Visibility: "public", BodyHash: model.HashString(w.s[pstart:endSlice]), Provenance: "verified",
  })
  plain := reSelAttr.ReplaceAllString(sel, "")
  seen := map[string]bool{}
  for _, m := range reSelID.FindAllStringSubmatch(plain, -1) {
    if !seen["#"+m[1]] {
      seen["#"+m[1]] = true
      w.f.Refs = append(w.f.Refs, model.RawRef{Kind: "css_id", Name: m[1], Line: line, From: sel})
    }
  }
  for _, m := range reSelClass.FindAllStringSubmatch(plain, -1) {
    if !seen["."+m[1]] {
      seen["."+m[1]] = true
      w.f.Refs = append(w.f.Refs, model.RawRef{Kind: "css_class", Name: m[1], Line: line, From: sel})
    }
  }
  body := w.s[open+1 : closeIdx]
  if strings.Contains(body, "--") {
    for _, m := range reCSSVarDef.FindAllStringSubmatchIndex(body, -1) {
      name := body[m[2]:m[3]]
      vl := lineAt(w.starts, open+1+m[2])
      w.f.Symbols = append(w.f.Symbols, model.Symbol{
        Kind: "css_var", Name: name, StartLine: vl, EndLine: vl,
        Visibility: "public", BodyHash: model.HashString(name), Provenance: "verified",
      })
    }
    for _, m := range reCSSVarUse.FindAllStringSubmatch(body, -1) {
      w.f.Refs = append(w.f.Refs, model.RawRef{Kind: "css_var_use", Name: m[1], Line: line, From: sel})
    }
  }
}

// extractCSS emits css_rule, css_var and keyframes symbols plus css_id, css_class, css_var_use and css_import refs.
func extractCSS(path string, src []byte) model.FileFacts {
  facts := model.FileFacts{Path: path, Language: "css", ParseStatus: "heuristic"}
  s := blankComments(strings.ReplaceAll(string(src), "\r\n", "\n"))
  w := &cssWalker{s: s, starts: lineStarts(s), f: &facts}
  w.walk(0, len(s))
  return facts
}
