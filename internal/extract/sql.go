package extract

import (
  "regexp"
  "strings"

  "pidx/internal/model"
)

var (
  reSQLTable   = regexp.MustCompile(`(?i)\bcreate\s+(?:temp(?:orary)?\s+)?table\s+(?:if\s+not\s+exists\s+)?[\x22\x60]?([A-Za-z_][\w.]*)[\x22\x60]?\s*\(`)
  reSQLView    = regexp.MustCompile(`(?i)\bcreate\s+(?:or\s+replace\s+)?(?:temp(?:orary)?\s+)?view\s+(?:if\s+not\s+exists\s+)?[\x22\x60]?([A-Za-z_][\w.]*)`)
  reSQLIndex   = regexp.MustCompile(`(?i)\bcreate\s+(?:unique\s+)?index\s+(?:if\s+not\s+exists\s+)?[\x22\x60]?([A-Za-z_]\w*)[\x22\x60]?\s+on\s+[\x22\x60]?([A-Za-z_][\w.]*)`)
  reSQLTrigger = regexp.MustCompile(`(?i)\bcreate\s+(?:temp(?:orary)?\s+)?trigger\s+(?:if\s+not\s+exists\s+)?[\x22\x60]?([A-Za-z_]\w*)`)
  reSQLRefs    = regexp.MustCompile(`(?i)\breferences\s+[\x22\x60]?([A-Za-z_][\w.]*)`)
)

var sqlTypeStop = map[string]bool{
  "primary": true, "not": true, "null": true, "references": true, "unique": true, "default": true,
  "check": true, "constraint": true, "collate": true, "generated": true, "autoincrement": true,
}

var sqlTableKw = map[string]bool{
  "primary": true, "foreign": true, "unique": true, "constraint": true, "check": true, "key": true,
  "index": true, "fulltext": true, "exclude": true,
}

type sqlPart struct {
  text string
  off  int
}

func lastSeg(n string) string {
  if i := strings.LastIndex(n, "."); i >= 0 {
    return n[i+1:]
  }
  return n
}

// blankDashComments replaces -- line comments with spaces, keeping offsets and newlines.
func blankDashComments(s string) string {
  b := []byte(s)
  for i := 0; i+1 < len(b); i++ {
    if b[i] == '-' && b[i+1] == '-' {
      for j := i; j < len(b) && b[j] != '\n'; j++ {
        b[j] = ' '
      }
    }
  }
  return string(b)
}

func matchParenSQL(s string, open int) int {
  depth := 0
  inQ := false
  for i := open; i < len(s); i++ {
    c := s[i]
    if inQ {
      if c == '\'' {
        inQ = false
      }
      continue
    }
    switch c {
    case '\'':
      inQ = true
    case '(':
      depth++
    case ')':
      depth--
      if depth == 0 {
        return i
      }
    }
  }
  return -1
}

// splitParens splits a table body on top-level commas; off is relative to the statement text.
func splitParens(body string, base int) []sqlPart {
  var out []sqlPart
  depth, start := 0, 0
  inQ := false
  for i := 0; i < len(body); i++ {
    c := body[i]
    if inQ {
      if c == '\'' {
        inQ = false
      }
      continue
    }
    switch c {
    case '\'':
      inQ = true
    case '(':
      depth++
    case ')':
      depth--
    case ',':
      if depth == 0 {
        out = append(out, sqlPart{body[start:i], base + start})
        start = i + 1
      }
    }
  }
  out = append(out, sqlPart{body[start:], base + start})
  return out
}

func sqlColumn(p sqlPart, table string, ln func(int) int, f *model.FileFacts) {
  t := strings.TrimSpace(p.text)
  if t == "" {
    return
  }
  lead := len(p.text) - len(strings.TrimLeft(p.text, " \t\r\n"))
  line := ln(p.off + lead)
  fs := strings.Fields(t)
  first := strings.ToLower(strings.Trim(fs[0], "\x22\x60[]"))
  if sqlTableKw[first] {
    for _, r := range reSQLRefs.FindAllStringSubmatch(t, -1) {
      f.Refs = append(f.Refs, model.RawRef{Kind: "sql_fk", Name: lastSeg(r[1]), Line: line, From: table})
    }
    return
  }
  typ := ""
  if len(fs) > 1 && !sqlTypeStop[strings.ToLower(fs[1])] {
    typ = fs[1]
  }
  low := strings.ToLower(t)
  ret := typ
  if strings.Contains(low, "primary key") {
    ret += " PK"
  }
  if strings.Contains(low, "not null") {
    ret += " NOT NULL"
  }
  name := strings.Trim(fs[0], "\x22\x60[]")
  for _, r := range reSQLRefs.FindAllStringSubmatch(t, -1) {
    ref := lastSeg(r[1])
    ret += " FK->" + ref
    f.Refs = append(f.Refs, model.RawRef{Kind: "sql_fk", Name: ref, Target: name, Line: line, From: table})
  }
  f.Symbols = append(f.Symbols, model.Symbol{
    Kind: "column", Name: name, Parent: table, StartLine: line, EndLine: line,
    Returns: strings.TrimSpace(ret), ReturnsSource: srcOf(typ), Visibility: "public",
    BodyHash: model.HashString(t), Provenance: "inferred",
  })
}

// sqlFacts extracts DDL from text. lineBase is the 1-based line where text starts in the host file.
func sqlFacts(raw string, lineBase int, f *model.FileFacts) {
  s := blankComments(blankDashComments(raw))
  if s == "" {
    return
  }
  starts := lineStarts(s)
  ln := func(off int) int { return lineBase + lineAt(starts, off) - 1 }
  for _, m := range reSQLTable.FindAllStringSubmatchIndex(s, -1) {
    name := lastSeg(s[m[2]:m[3]])
    open := m[1] - 1
    end := matchParenSQL(s, open)
    if end < 0 {
      continue
    }
    f.Symbols = append(f.Symbols, model.Symbol{
      Kind: "table", Name: name, StartLine: ln(m[0]), EndLine: ln(end), Visibility: "public",
      BodyHash: model.HashString(s[m[0] : end+1]), Provenance: "inferred",
    })
    for _, p := range splitParens(s[open+1:end], open+1) {
      sqlColumn(p, name, ln, f)
    }
  }
  stmtEnd := func(from int) int {
    if i := strings.IndexByte(s[from:], ';'); i >= 0 {
      return from + i
    }
    return len(s) - 1
  }
  for _, m := range reSQLView.FindAllStringSubmatchIndex(s, -1) {
    e := stmtEnd(m[0])
    f.Symbols = append(f.Symbols, model.Symbol{
      Kind: "view", Name: lastSeg(s[m[2]:m[3]]), StartLine: ln(m[0]), EndLine: ln(e), Visibility: "public",
      BodyHash: model.HashString(s[m[0] : e+1]), Provenance: "inferred",
    })
  }
  for _, m := range reSQLIndex.FindAllStringSubmatchIndex(s, -1) {
    e := stmtEnd(m[0])
    f.Symbols = append(f.Symbols, model.Symbol{
      Kind: "index", Name: s[m[2]:m[3]], Parent: lastSeg(s[m[4]:m[5]]), StartLine: ln(m[0]), EndLine: ln(e),
      Visibility: "public", BodyHash: model.HashString(s[m[0] : e+1]), Provenance: "inferred",
    })
  }
  for _, m := range reSQLTrigger.FindAllStringSubmatchIndex(s, -1) {
    e := stmtEnd(m[0])
    f.Symbols = append(f.Symbols, model.Symbol{
      Kind: "trigger", Name: s[m[2]:m[3]], StartLine: ln(m[0]), EndLine: ln(e), Visibility: "public",
      BodyHash: model.HashString(s[m[0] : e+1]), Provenance: "inferred",
    })
  }
}

// extractSQL emits table, column, view, index and trigger symbols plus sql_fk refs.
func extractSQL(path string, src []byte) model.FileFacts {
  facts := model.FileFacts{Path: path, Language: "sql", ParseStatus: "heuristic"}
  sqlFacts(string(src), 1, &facts)
  return facts
}
