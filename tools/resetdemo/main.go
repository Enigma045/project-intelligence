package main

import (
  "fmt"
  "os"
)

func main() {
  files := map[string]string{
    "demo/a.py": "def foo(x):\n    return x + 1\n",
    "demo/b.py": "import a\n\ndef run():\n    return a.foo(2)\n",
  }
  if err := os.MkdirAll("demo", 0o755); err != nil {
    fmt.Fprintln(os.Stderr, "error:", err)
    os.Exit(1)
  }
  for p, c := range files {
    if err := os.WriteFile(p, []byte(c), 0o644); err != nil {
      fmt.Fprintln(os.Stderr, "error:", err)
      os.Exit(1)
    }
  }
  fmt.Println("demo files reset")
}
