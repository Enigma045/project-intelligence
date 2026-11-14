package eval

import (
  "fmt"
  "sort"
)

// Metric is one tracked number. Dir is "up" (higher is better) or "down". Tol is the allowed movement
// (relative to the baseline value when Rel is set) before a change counts as a regression.
type Metric struct {
  Value float64 `json:"value"`
  Dir   string  `json:"dir"`
  Tol   float64 `json:"tol"`
  Rel   bool    `json:"rel,omitempty"`
}

type Baseline map[string]Metric

func Merge(a, b Baseline) Baseline {
  out := Baseline{}
  for k, v := range a {
    out[k] = v
  }
  for k, v := range b {
    out[k] = v
  }
  return out
}

func rate(m Baseline, k string, v float64) {
  m[k] = Metric{Value: v, Dir: "up", Tol: 0.001}
}

// EdgeMetrics turns an edge report into gate metrics (precision and recall overall and per language).
func EdgeMetrics(r Report) Baseline {
  m := Baseline{}
  if p, n := r.All.Precision(); n > 0 {
    rate(m, "edges.precision", p)
  }
  if v, n := r.All.Recall(); n > 0 {
    rate(m, "edges.recall", v)
  }
  for lang, c := range r.ByLang {
    if p, n := c.Precision(); n > 0 {
      rate(m, "edges.precision."+lang, p)
    }
    if v, n := c.Recall(); n > 0 {
      rate(m, "edges.recall."+lang, v)
    }
  }
  return m
}

// TaskMetrics turns task results into gate metrics: recall (overall and per task), noise hits, tokens.
func TaskMetrics(rs []TaskResult) Baseline {
  m := Baseline{}
  if len(rs) == 0 {
    return m
  }
  var rec, tok float64
  noise := 0
  for _, r := range rs {
    rate(m, "task."+r.Name+".recall", r.Recall)
    rec += r.Recall
    tok += float64(r.Tokens)
    noise += len(r.NoiseHits)
  }
  n := float64(len(rs))
  rate(m, "tasks.recall", rec/n)
  m["tasks.noise"] = Metric{Value: float64(noise), Dir: "down", Tol: 0}
  m["tasks.tokens"] = Metric{Value: tok / n, Dir: "down", Tol: 0.15, Rel: true}
  return m
}

// Compare reports metrics that got worse (regressions), better (improvements) or are new (added).
func Compare(base, cur Baseline) (regs, imps, added []string) {
  const eps = 1e-9
  keys := make([]string, 0, len(base))
  for k := range base {
    keys = append(keys, k)
  }
  sort.Strings(keys)
  for _, k := range keys {
    b := base[k]
    c, ok := cur[k]
    if !ok {
      regs = append(regs, fmt.Sprintf("%s: metric missing (baseline %.3f)", k, b.Value))
      continue
    }
    msg := fmt.Sprintf("%s: %.3f -> %.3f", k, b.Value, c.Value)
    if b.Dir == "down" {
      lim := b.Tol
      if b.Rel {
        lim = b.Value * b.Tol
      }
      if c.Value > b.Value+lim+eps {
        regs = append(regs, msg)
      } else if c.Value < b.Value-lim-eps {
        imps = append(imps, msg)
      }
      continue
    }
    if c.Value < b.Value-b.Tol-eps {
      regs = append(regs, msg)
    } else if c.Value > b.Value+b.Tol+eps {
      imps = append(imps, msg)
    }
  }
  var nk []string
  for k := range cur {
    if _, ok := base[k]; !ok {
      nk = append(nk, k)
    }
  }
  sort.Strings(nk)
  for _, k := range nk {
    added = append(added, fmt.Sprintf("%s: %.3f (not in baseline)", k, cur[k].Value))
  }
  return
}
