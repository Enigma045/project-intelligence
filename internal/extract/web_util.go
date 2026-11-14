package extract

import (
  "sort"

  "pidx/internal/model"
)

func lineStarts(s string) []int {
  st := []int{0}
  for i := 0; i < len(s); i++ {
    if s[i] == '\n' {
      st = append(st, i+1)
    }
  }
  return st
}

// lineAt returns the 1-based line containing byte offset off.
func lineAt(starts []int, off int) int {
  return sort.SearchInts(starts, off+1)
}

// asciiLower lowercases ASCII only, so byte offsets stay valid.
func asciiLower(s string) string {
  b := []byte(s)
  for i, c := range b {
    if c >= 'A' && c <= 'Z' {
      b[i] = c + 32
    }
  }
  return string(b)
}

func isWS(c byte) bool {
  return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f'
}

func isLetter(c byte) bool {
  return (c|32) >= 'a' && (c|32) <= 'z'
}

// mergeInline appends facts extracted from an embedded block (inline script/style), shifting line numbers.
func mergeInline(dst *model.FileFacts, f model.FileFacts, offset int) {
  for _, s := range f.Symbols {
    s.StartLine += offset
    s.EndLine += offset
    dst.Symbols = append(dst.Symbols, s)
  }
  for _, r := range f.Refs {
    r.Line += offset
    dst.Refs = append(dst.Refs, r)
  }
}
