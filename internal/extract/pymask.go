package extract

import "strings"

// maskTriple blanks the contents of triple-quoted strings (multi-line strings and docstrings) so code-like text
// inside them, such as JS embedded in a Python script, is not scanned for calls. Line count and the quote
// delimiters are kept. Single-line strings are copied through untouched; comments run to the end of the line.
func maskTriple(lines []string) []string {
  out := make([]string, len(lines))
  delim := ""
  for i, l := range lines {
    var b strings.Builder
    j := 0
    for j < len(l) {
      if delim != "" {
        k := strings.Index(l[j:], delim)
        if k < 0 {
          b.WriteString(strings.Repeat(" ", len(l)-j))
          j = len(l)
          break
        }
        b.WriteString(strings.Repeat(" ", k))
        b.WriteString(delim)
        j += k + 3
        delim = ""
        continue
      }
      c := l[j]
      if c == '#' {
        b.WriteString(l[j:])
        j = len(l)
        break
      }
      if strings.HasPrefix(l[j:], `"""`) || strings.HasPrefix(l[j:], "'''") {
        delim = l[j : j+3]
        b.WriteString(delim)
        j += 3
        continue
      }
      if c == '"' || c == '\'' {
        q := c
        b.WriteByte(c)
        j++
        for j < len(l) {
          if l[j] == '\\' && j+1 < len(l) {
            b.WriteString(l[j : j+2])
            j += 2
            continue
          }
          b.WriteByte(l[j])
          if l[j] == q {
            j++
            break
          }
          j++
        }
        continue
      }
      b.WriteByte(c)
      j++
    }
    out[i] = b.String()
  }
  return out
}
