package query

import (
  "sort"
  "strings"
  "unicode"

  "pidx/internal/model"
)

type Scored struct {
  Sym   model.Symbol
  Score float64
}

var stop = map[string]bool{}

func init() {
  for _, w := range strings.Fields("the an to of for in on and or with that this is it be as at by from into before after when then do does should will can add modify change update make create fix implement need want so we my our your user code let set get new just only via per each please also all any use using how what") {
    stop[w] = true
  }
}

var kindWeight = map[string]float64{
  "function": 1, "method": 1, "class": 1, "struct": 1, "interface": 1, "type": 0.9, "form": 1.1,
  "element": 1, "field": 0.9, "css_rule": 0.6, "keyframes": 0.5, "css_var": 0.4, "const": 0.4, "var": 0.4,
  "table": 1.1, "route": 1.1, "column": 0.8, "view": 0.9,
}

// Tokens splits identifiers and text into lowercase words (camelCase, snake_case, paths, selectors).
func Tokens(s string) []string {
  var out []string
  var cur []rune
  flush := func() {
    if len(cur) > 1 {
      out = append(out, strings.ToLower(string(cur)))
    }
    cur = cur[:0]
  }
  rs := []rune(s)
  for i, r := range rs {
    if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
      flush()
      continue
    }
    if i > 0 && len(cur) > 0 {
      p := rs[i-1]
      if unicode.IsUpper(r) && unicode.IsLower(p) {
        flush()
      } else if unicode.IsUpper(r) && unicode.IsUpper(p) && i+1 < len(rs) && unicode.IsLower(rs[i+1]) {
        flush()
      }
    }
    cur = append(cur, r)
  }
  flush()
  return out
}

// TaskTokens tokenizes a task description, dropping stop words and duplicates.
func TaskTokens(task string) []string {
  seen := map[string]bool{}
  var out []string
  for _, t := range Tokens(task) {
    if stop[t] || seen[t] {
      continue
    }
    seen[t] = true
    out = append(out, t)
  }
  return out
}

// prefixMatch is a cheap stemmer: register~registration, submit~submission.
func prefixMatch(a, b string) bool {
  n := 0
  for n < len(a) && n < len(b) && a[n] == b[n] {
    n++
  }
  m := len(a)
  if len(b) < m {
    m = len(b)
  }
  return n >= 5 && n*10 >= 6*m
}

// matchStrength: 3 = exact name token, 2 = stem match, 0 = none.
func matchStrength(t string, names []string, nameSet map[string]bool) float64 {
  if nameSet[t] {
    return 3
  }
  for _, n := range names {
    if prefixMatch(t, n) {
      return 2
    }
  }
  return 0
}

func nameTokens(s model.Symbol) ([]string, map[string]bool) {
  names := Tokens(s.Qualified())
  set := make(map[string]bool, len(names))
  for _, n := range names {
    set[n] = true
  }
  return names, set
}

// tokenWeights down-weights words that match many symbols: full weight up to 2 matches, then
// 1/(1+0.25*(df-2)). A word that names one or two things counts fully; a word shared by a handful
// of symbols ("users", "store") counts about half or less and cannot seed on its own.
func tokenWeights(task []string, syms []model.Symbol) map[string]float64 {
  df := make(map[string]int, len(task))
  for _, s := range syms {
    names, set := nameTokens(s)
    for _, t := range task {
      if matchStrength(t, names, set) > 0 {
        df[t]++
      }
    }
  }
  w := make(map[string]float64, len(task))
  for _, t := range task {
    if d := df[t]; d > 2 {
      w[t] = 1 / (1 + 0.25*float64(d-2))
    } else {
      w[t] = 1
    }
  }
  return w
}

func scoreSym(task []string, s model.Symbol, w map[string]float64) float64 {
  names, nameSet := nameTokens(s)
  pathSet := map[string]bool{}
  for _, p := range Tokens(s.File) {
    pathSet[p] = true
  }
  total := 0.0
  nameHit := false
  for _, t := range task {
    if m := matchStrength(t, names, nameSet); m > 0 {
      total += m * w[t]
      nameHit = true
    } else if pathSet[t] {
      total += 1
    }
  }
  if !nameHit {
    return 0
  }
  kw, ok := kindWeight[s.Kind]
  if !ok {
    kw = 0.7
  }
  if strings.HasSuffix(s.File, "_test.go") {
    kw *= 0.4
  }
  return total * kw
}

// Seeds returns up to k best-matching symbols for a task, best first. A seed must score at least 1.0
// and at least half of the best score, so a weak shared word cannot seed next to a strong match.
func Seeds(task string, syms []model.Symbol, k int) []Scored {
  tt := TaskTokens(task)
  if len(tt) == 0 {
    return nil
  }
  w := tokenWeights(tt, syms)
  var all []Scored
  top := 0.0
  for _, s := range syms {
    sc := scoreSym(tt, s, w)
    if sc < 1 {
      continue
    }
    all = append(all, Scored{s, sc})
    if sc > top {
      top = sc
    }
  }
  var out []Scored
  for _, o := range all {
    if o.Score >= 0.5*top {
      out = append(out, o)
    }
  }
  sort.Slice(out, func(i, j int) bool {
    if out[i].Score != out[j].Score {
      return out[i].Score > out[j].Score
    }
    return out[i].Sym.ID < out[j].Sym.ID
  })
  if len(out) > k {
    out = out[:k]
  }
  return out
}
