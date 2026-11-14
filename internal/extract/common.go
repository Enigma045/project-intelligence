package extract

import (
  "sort"
  "strconv"
  "strings"

  "pidx/internal/model"
)

// finalize assigns file, stable IDs and signature hashes.
func finalize(path string, f model.FileFacts) model.FileFacts {
  sort.SliceStable(f.Symbols, func(i, j int) bool { return f.Symbols[i].StartLine < f.Symbols[j].StartLine })
  counts := map[string]int{}
  for i := range f.Symbols {
    s := &f.Symbols[i]
    s.File = path
    base := s.Kind + ":" + path + "#" + s.Qualified()
    n := counts[base]
    counts[base] = n + 1
    if n > 0 {
      base += ":" + strconv.Itoa(n+1)
    }
    s.ID = base
    sig := []string{s.Kind, s.Qualified(), s.Visibility, s.Returns}
    for _, p := range s.Params {
      sig = append(sig, p.Name+":"+p.Type)
    }
    s.SigHash = model.HashString(strings.Join(sig, "|"))
  }
  return f
}

// enclosing returns the qualified name of the innermost function/method containing line (1-based).
func enclosing(syms []model.Symbol, line int) string {
  best, span := -1, 1<<30
  for i, s := range syms {
    if (s.Kind != "function" && s.Kind != "method") || line < s.StartLine || line > s.EndLine {
      continue
    }
    if sp := s.EndLine - s.StartLine; sp < span {
      span, best = sp, i
    }
  }
  if best < 0 {
    return ""
  }
  return syms[best].Qualified()
}

func appendUnique(list []string, s string) []string {
  for _, x := range list {
    if x == s {
      return list
    }
  }
  return append(list, s)
}

func bodyHash(lines []string, a, b int) string {
  if a < 0 || b >= len(lines) || a > b {
    return ""
  }
  return model.HashString(strings.Join(lines[a:b+1], "\n"))
}

func srcOf(t string) string {
  if t == "" {
    return "unknown"
  }
  return "declared"
}

// splitTop splits on commas at bracket depth 0.
func splitTop(s string) []string {
  var out []string
  depth, start := 0, 0
  for i := 0; i < len(s); i++ {
    switch s[i] {
    case '(', '[', '{', '<':
      depth++
    case ')', ']', '}':
      depth--
    case '>':
      if i == 0 || s[i-1] != '=' {
        depth--
      }
    case ',':
      if depth == 0 {
        out = append(out, s[start:i])
        start = i + 1
      }
    }
    if depth < 0 {
      depth = 0
    }
  }
  out = append(out, s[start:])
  var clean []string
  for _, p := range out {
    if p = strings.TrimSpace(p); p != "" {
      clean = append(clean, p)
    }
  }
  return clean
}

// topIndex returns the index of the first char from targets at bracket depth 0, or -1.
func topIndex(s, targets string) int {
  depth := 0
  for i := 0; i < len(s); i++ {
    c := s[i]
    switch c {
    case '(', '[', '{', '<':
      depth++
      continue
    case ')', ']', '}':
      depth--
      continue
    case '>':
      if i == 0 || s[i-1] != '=' {
        depth--
      }
      continue
    }
    if depth < 0 {
      depth = 0
    }
    if depth == 0 && strings.IndexByte(targets, c) >= 0 {
      return i
    }
  }
  return -1
}

func parseParam(p string) model.Param {
  p = strings.TrimSpace(p)
  for _, m := range []string{"public ", "private ", "protected ", "readonly "} {
    p = strings.TrimPrefix(p, m)
  }
  decl := p
  if eq := topIndex(p, "="); eq >= 0 {
    decl = strings.TrimSpace(p[:eq])
  }
  name, typ := decl, ""
  if c := topIndex(decl, ":"); c >= 0 {
    name = strings.TrimSpace(decl[:c])
    typ = strings.TrimSpace(decl[c+1:])
  }
  name = strings.TrimSuffix(name, "?")
  return model.Param{Name: name, Type: typ, TypeSource: srcOf(typ)}
}

func parseParams(s string) []model.Param {
  var out []model.Param
  for _, p := range splitTop(s) {
    out = append(out, parseParam(p))
  }
  return out
}

// stripStrings empties the content of quoted strings on one line (keeps the quotes).
func stripStrings(l string) string {
  var b strings.Builder
  var q byte
  for i := 0; i < len(l); i++ {
    c := l[i]
    if q != 0 {
      if c == '\\' {
        i++
        continue
      }
      if c == q {
        q = 0
        b.WriteByte(c)
      }
      continue
    }
    if c == '\'' || c == '"' || c == '`' {
      q = c
    }
    b.WriteByte(c)
  }
  return b.String()
}
