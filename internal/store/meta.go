package store

// GetMeta returns the stored value for k, or "" if absent.
func (s *Store) GetMeta(k string) string {
  var v string
  if err := s.db.QueryRow("SELECT v FROM meta WHERE k=?", k).Scan(&v); err != nil {
    return ""
  }
  return v
}
