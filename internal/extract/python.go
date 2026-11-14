package extract

import (
  "regexp"
  "strings"

  "pidx/internal/model"
)

var (
  rePyDef    = regexp.MustCompile(`^(\s*)(async\s+)?def\s+([A-Za-z_]\w*)\s*\(`)
  rePyClass  = regexp.MustCompile(`^(\s*)class\s+([A-Za-z_]\w*)`)
  rePyConst  = regexp.MustCompile(`^([A-Z][A-Z0-9_]*)\s*(?::[^=]+)?=`)
  rePyImport = regexp.MustCompile(`^\s*import\s+(.+)`)
  rePyFrom   = regexp.MustCompile(`^\s*from\s+(\.*[\w.]*)\s+import\s`)
  rePyCall   = regexp.MustCompile(`([A-Za-z_]\w*(?:\.[A-Za-z_]\w*)*)\s*\(`)
)

var pyKw = map[string]bool{
  "if": true, "elif": true, "else": true, "while": true, "for": true, "def": true, "class": true, "return": true,
  "and": true, "or": true, "not": true, "in": true, "is": true, "lambda": true, "with": true, "assert": true,
  "yield": true, "await": true, "raise": true, "except": true, "import": true, "from": true,
}

func indentOf(l string) int {
  n := 0
  for _, c := range l {
    if c == ' ' {
      n++
    } else if c == '\t' {
      n += 4
    } else {
      break
    }
  }
  return n
}

func pyVis(name string) string {
  if strings.HasPrefix(name, "_") && !(strings.HasPrefix(name, "__") && strings.HasSuffix(name, "__")) {
    return "private"
  }
  return "public"
}

func matchParen(s string, open int) int {
  depth := 0
  for k := open; k < len(s); k++ {
    switch s[k] {
    case '(', '[', '{':
      depth++
    case ')', ']', '}':
      depth--
      if depth == 0 {
        return k
      }
    }
  }
  return -1
}

// pySignature joins a possibly multi-line def header until brackets balance.
func pySignature(lines []string, i int) (int, string) {
  depth := 0
  var b strings.Builder
  for j := i; j < len(lines); j++ {
    l := stripStrings(lines[j])
    if k := strings.Index(l, "#"); k >= 0 {
      l = l[:k]
    }
    b.WriteString(l)
    b.WriteByte(' ')
    for _, c := range l {
      if c == '(' || c == '[' || c == '{' {
        depth++
      } else if c == ')' || c == ']' || c == '}' {
        depth--
      }
    }
    if depth <= 0 || j-i > 40 {
      return j, b.String()
    }
  }
  return len(lines) - 1, b.String()
}

func pyBlockEnd(lines []string, from, ind int) int {
  last := from
  for j := from + 1; j < len(lines); j++ {
    t := strings.TrimSpace(lines[j])
    if t == "" || strings.HasPrefix(t, "#") {
      continue
    }
    if indentOf(lines[j]) <= ind {
      break
    }
    last = j
  }
  return last
}

func extractPython(path string, src []byte) model.FileFacts {
  facts := model.FileFacts{Path: path, Language: "python", ParseStatus: "heuristic"}
  lines := strings.Split(strings.ReplaceAll(string(src), "\r\n", "\n"), "\n")
  skip := map[int]bool{}
  masked := maskTriple(lines)
  type cls struct {
    name        string
    indent, end int
  }
  var stack []cls

  for i := 0; i < len(lines); i++ {
    line := lines[i]
    t := strings.TrimSpace(line)
    if t == "" || strings.HasPrefix(t, "#") {
      continue
    }
    ind := indentOf(line)
    for len(stack) > 0 && stack[len(stack)-1].end < i {
      stack = stack[:len(stack)-1]
    }
    parent := ""
    if len(stack) > 0 && stack[len(stack)-1].indent < ind {
      parent = stack[len(stack)-1].name
    }
    if m := rePyClass.FindStringSubmatch(line); m != nil {
      end := pyBlockEnd(lines, i, ind)
      facts.Symbols = append(facts.Symbols, model.Symbol{
        Kind: "class", Name: m[2], StartLine: i + 1, EndLine: end + 1,
        Visibility: pyVis(m[2]), BodyHash: bodyHash(lines, i, end), Provenance: "inferred",
      })
      stack = append(stack, cls{m[2], ind, end})
      skip[i+1] = true
      continue
    }
    if m := rePyDef.FindStringSubmatch(line); m != nil {
      sigEnd, sig := pySignature(lines, i)
      p := strings.Index(sig, "(")
      if p < 0 {
        continue
      }
      q := matchParen(sig, p)
      if q < 0 {
        continue
      }
      rest := sig[q+1:]
      ret := ""
      if k := strings.Index(rest, "->"); k >= 0 {
        ret = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(rest[k+2:]), ":"))
      }
      end := pyBlockEnd(lines, sigEnd, ind)
      kind := "function"
      if parent != "" {
        kind = "method"
      }
      facts.Symbols = append(facts.Symbols, model.Symbol{
        Kind: kind, Name: m[3], Parent: parent, StartLine: i + 1, EndLine: end + 1,
        Params: parseParams(sig[p+1 : q]), Returns: ret, ReturnsSource: srcOf(ret),
        Visibility: pyVis(m[3]), Async: m[2] != "", BodyHash: bodyHash(lines, i, end), Provenance: "inferred",
      })
      for k := i; k <= sigEnd; k++ {
        skip[k+1] = true
      }
      i = sigEnd
      continue
    }
    if ind == 0 {
      if m := rePyConst.FindStringSubmatch(line); m != nil {
        facts.Symbols = append(facts.Symbols, model.Symbol{
          Kind: "const", Name: m[1], StartLine: i + 1, EndLine: i + 1,
          Visibility: pyVis(m[1]), BodyHash: bodyHash(lines, i, i), Provenance: "inferred",
        })
      }
    }
    if m := rePyFrom.FindStringSubmatch(line); m != nil {
      facts.Imports = appendUnique(facts.Imports, m[1])
    } else if m := rePyImport.FindStringSubmatch(line); m != nil {
      for _, part := range strings.Split(m[1], ",") {
        if f := strings.Fields(strings.TrimSpace(part)); len(f) > 0 {
          facts.Imports = appendUnique(facts.Imports, f[0])
        }
      }
    }
  }

  for i := range lines {
    ln := i + 1
    if skip[ln] {
      continue
    }
    code := stripStrings(masked[i])
    if k := strings.Index(code, "#"); k >= 0 {
      code = code[:k]
    }
    if strings.TrimSpace(code) == "" {
      continue
    }
    from := enclosing(facts.Symbols, ln)
    for _, m := range rePyCall.FindAllStringSubmatch(code, -1) {
      n := m[1]
      last := n[strings.LastIndex(n, ".")+1:]
      if pyKw[n] || pyKw[last] {
        continue
      }
      facts.Refs = append(facts.Refs, model.RawRef{Kind: "call", Name: n, Line: ln, From: from})
    }
  }
  return facts
}
