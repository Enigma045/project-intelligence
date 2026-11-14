package edit

import (
  "context"
  "encoding/json"
  "fmt"
  "go/parser"
  "go/token"
  "os"
  "os/exec"
  "path/filepath"
  "strings"
  "time"
)

// SyntaxCheck validates staged content before it is written. checked=false means no checker applied
// (unknown file type, or the needed tool such as node/python is not installed): that is not a pass.
func SyntaxCheck(rel string, content []byte) (bool, error) {
  switch strings.ToLower(filepath.Ext(rel)) {
  case ".go":
    _, err := parser.ParseFile(token.NewFileSet(), rel, content, parser.AllErrors|parser.SkipObjectResolution)
    if err != nil {
      return true, fmt.Errorf("%s", trimMsg(err.Error()))
    }
    return true, nil
  case ".json":
    if json.Valid(content) {
      return true, nil
    }
    return true, fmt.Errorf("invalid JSON")
  case ".css":
    return true, balanced(string(content))
  case ".js", ".mjs", ".cjs":
    return checkWithTool(rel, content, "node", []string{"--version"}, []string{"--check"})
  case ".py":
    snippet := "import ast,sys; ast.parse(open(sys.argv[1], encoding='utf-8').read())"
    for _, bin := range []string{"python", "python3"} {
      if checked, err := checkWithTool(rel, content, bin, []string{"-c", "pass"}, []string{"-c", snippet}); checked {
        return true, err
      }
    }
  }
  return false, nil
}

func trimMsg(s string) string {
  if len(s) > 500 {
    return s[:500] + "..."
  }
  return s
}

// balanced checks brace and paren balance, ignoring strings and /* comments */.
func balanced(s string) error {
  braces, parens, line := 0, 0, 1
  var q byte
  for i := 0; i < len(s); i++ {
    c := s[i]
    if c == '\n' {
      line++
    }
    if q != 0 {
      if c == '\n' {
        q = 0
      } else if c == '\\' {
        i++
      } else if c == q {
        q = 0
      }
      continue
    }
    if c == '/' && i+1 < len(s) && s[i+1] == '*' {
      j := strings.Index(s[i+2:], "*/")
      if j < 0 {
        return fmt.Errorf("unterminated comment at line %d", line)
      }
      line += strings.Count(s[i:i+2+j], "\n")
      i += 2 + j + 1
      continue
    }
    switch c {
    case '"', '\'':
      q = c
    case '{':
      braces++
    case '}':
      braces--
      if braces < 0 {
        return fmt.Errorf("unmatched } at line %d", line)
      }
    case '(':
      parens++
    case ')':
      parens--
      if parens < 0 {
        return fmt.Errorf("unmatched ) at line %d", line)
      }
    }
  }
  if braces != 0 {
    return fmt.Errorf("%d unclosed {", braces)
  }
  if parens != 0 {
    return fmt.Errorf("%d unclosed (", parens)
  }
  return nil
}

// checkWithTool runs an external syntax checker on a temp copy. Returns checked=false if the tool is
// missing or does not even start (so a broken python launcher never rejects a good edit).
func checkWithTool(rel string, content []byte, bin string, probe, before []string) (bool, error) {
  p, err := exec.LookPath(bin)
  if err != nil {
    return false, nil
  }
  if err := exec.Command(p, probe...).Run(); err != nil {
    return false, nil
  }
  dir, err := os.MkdirTemp("", "pidx-check-")
  if err != nil {
    return false, nil
  }
  defer os.RemoveAll(dir)
  tmp := filepath.Join(dir, "check"+filepath.Ext(rel))
  if err := os.WriteFile(tmp, content, 0o600); err != nil {
    return false, nil
  }
  ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
  defer cancel()
  args := append(append([]string{}, before...), tmp)
  out, err := exec.CommandContext(ctx, p, args...).CombinedOutput()
  if err != nil {
    msg := strings.TrimSpace(string(out))
    if msg == "" {
      msg = err.Error()
    }
    return true, fmt.Errorf("%s", trimMsg(msg))
  }
  return true, nil
}
