package extract

import (
  "regexp"
  "strings"

  "pidx/internal/model"
)

var (
  reFunc       = regexp.MustCompile(`^\s*(export\s+)?(default\s+)?(async\s+)?function\s*\*?\s*([A-Za-z_$][\w$]*)\s*(?:<[^>]*>)?\s*\(([^)]*)\)\s*(?::\s*([^{]+?))?\s*\{`)
  reFuncExpr   = regexp.MustCompile(`^\s*(export\s+)?(?:const|let|var)\s+([A-Za-z_$][\w$]*)\s*=\s*(async\s+)?function\s*\*?\s*[A-Za-z_$]*\s*\(([^)]*)\)`)
  reArrow      = regexp.MustCompile(`^\s*(export\s+)?(?:const|let|var)\s+([A-Za-z_$][\w$]*)\s*(?::[^=]+)?=\s*(async\s+)?(?:\(([^)]*)\)|([A-Za-z_$][\w$]*))\s*(?::\s*([^=]+?))?\s*=>`)
  reClass      = regexp.MustCompile(`^\s*(export\s+)?(?:default\s+)?(?:abstract\s+)?class\s+([A-Za-z_$][\w$]*)`)
  reIface      = regexp.MustCompile(`^\s*(export\s+)?(interface|type|enum)\s+([A-Za-z_$][\w$]*)`)
  reMethod     = regexp.MustCompile(`^\s*(?:(?:public|private|protected|static|async|override|readonly)\s+)*([A-Za-z_$][\w$]*)\s*(?:<[^>]*>)?\s*\(([^)]*)\)\s*(?::\s*([^{]+?))?\s*\{`)
  reConst      = regexp.MustCompile(`^(export\s+)?const\s+([A-Z][A-Z0-9_]*)\s*(?::[^=]+)?=`)
  reImportFrom = regexp.MustCompile(`(?:import|export)\s[^'"]*?from\s*['"]([^'"]+)['"]`)
  reImportBare = regexp.MustCompile(`^\s*import\s*['"]([^'"]+)['"]`)
  reRequire    = regexp.MustCompile(`require\(\s*['"]([^'"]+)['"]\s*\)`)
  reCall       = regexp.MustCompile(`([A-Za-z_$][\w$]*(?:\.[A-Za-z_$][\w$]*)*)\s*\(`)
  reDomID      = regexp.MustCompile(`getElementById\(\s*['"]([^'"]+)['"]`)
  reQuery      = regexp.MustCompile(`querySelector(?:All)?\(\s*['"]([^'"]+)['"]`)
  reListen     = regexp.MustCompile(`addEventListener\(\s*['"]([^'"]+)['"]\s*,\s*([A-Za-z_$][\w$.]*)?`)
  reFetch      = regexp.MustCompile(`fetch\(\s*['"\x60]([^'"\x60]+)['"\x60]`)
)

var jsKw = map[string]bool{
  "if": true, "for": true, "while": true, "switch": true, "catch": true, "function": true, "return": true,
  "typeof": true, "super": true, "import": true, "else": true, "do": true, "with": true,
}

func jsVis(export string) string {
  if export != "" {
    return "exported"
  }
  return "private"
}

// blockEnd returns the 0-based line where the brace block opened at/after start closes.
// optional=true: if the first line has no brace, the construct is a one-liner.
func blockEnd(lines []string, start int, optional bool) int {
  depth := 0
  seen := false
  var q byte
  for j := start; j < len(lines); j++ {
    l := lines[j]
    for k := 0; k < len(l); k++ {
      c := l[k]
      if q != 0 {
        if c == '\\' {
          k++
          continue
        }
        if c == q {
          q = 0
        }
        continue
      }
      if c == '/' && k+1 < len(l) && l[k+1] == '/' {
        break
      }
      if c == '\'' || c == '"' || c == '`' {
        q = c
        continue
      }
      if c == '{' {
        depth++
        seen = true
      } else if c == '}' {
        depth--
      }
    }
    if q == '\'' || q == '"' {
      q = 0
    }
    if seen && depth <= 0 {
      return j
    }
    if !seen && optional && j == start && !strings.HasSuffix(strings.TrimSpace(l), "=>") {
      return start
    }
  }
  if !seen {
    return start
  }
  return len(lines) - 1
}

func remainder(code string) string {
  if i := strings.Index(code, "=>"); i >= 0 {
    return code[i+2:]
  }
  if i := strings.Index(code, "{"); i >= 0 {
    return code[i+1:]
  }
  return ""
}

func isCommentLine(t string) bool {
  return t == "" || strings.HasPrefix(t, "//") || strings.HasPrefix(t, "*") || strings.HasPrefix(t, "/*")
}

func extractJS(path string, src []byte) model.FileFacts {
  facts := model.FileFacts{Path: path, Language: "javascript", ParseStatus: "heuristic"}
  ts := strings.HasSuffix(path, ".ts") || strings.HasSuffix(path, ".tsx")
  if ts {
    facts.Language = "typescript"
  }
  lines := strings.Split(strings.ReplaceAll(string(src), "\r\n", "\n"), "\n")
  starts := map[int]bool{}
  className := ""
  classEnd, skipTo := -1, -1

  add := func(s model.Symbol, start, end int) {
    s.StartLine, s.EndLine = start+1, end+1
    s.BodyHash = bodyHash(lines, start, end)
    s.Provenance = "inferred"
    facts.Symbols = append(facts.Symbols, s)
    starts[start+1] = true
  }

  for i := 0; i < len(lines); i++ {
    line := lines[i]
    if isCommentLine(strings.TrimSpace(line)) || i <= skipTo {
      continue
    }
    if i > classEnd {
      className = ""
    }
    if m := reClass.FindStringSubmatch(line); m != nil {
      end := blockEnd(lines, i, false)
      add(model.Symbol{Kind: "class", Name: m[2], Visibility: jsVis(m[1])}, i, end)
      className, classEnd = m[2], end
      continue
    }
    if m := reFunc.FindStringSubmatch(line); m != nil {
      end := blockEnd(lines, i, false)
      ret := strings.TrimSpace(m[6])
      add(model.Symbol{Kind: "function", Name: m[4], Params: parseParams(m[5]), Returns: ret, ReturnsSource: srcOf(ret), Visibility: jsVis(m[1]), Async: m[3] != ""}, i, end)
      continue
    }
    if m := reFuncExpr.FindStringSubmatch(line); m != nil {
      end := blockEnd(lines, i, false)
      add(model.Symbol{Kind: "function", Name: m[2], Params: parseParams(m[4]), ReturnsSource: "unknown", Visibility: jsVis(m[1]), Async: m[3] != ""}, i, end)
      continue
    }
    if m := reArrow.FindStringSubmatch(line); m != nil {
      end := blockEnd(lines, i, true)
      ps := m[4]
      if ps == "" {
        ps = m[5]
      }
      ret := strings.TrimSpace(m[6])
      add(model.Symbol{Kind: "function", Name: m[2], Params: parseParams(ps), Returns: ret, ReturnsSource: srcOf(ret), Visibility: jsVis(m[1]), Async: m[3] != ""}, i, end)
      continue
    }
    if ts {
      if m := reIface.FindStringSubmatch(line); m != nil {
        end := blockEnd(lines, i, m[2] == "type")
        add(model.Symbol{Kind: m[2], Name: m[3], Visibility: jsVis(m[1])}, i, end)
        continue
      }
    }
    if className != "" {
      if m := reMethod.FindStringSubmatch(line); m != nil && !jsKw[m[1]] {
        prefix := line[:strings.Index(line, m[1])]
        vis := "public"
        if strings.Contains(prefix, "private") {
          vis = "private"
        } else if strings.Contains(prefix, "protected") {
          vis = "protected"
        }
        end := blockEnd(lines, i, false)
        ret := strings.TrimSpace(m[3])
        add(model.Symbol{Kind: "method", Name: m[1], Parent: className, Params: parseParams(m[2]), Returns: ret, ReturnsSource: srcOf(ret), Visibility: vis, Async: strings.Contains(prefix, "async")}, i, end)
        skipTo = end
        continue
      }
    }
    if m := reConst.FindStringSubmatch(line); m != nil {
      end := blockEnd(lines, i, true)
      add(model.Symbol{Kind: "const", Name: m[2], Visibility: jsVis(m[1])}, i, end)
    }
  }

  for i, line := range lines {
    t := strings.TrimSpace(line)
    if isCommentLine(t) {
      continue
    }
    for _, m := range reImportFrom.FindAllStringSubmatch(line, -1) {
      facts.Imports = appendUnique(facts.Imports, m[1])
    }
    for _, m := range reImportBare.FindAllStringSubmatch(line, -1) {
      facts.Imports = appendUnique(facts.Imports, m[1])
    }
    for _, m := range reRequire.FindAllStringSubmatch(line, -1) {
      facts.Imports = appendUnique(facts.Imports, m[1])
    }
    ln := i + 1
    from := enclosing(facts.Symbols, ln)
    addRef := func(kind, name, target string) {
      facts.Refs = append(facts.Refs, model.RawRef{Kind: kind, Name: name, Target: target, Line: ln, From: from})
    }
    for _, m := range reDomID.FindAllStringSubmatch(line, -1) {
      addRef("dom_id", m[1], "")
    }
    for _, m := range reQuery.FindAllStringSubmatch(line, -1) {
      addRef("dom_selector", m[1], "")
    }
    for _, m := range reListen.FindAllStringSubmatch(line, -1) {
      addRef("event", m[1], m[2])
    }
    for _, m := range reFetch.FindAllStringSubmatch(line, -1) {
      addRef("api_request", m[1], "")
    }
    code := stripStrings(line)
    if starts[ln] {
      code = remainder(code)
    }
    for _, m := range reCall.FindAllStringSubmatch(code, -1) {
      n := m[1]
      last := n[strings.LastIndex(n, ".")+1:]
      if jsKw[n] || jsKw[last] {
        continue
      }
      addRef("call", n, "")
    }
  }
  return facts
}
