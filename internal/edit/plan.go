package edit

import (
  "fmt"
  "path"
  "sort"
  "strconv"
  "strings"

  "pidx/internal/model"
)

// Op is one edit. Ops: replace | replace_symbol | insert_before_symbol | insert_after_symbol |
// delete_symbol | create | delete_file. Symbol ops address code by symbol id (never by line number).
type Op struct {
  File     string `json:"file,omitempty"`
  Op       string `json:"op"`
  BaseHash string `json:"base_hash,omitempty"`
  Old      string `json:"old,omitempty"`
  New      string `json:"new,omitempty"`
  Symbol   string `json:"symbol,omitempty"`
  Text     string `json:"text,omitempty"`
}

type Request struct {
  Edits            []Op   `json:"edits"`
  DryRun           bool   `json:"dry_run,omitempty"`
  KeepOnFail       bool   `json:"keep_on_fail,omitempty"`
  AllowLargeDelete bool   `json:"allow_large_delete,omitempty"`
  TestCommand      string `json:"test_command,omitempty"`
}

// SymRange is where the index says a symbol lives, plus the hash of the file the index saw.
type SymRange struct {
  File       string
  Start, End int
  FileHash   string
}

// Env isolates all I/O so Plan is a pure function of its inputs.
type Env struct {
  Read   func(rel string) ([]byte, bool, error)
  Lookup func(q string) ([]SymRange, error)
}

type Staged struct {
  Path    string
  Before  []byte
  Existed bool
  After   []byte
  Delete  bool
}

var symOps = map[string]bool{"replace_symbol": true, "insert_before_symbol": true, "insert_after_symbol": true, "delete_symbol": true}

// SafePath validates and normalises a project-relative path.
func SafePath(rel string) (string, error) {
  if rel == "" {
    return "", fmt.Errorf("missing file path")
  }
  if strings.Contains(rel, "\\") || strings.HasPrefix(rel, "/") || (len(rel) > 1 && rel[1] == ':') {
    return "", fmt.Errorf("%q: use a relative path with forward slashes", rel)
  }
  c := path.Clean(rel)
  if c == "." || c == ".." || strings.HasPrefix(c, "../") {
    return "", fmt.Errorf("%q escapes the project root", rel)
  }
  first := strings.SplitN(c, "/", 2)[0]
  if first == ".git" || first == ".project-intelligence" {
    return "", fmt.Errorf("%q is protected", rel)
  }
  return c, nil
}

type symEdit struct {
  op  Op
  r   SymRange
  idx int
}

type fstate struct {
  orig    []byte
  exists  bool
  text    string
  crlf    bool
  created bool
  deleted bool
  sym     []symEdit
  txt     []int
}

func norm(s string) string { return strings.ReplaceAll(s, "\r\n", "\n") }

func splitText(t string) []string {
  return strings.Split(strings.TrimSuffix(norm(t), "\n"), "\n")
}

func splice(lines []string, a, b int, ins []string) []string {
  out := make([]string, 0, len(lines)-(b-a)+len(ins))
  out = append(out, lines[:a]...)
  out = append(out, ins...)
  out = append(out, lines[b:]...)
  return out
}

func firstLine(s string) string {
  if i := strings.Index(s, "\n"); i >= 0 {
    s = s[:i]
  }
  if len(s) > 60 {
    s = s[:60]
  }
  return s
}

func lineList(text, old string) string {
  var ls []string
  off := 0
  for len(ls) < 5 && off < len(text) {
    i := strings.Index(text[off:], old)
    if i < 0 {
      break
    }
    ls = append(ls, strconv.Itoa(strings.Count(text[:off+i], "\n")+1))
    off += i + 1
  }
  return strings.Join(ls, ",")
}

// applySymbolOps applies symbol-addressed ops using the ORIGINAL line numbers, bottom-up so earlier
// edits never shift later ones. Overlapping replace/delete ranges are rejected.
func applySymbolOps(text string, ops []symEdit) (string, error) {
  lines := strings.Split(text, "\n")
  type iv struct{ s, e, idx int }
  var ivs []iv
  for _, o := range ops {
    if o.r.Start < 1 || o.r.End < o.r.Start || o.r.End > len(lines) {
      return "", fmt.Errorf("edit %d: symbol range %d-%d is outside the file", o.idx+1, o.r.Start, o.r.End)
    }
    if o.op.Op == "replace_symbol" || o.op.Op == "delete_symbol" {
      ivs = append(ivs, iv{o.r.Start, o.r.End, o.idx})
    }
  }
  sort.Slice(ivs, func(i, j int) bool { return ivs[i].s < ivs[j].s })
  for k := 1; k < len(ivs); k++ {
    if ivs[k].s <= ivs[k-1].e {
      return "", fmt.Errorf("edits %d and %d overlap (replace/delete the same lines)", ivs[k-1].idx+1, ivs[k].idx+1)
    }
  }
  anchor := func(o symEdit) int {
    if o.op.Op == "insert_after_symbol" {
      return o.r.End
    }
    return o.r.Start - 1
  }
  rank := func(o symEdit) int {
    if o.op.Op == "replace_symbol" || o.op.Op == "delete_symbol" {
      return 0
    }
    return 1
  }
  sorted := append([]symEdit(nil), ops...)
  sort.SliceStable(sorted, func(i, j int) bool {
    a, b := anchor(sorted[i]), anchor(sorted[j])
    if a != b {
      return a > b
    }
    return rank(sorted[i]) < rank(sorted[j])
  })
  for _, o := range sorted {
    s, e := o.r.Start, o.r.End
    switch o.op.Op {
    case "replace_symbol":
      lines = splice(lines, s-1, e, splitText(o.op.Text))
    case "insert_before_symbol":
      lines = splice(lines, s-1, s-1, splitText(o.op.Text))
    case "insert_after_symbol":
      lines = splice(lines, e, e, splitText(o.op.Text))
    case "delete_symbol":
      end := e
      if end < len(lines)-1 && strings.TrimSpace(lines[end]) == "" {
        end++
      }
      lines = splice(lines, s-1, end, nil)
    }
  }
  return strings.Join(lines, "\n"), nil
}

// Plan validates every op and returns the staged result for each changed file. Nothing is written.
// All problems are collected so the caller can fix them in one round trip.
func Plan(ops []Op, env Env, allowLarge bool) ([]Staged, error) {
  var errs []string
  bad := func(i int, f string, a ...any) {
    errs = append(errs, fmt.Sprintf("edit %d (%s): %s", i+1, ops[i].Op, fmt.Sprintf(f, a...)))
  }
  states := map[string]*fstate{}
  var order []string
  load := func(i int, rel string) *fstate {
    p, err := SafePath(rel)
    if err != nil {
      bad(i, "%v", err)
      return nil
    }
    if st, ok := states[p]; ok {
      return st
    }
    b, exists, err := env.Read(p)
    if err != nil {
      bad(i, "%v", err)
      return nil
    }
    st := &fstate{orig: b, exists: exists}
    if exists {
      s := string(b)
      st.crlf = strings.Contains(s, "\r\n")
      st.text = norm(s)
    }
    states[p] = st
    order = append(order, p)
    return st
  }

  pathOf := map[*fstate]string{}
  for i, op := range ops {
    if !symOps[op.Op] && op.Op != "create" && op.Op != "delete_file" && op.Op != "replace" {
      bad(i, "unknown op %q", op.Op)
      continue
    }
    file := op.File
    var sr SymRange
    if symOps[op.Op] {
      if op.Symbol == "" {
        bad(i, "missing symbol")
        continue
      }
      rs, err := env.Lookup(op.Symbol)
      if err != nil {
        bad(i, "%v", err)
        continue
      }
      if len(rs) != 1 {
        bad(i, "symbol %q matched %d definitions; use a full symbol id (see: pidx search)", op.Symbol, len(rs))
        continue
      }
      sr = rs[0]
      if file != "" && file != sr.File {
        bad(i, "symbol %q is in %s, not %s", op.Symbol, sr.File, file)
        continue
      }
      file = sr.File
    }
    st := load(i, file)
    if st == nil {
      continue
    }
    if _, ok := pathOf[st]; !ok {
      p, _ := SafePath(file)
      pathOf[st] = p
    }
    if op.BaseHash != "" && st.exists && op.BaseHash != model.HashBytes(st.orig) {
      bad(i, "base_hash mismatch for %s: the file changed since you read it; re-read it", pathOf[st])
      continue
    }
    switch op.Op {
    case "create":
      if st.exists || st.created {
        bad(i, "%s already exists", pathOf[st])
        continue
      }
      st.created = true
      st.text = norm(op.Text)
    case "delete_file":
      if !st.exists || st.deleted {
        bad(i, "%s does not exist", pathOf[st])
        continue
      }
      st.deleted = true
    case "replace":
      if op.Old == "" {
        bad(i, "empty old text")
        continue
      }
      if !st.exists && !st.created {
        bad(i, "%s does not exist", pathOf[st])
        continue
      }
      st.txt = append(st.txt, i)
    default:
      if st.created || !st.exists {
        bad(i, "symbol ops need an indexed file")
        continue
      }
      if sr.FileHash != model.HashBytes(st.orig) {
        bad(i, "index is stale for %s; run pidx scan and retry", pathOf[st])
        continue
      }
      if op.Op != "delete_symbol" && strings.TrimSpace(op.Text) == "" {
        bad(i, "empty text")
        continue
      }
      st.sym = append(st.sym, symEdit{op, sr, i})
    }
  }

  var out []Staged
  for _, p := range order {
    st := states[p]
    if st.deleted {
      if len(st.sym)+len(st.txt) > 0 {
        errs = append(errs, p+": edited and deleted in the same request")
        continue
      }
      out = append(out, Staged{Path: p, Before: st.orig, Existed: true, Delete: true})
      continue
    }
    text := st.text
    if len(st.sym) > 0 {
      t, err := applySymbolOps(text, st.sym)
      if err != nil {
        errs = append(errs, p+": "+err.Error())
        continue
      }
      text = t
    }
    failed := false
    for _, i := range st.txt {
      old, neu := norm(ops[i].Old), norm(ops[i].New)
      n := strings.Count(text, old)
      if n == 0 {
        bad(i, "old text not found in %s (starts with %q)", p, firstLine(old))
        failed = true
        continue
      }
      if n > 1 {
        bad(i, "old text matches %d times in %s (lines %s); add surrounding context", n, p, lineList(text, old))
        failed = true
        continue
      }
      text = strings.Replace(text, old, neu, 1)
    }
    if failed {
      continue
    }
    if st.exists && !st.created && text == st.text {
      continue
    }
    if st.exists && !allowLarge {
      before, after := strings.Count(st.text, "\n")+1, strings.Count(text, "\n")+1
      if before > 20 && after*2 < before {
        errs = append(errs, fmt.Sprintf("%s: would remove more than half of the file (%d -> %d lines); set allow_large_delete to confirm", p, before, after))
        continue
      }
    }
    after := text
    if st.crlf {
      after = strings.ReplaceAll(after, "\n", "\r\n")
    }
    out = append(out, Staged{Path: p, Before: st.orig, Existed: st.exists, After: []byte(after)})
  }
  if len(errs) > 0 {
    return nil, fmt.Errorf("%s", strings.Join(errs, "\n"))
  }
  sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
  return out, nil
}
