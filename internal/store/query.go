package store

import "pidx/internal/model"

// SymbolsByIDs loads symbols for the given IDs (any order).
func (s *Store) SymbolsByIDs(ids []string) ([]model.Symbol, error) {
  var out []model.Symbol
  for i := 0; i < len(ids); i += 400 {
    j := i + 400
    if j > len(ids) {
      j = len(ids)
    }
    chunk := ids[i:j]
    args := make([]any, len(chunk))
    for k, id := range chunk {
      args[k] = id
    }
    rows, err := s.db.Query("SELECT "+symCols+" FROM symbols WHERE id IN ("+placeholders(len(chunk))+")", args...)
    if err != nil {
      return nil, err
    }
    got, err := scanSymbols(rows)
    if err != nil {
      return nil, err
    }
    out = append(out, got...)
  }
  return out, nil
}

// EdgesByType returns every edge whose type is in types.
func (s *Store) EdgesByType(types []string) ([]model.Edge, error) {
  if len(types) == 0 {
    return nil, nil
  }
  if err := s.ensureEdges(); err != nil {
    return nil, err
  }
  args := make([]any, len(types))
  for i, t := range types {
    args[i] = t
  }
  rows, err := s.db.Query("SELECT "+edgeCols+" FROM edges WHERE type IN ("+placeholders(len(types))+") ORDER BY file, line, id", args...)
  if err != nil {
    return nil, err
  }
  return scanEdges(rows)
}
