package query

import "pidx/internal/model"

// Explain exposes the seed scorer for diagnostics: the task words, their weights, and every symbol's raw name score.
func Explain(task string, syms []model.Symbol) ([]string, map[string]float64, map[string]float64) {
  tt := TaskTokens(task)
  w := tokenWeights(tt, syms)
  raw := make(map[string]float64)
  for _, s := range syms {
    if v := scoreSym(tt, s, w); v > 0 {
      raw[s.ID] = v
    }
  }
  return tt, w, raw
}
