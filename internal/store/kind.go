package store

import "pidx/internal/model"

// SymbolsByKinds returns every symbol whose kind is in kinds, ordered by file and line.
func (s *Store) SymbolsByKinds(kinds []string) ([]model.Symbol, error) {
  if len(kinds) == 0 {
    return nil, nil
  }
  args := make([]any, len(kinds))
  for i, k := range kinds {
    args[i] = k
  }
  rows, err := s.db.Query("SELECT "+symCols+" FROM symbols WHERE kind IN ("+placeholders(len(kinds))+") ORDER BY file, start_line", args...)
  if err != nil {
    return nil, err
  }
  return scanSymbols(rows)
}
