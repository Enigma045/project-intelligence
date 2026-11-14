package store

// HubCallers returns, for every symbol called from at least min distinct callers (resolved or inferred),
// the number of distinct callers.
func (s *Store) HubCallers(min int) (map[string]int, error) {
  if err := s.ensureEdges(); err != nil {
    return nil, err
  }
  rows, err := s.db.Query("SELECT dst, COUNT(DISTINCT src) FROM edges WHERE type='calls' AND dst<>'' AND status IN ('resolved','inferred') GROUP BY dst HAVING COUNT(DISTINCT src) >= ?", min)
  if err != nil {
    return nil, err
  }
  defer rows.Close()
  out := map[string]int{}
  for rows.Next() {
    var id string
    var n int
    if err := rows.Scan(&id, &n); err != nil {
      return nil, err
    }
    out[id] = n
  }
  return out, rows.Err()
}
