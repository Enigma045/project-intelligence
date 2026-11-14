package eval

import (
  "fmt"
  "hash/fnv"
  "path"
  "sort"
  "strconv"
  "strings"

  "pidx/internal/model"
)

// EdgeLabel is one hand-checked fact about the target project. Review: ok (the prefilled guess is right),
// fixed (you edited truth/no_link), skip (cannot tell). Anything else is ignored as unreviewed.
type EdgeLabel struct {
  File   string `json:"file"`
  Line   int    `json:"line"`
  Type   string `json:"type"`
  Name   string `json:"name"`
  Truth  string `json:"truth,omitempty"`
  NoLink bool   `json:"no_link,omitempty"`
  Review string `json:"review"`
  Code   string `json:"code,omitempty"`
  Was    string `json:"was,omitempty"`
  Note   string `json:"note,omitempty"`
}

type LabelFile struct {
  Name  string      `json:"name"`
  Edges []EdgeLabel `json:"edges"`
}

func Key(l EdgeLabel) string {
  return l.File + "|" + strconv.Itoa(l.Line) + "|" + l.Type + "|" + l.Name
}

func EdgeKey(e model.Edge) string {
  return e.File + "|" + strconv.Itoa(e.Line) + "|" + e.Type + "|" + e.DstName
}

// Snip normalises a source line for stale-label detection.
func Snip(s string) string {
  r := []rune(strings.TrimSpace(s))
  if len(r) > 140 {
    r = r[:140]
  }
  return string(r)
}

func LangOf(file string) string {
  switch strings.ToLower(path.Ext(file)) {
  case ".go":
    return "go"
  case ".js", ".jsx", ".mjs", ".cjs", ".ts", ".tsx", ".html", ".htm":
    return "js"
  case ".py":
    return "py"
  case ".sql":
    return "sql"
  case ".css":
    return "css"
  }
  return "other"
}

func linked(s string) bool { return s == "resolved" || s == "inferred" }

func match(dst, truth string) bool {
  return dst == truth || strings.HasSuffix(dst, "#"+truth) || strings.HasSuffix(dst, "/"+truth) || strings.HasSuffix(dst, ":"+truth)
}

// Judge compares one label with the edges pidx emitted for the same key.
// Outcomes: correct, wrong, missed, falselink, ambiguous, ok_nolink. The second value is the status of the
// edge that decided the outcome (for calibration).
func Judge(l EdgeLabel, es []model.Edge) (string, string) {
  if l.NoLink {
    for _, e := range es {
      if e.Dst != "" && linked(e.Status) {
        return "falselink", e.Status
      }
    }
    return "ok_nolink", ""
  }
  amb := false
  for _, e := range es {
    if e.Dst == "" || !match(e.Dst, l.Truth) {
      continue
    }
    if linked(e.Status) {
      return "correct", e.Status
    }
    if e.Status == "ambiguous" {
      amb = true
    }
  }
  if amb {
    return "ambiguous", "ambiguous"
  }
  for _, e := range es {
    if e.Dst != "" && linked(e.Status) {
      return "wrong", e.Status
    }
  }
  return "missed", ""
}

type Counts struct {
  Correct, Wrong, Missed, FalseLink, Ambiguous, NoLinkOK int
}

func (c *Counts) add(out string) {
  switch out {
  case "correct":
    c.Correct++
  case "wrong":
    c.Wrong++
  case "missed":
    c.Missed++
  case "falselink":
    c.FalseLink++
  case "ambiguous":
    c.Ambiguous++
  case "ok_nolink":
    c.NoLinkOK++
  }
}

// Precision: of the links pidx asserted on labelled edges, how many were right.
func (c Counts) Precision() (float64, int) {
  d := c.Correct + c.Wrong + c.FalseLink
  if d == 0 {
    return 0, 0
  }
  return float64(c.Correct) / float64(d), d
}

// Recall: of the labelled edges that should link, how many pidx linked correctly.
func (c Counts) Recall() (float64, int) {
  d := c.Correct + c.Wrong + c.Missed + c.Ambiguous
  if d == 0 {
    return 0, 0
  }
  return float64(c.Correct) / float64(d), d
}

type Failure struct {
  Label   EdgeLabel `json:"label"`
  Outcome string    `json:"outcome"`
  Got     []string  `json:"got"`
}

type Report struct {
  Used     int                `json:"used"`
  All      Counts             `json:"all"`
  ByType   map[string]*Counts `json:"by_type"`
  ByLang   map[string]*Counts `json:"by_lang"`
  ByStatus map[string]*Counts `json:"by_status"`
  Failures []Failure          `json:"failures"`
}

func bump(m map[string]*Counts, k, out string) {
  c := m[k]
  if c == nil {
    c = &Counts{}
    m[k] = c
  }
  c.add(out)
}

func gotList(es []model.Edge) []string {
  if len(es) == 0 {
    return []string{"(no edge emitted)"}
  }
  var out []string
  for _, e := range es {
    t := e.Dst
    if t == "" {
      t = e.DstName
    }
    out = append(out, fmt.Sprintf("%s %s", e.Status, t))
  }
  return out
}

// Evaluate scores reviewed labels against the current graph.
func Evaluate(labels []EdgeLabel, edges []model.Edge) Report {
  idx := map[string][]model.Edge{}
  for _, e := range edges {
    k := EdgeKey(e)
    idx[k] = append(idx[k], e)
  }
  r := Report{ByType: map[string]*Counts{}, ByLang: map[string]*Counts{}, ByStatus: map[string]*Counts{}}
  for _, l := range labels {
    es := idx[Key(l)]
    out, st := Judge(l, es)
    r.Used++
    r.All.add(out)
    bump(r.ByType, l.Type, out)
    bump(r.ByLang, LangOf(l.File), out)
    if linked(st) {
      bump(r.ByStatus, st, out)
    }
    if out != "correct" && out != "ok_nolink" {
      r.Failures = append(r.Failures, Failure{Label: l, Outcome: out, Got: gotList(es)})
    }
  }
  return r
}

// DefaultTypes are the edge types worth labelling by hand (styles and imports are noise or trivial).
var DefaultTypes = []string{"calls", "listens", "listens_dom", "references_dom", "requests_api", "handles_route", "queries_table"}

func hash(seed, key string) uint32 {
  h := fnv.New32a()
  _, _ = h.Write([]byte(seed + "|" + key))
  return h.Sum32()
}

// SampleEdges picks up to n edges, round-robin across edge types so one type cannot dominate.
// Deterministic for a given seed. Keys in skip (already labelled) are never picked.
func SampleEdges(edges []model.Edge, types []string, n int, seed string, skip map[string]bool) []model.Edge {
  allow := map[string]bool{}
  for _, t := range types {
    allow[strings.TrimSpace(t)] = true
  }
  seen := map[string]bool{}
  groups := map[string][]model.Edge{}
  for _, e := range edges {
    if !allow[e.Type] || e.Line <= 0 || e.File == "" || strings.HasPrefix(e.DstName, "?.") {
      continue
    }
    k := EdgeKey(e)
    if skip[k] || seen[k] {
      continue
    }
    seen[k] = true
    groups[e.Type] = append(groups[e.Type], e)
  }
  var names []string
  for t, g := range groups {
    g := g
    sort.Slice(g, func(i, j int) bool {
      ki, kj := EdgeKey(g[i]), EdgeKey(g[j])
      hi, hj := hash(seed, ki), hash(seed, kj)
      if hi != hj {
        return hi < hj
      }
      return ki < kj
    })
    groups[t] = g
    names = append(names, t)
  }
  sort.Strings(names)
  var out []model.Edge
  for i := 0; len(out) < n; i++ {
    progressed := false
    for _, t := range names {
      if i < len(groups[t]) {
        out = append(out, groups[t][i])
        progressed = true
        if len(out) >= n {
          break
        }
      }
    }
    if !progressed {
      break
    }
  }
  return out
}

// NewLabel prefills a label from the current graph. Linked mode guesses the current target is right;
// unlinked mode guesses there should be no project link.
func NewLabel(e model.Edge, linkedMode bool) EdgeLabel {
  l := EdgeLabel{File: e.File, Line: e.Line, Type: e.Type, Name: e.DstName, Review: "?", Was: e.Status}
  if e.Dst != "" {
    l.Was += " -> " + e.Dst
  }
  if linkedMode && e.Dst != "" {
    l.Truth = e.Dst
  } else {
    l.NoLink = true
  }
  return l
}
