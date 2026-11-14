package main

import (
  "flag"
  "fmt"
  "os"
  "path/filepath"
  "sort"
  "strings"
)

// Patch format (*.patch), any number of blocks:
//   @@FILE relative/path
//   @@OLD
//   exact text, must match exactly once
//   @@NEW
//   replacement text
//   @@END
// All-or-nothing: nothing is written unless every block applies. Re-running is safe
// (a block whose NEW text is already present is skipped). Originals go to patches/backup.

type edit struct{ file, old, neu, src string }

func parse(p string) ([]edit, error) {
  data, err := os.ReadFile(p)
  if err != nil {
    return nil, err
  }
  lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
  var edits []edit
  var cur *edit
  var oldB, newB []string
  mode, file := "", ""
  for n, l := range lines {
    switch {
    case strings.HasPrefix(l, "@@FILE "):
      file = strings.TrimSpace(l[7:])
    case l == "@@OLD":
      cur = &edit{file: file, src: fmt.Sprintf("%s:%d", filepath.Base(p), n+1)}
      oldB, newB, mode = nil, nil, "old"
    case l == "@@NEW":
      mode = "new"
    case l == "@@END":
      if cur == nil {
        return nil, fmt.Errorf("%s:%d: @@END without @@OLD", p, n+1)
      }
      cur.old, cur.neu = strings.Join(oldB, "\n"), strings.Join(newB, "\n")
      edits = append(edits, *cur)
      cur, mode = nil, ""
    default:
      if mode == "old" {
        oldB = append(oldB, l)
      } else if mode == "new" {
        newB = append(newB, l)
      }
    }
  }
  if cur != nil {
    return nil, fmt.Errorf("%s: unterminated block (missing @@END)", p)
  }
  return edits, nil
}

type fstate struct {
  text    string
  crlf    bool
  changed bool
}

func main() {
  root := flag.String("root", ".", "project root")
  flag.Parse()
  if flag.NArg() < 1 {
    fmt.Fprintln(os.Stderr, "usage: applypatch [-root dir] <patch dir>")
    os.Exit(2)
  }
  dir := flag.Arg(0)
  if !filepath.IsAbs(dir) {
    dir = filepath.Join(*root, dir)
  }
  names, _ := filepath.Glob(filepath.Join(dir, "*.patch"))
  sort.Strings(names)
  var edits []edit
  for _, n := range names {
    es, err := parse(n)
    if err != nil {
      fmt.Fprintln(os.Stderr, "error:", err)
      os.Exit(1)
    }
    edits = append(edits, es...)
  }
  states := map[string]*fstate{}
  var order []string
  failed := 0
  for _, e := range edits {
    st := states[e.file]
    if st == nil {
      b, err := os.ReadFile(filepath.Join(*root, filepath.FromSlash(e.file)))
      if err != nil {
        fmt.Printf("FAIL %s (%s): %v\n", e.file, e.src, err)
        failed++
        continue
      }
      t := string(b)
      st = &fstate{text: strings.ReplaceAll(t, "\r\n", "\n"), crlf: strings.Contains(t, "\r\n")}
      states[e.file] = st
      order = append(order, e.file)
    }
    if e.old == "" {
      fmt.Printf("FAIL %s (%s): empty OLD block\n", e.file, e.src)
      failed++
      continue
    }
    if e.neu != "" && strings.Contains(st.text, e.neu) {
      fmt.Printf("skip %s (%s): already applied\n", e.file, e.src)
      continue
    }
    if n := strings.Count(st.text, e.old); n != 1 {
      fmt.Printf("FAIL %s (%s): OLD must match exactly once, found %d\n", e.file, e.src, n)
      failed++
      continue
    }
    st.text = strings.Replace(st.text, e.old, e.neu, 1)
    st.changed = true
    fmt.Printf("ok   %s (%s)\n", e.file, e.src)
  }
  if failed > 0 {
    fmt.Println("patch failed; no files were written")
    os.Exit(1)
  }
  for _, f := range order {
    st := states[f]
    if !st.changed {
      continue
    }
    full := filepath.Join(*root, filepath.FromSlash(f))
    backup := filepath.Join(*root, "patches", "backup", filepath.FromSlash(f)+".orig")
    if _, err := os.Stat(backup); err != nil {
      if orig, rerr := os.ReadFile(full); rerr == nil {
        _ = os.MkdirAll(filepath.Dir(backup), 0o755)
        _ = os.WriteFile(backup, orig, 0o644)
      }
    }
    out := st.text
    if st.crlf {
      out = strings.ReplaceAll(out, "\n", "\r\n")
    }
    if err := os.WriteFile(full, []byte(out), 0o644); err != nil {
      fmt.Fprintln(os.Stderr, "error:", err)
      os.Exit(1)
    }
    fmt.Println("wrote", f)
  }
}
