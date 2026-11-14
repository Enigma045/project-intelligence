package eval

import (
  "strings"
  "testing"

  "pidx/internal/model"
)

func ed(file string, line int, typ, name, dst, st string) model.Edge {
  return model.Edge{File: file, Line: line, Type: typ, DstName: name, Dst: dst, Status: st}
}

func TestJudge(t *testing.T) {
  l := EdgeLabel{File: "a.js", Line: 3, Type: "calls", Name: "f", Truth: "function:a.js#f"}
  cases := []struct {
    es   []model.Edge
    want string
  }{
    {[]model.Edge{ed("a.js", 3, "calls", "f", "function:a.js#f", "resolved")}, "correct"},
    {[]model.Edge{ed("a.js", 3, "calls", "f", "function:b.js#f", "resolved")}, "wrong"},
    {[]model.Edge{ed("a.js", 3, "calls", "f", "", "value")}, "missed"},
    {nil, "missed"},
    {[]model.Edge{ed("a.js", 3, "calls", "f", "function:a.js#f", "ambiguous"), ed("a.js", 3, "calls", "f", "function:b.js#f", "ambiguous")}, "ambiguous"},
  }
  for i, c := range cases {
    if got, _ := Judge(l, c.es); got != c.want {
      t.Errorf("case %d: want %s got %s", i, c.want, got)
    }
  }
  short := EdgeLabel{File: "a.js", Line: 3, Type: "calls", Name: "f", Truth: "f"}
  if got, _ := Judge(short, cases[0].es); got != "correct" {
    t.Errorf("suffix truth must match: %s", got)
  }
  nl := EdgeLabel{File: "a.js", Line: 3, Type: "calls", Name: "f", NoLink: true}
  if got, _ := Judge(nl, cases[0].es); got != "falselink" {
    t.Errorf("want falselink got %s", got)
  }
  if got, _ := Judge(nl, []model.Edge{ed("a.js", 3, "calls", "f", "", "external")}); got != "ok_nolink" {
    t.Errorf("want ok_nolink got %s", got)
  }
}

func TestEvaluate(t *testing.T) {
  edges := []model.Edge{
    ed("a.go", 1, "calls", "x", "function:a.go#x", "resolved"),
    ed("a.go", 2, "calls", "y", "function:a.go#y", "resolved"),
    ed("a.go", 3, "calls", "z", "function:b.go#WRONG", "inferred"),
  }
  labels := []EdgeLabel{
    {File: "a.go", Line: 1, Type: "calls", Name: "x", Truth: "function:a.go#x"},
    {File: "a.go", Line: 2, Type: "calls", Name: "y", Truth: "function:a.go#y"},
    {File: "a.go", Line: 3, Type: "calls", Name: "z", Truth: "function:a.go#z"},
  }
  r := Evaluate(labels, edges)
  p, n := r.All.Precision()
  if n != 3 || p < 0.66 || p > 0.67 {
    t.Fatalf("precision %v over %d", p, n)
  }
  if c := r.ByLang["go"]; c == nil || c.Correct != 2 || c.Wrong != 1 {
    t.Fatalf("bad lang counts: %+v", c)
  }
  if c := r.ByStatus["inferred"]; c == nil || c.Wrong != 1 {
    t.Fatalf("bad status counts: %+v", c)
  }
  if len(r.Failures) != 1 || r.Failures[0].Outcome != "wrong" {
    t.Fatalf("bad failures: %+v", r.Failures)
  }
}

func TestSampleEdges(t *testing.T) {
  var edges []model.Edge
  for i := 1; i <= 30; i++ {
    typ := "calls"
    if i%3 == 0 {
      typ = "requests_api"
    }
    edges = append(edges, ed("f.js", i, typ, "n", "function:f.js#n", "resolved"))
  }
  edges = append(edges, ed("f.js", 99, "imports", "x", "file:x", "resolved"))
  a := SampleEdges(edges, DefaultTypes, 10, "s", nil)
  b := SampleEdges(edges, DefaultTypes, 10, "s", nil)
  if len(a) != 10 || len(b) != 10 {
    t.Fatalf("want 10 got %d/%d", len(a), len(b))
  }
  types := map[string]bool{}
  for i := range a {
    if EdgeKey(a[i]) != EdgeKey(b[i]) {
      t.Fatal("sampling is not deterministic")
    }
    if a[i].Type == "imports" {
      t.Fatal("imports must not be sampled")
    }
    types[a[i].Type] = true
  }
  if len(types) != 2 {
    t.Fatalf("round-robin must cover both types: %v", types)
  }
  skip := map[string]bool{EdgeKey(a[0]): true}
  for _, e := range SampleEdges(edges, DefaultTypes, 30, "s", skip) {
    if EdgeKey(e) == EdgeKey(a[0]) {
      t.Fatal("skipped key was sampled again")
    }
  }
}

func TestScoreTask(t *testing.T) {
  md := "# Task context\n\n## Relevant nodes (ranked)\n- fn A()  [1-2]  `function:x/a.go#A`\n- fn B()  [3-4]  `function:Audit_logs/b.go#B`\n\n## Module-scope code (outside any named function)\n- Js/Login.js:12  requests_api -> ANY /x\n\n## Retrieve more\n"
  r := ScoreTask(Task{Name: "t", Must: []string{"a.go#A", "Js/Login.js", "missing.go#Z"}, Noise: []string{"Audit_logs/"}}, md)
  if r.Found != 2 || r.Total != 3 || len(r.Missing) != 1 || r.Missing[0] != "missing.go#Z" {
    t.Fatalf("bad result: %+v", r)
  }
  if len(r.NoiseHits) != 1 || !strings.Contains(r.NoiseHits[0], "Audit_logs") || r.Nodes != 2 {
    t.Fatalf("bad noise/nodes: %+v", r)
  }
}

func TestCompare(t *testing.T) {
  base := Baseline{
    "edges.precision": {Value: 0.9, Dir: "up", Tol: 0.001},
    "tasks.noise":     {Value: 2, Dir: "down", Tol: 0},
    "tasks.tokens":    {Value: 1000, Dir: "down", Tol: 0.15, Rel: true},
    "gone":            {Value: 1, Dir: "up", Tol: 0.001},
  }
  cur := Baseline{
    "edges.precision": {Value: 0.85, Dir: "up", Tol: 0.001},
    "tasks.noise":     {Value: 1, Dir: "down", Tol: 0},
    "tasks.tokens":    {Value: 1100, Dir: "down", Tol: 0.15, Rel: true},
    "new.metric":      {Value: 1, Dir: "up", Tol: 0.001},
  }
  regs, imps, added := Compare(base, cur)
  if len(regs) != 2 {
    t.Fatalf("want precision drop and missing metric, got %v", regs)
  }
  if len(imps) != 1 || !strings.HasPrefix(imps[0], "tasks.noise") {
    t.Fatalf("want noise improvement, got %v", imps)
  }
  if len(added) != 1 {
    t.Fatalf("want one added metric, got %v", added)
  }
}
