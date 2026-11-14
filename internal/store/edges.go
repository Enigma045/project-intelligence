package store

import (
  "database/sql"
  "encoding/json"
  "strings"

  "pidx/internal/model"
)

const edgeSchema = `
CREATE TABLE IF NOT EXISTS edges (
  id INTEGER PRIMARY KEY AUTOINCREMENT, src TEXT, dst TEXT, dst_name TEXT, type TEXT,
  status TEXT, confidence REAL, line INTEGER, file TEXT, evidence TEXT, cnt INTEGER);
CREATE INDEX IF NOT EXISTS idx_edge_src ON edges(src, type);
CREATE INDEX IF NOT EXISTS idx_edge_dst ON edges(dst, type);
CREATE INDEX IF NOT EXISTS idx_edge_status ON edges(status);
`

const edgeCols = "src,dst,dst_name,type,status,confidence,line,file,evidence,cnt"

func (s *Store) ensureEdges() error {
  _, err := s.db.Exec(edgeSchema)
  return err
}

func placeholders(n int) string {
  return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}

func (s *Store) LoadFileRows() ([]model.FileRow, error) {
  rows, err := s.db.Query("SELECT path, language, imports FROM files ORDER BY path")
  if err != nil {
    return nil, err
  }
  defer rows.Close()
  var out []model.FileRow
  for rows.Next() {
    var fr model.FileRow
    var imps string
    if err := rows.Scan(&fr.Path, &fr.Language, &imps); err != nil {
      return nil, err
    }
    _ = json.Unmarshal([]byte(imps), &fr.Imports)
    out = append(out, fr)
  }
  return out, rows.Err()
}

func (s *Store) LoadSymbols() ([]model.Symbol, error) {
  rows, err := s.db.Query("SELECT " + symCols + " FROM symbols ORDER BY file, start_line")
  if err != nil {
    return nil, err
  }
  return scanSymbols(rows)
}

func (s *Store) LoadRefs() ([]model.RawRef, error) {
  rows, err := s.db.Query("SELECT file,kind,name,target,line,from_sym FROM raw_refs ORDER BY file, line, id")
  if err != nil {
    return nil, err
  }
  defer rows.Close()
  var out []model.RawRef
  for rows.Next() {
    var r model.RawRef
    if err := rows.Scan(&r.File, &r.Kind, &r.Name, &r.Target, &r.Line, &r.From); err != nil {
      return nil, err
    }
    out = append(out, r)
  }
  return out, rows.Err()
}

func (s *Store) ReplaceEdges(edges []model.Edge) error {
  if err := s.ensureEdges(); err != nil {
    return err
  }
  tx, err := s.db.Begin()
  if err != nil {
    return err
  }
  defer tx.Rollback()
  if _, err := tx.Exec("DELETE FROM edges"); err != nil {
    return err
  }
  stmt, err := tx.Prepare("INSERT INTO edges(" + edgeCols + ") VALUES(?,?,?,?,?,?,?,?,?,?)")
  if err != nil {
    return err
  }
  defer stmt.Close()
  for _, e := range edges {
    if _, err := stmt.Exec(e.Src, e.Dst, e.DstName, e.Type, e.Status, e.Confidence, e.Line, e.File, e.Evidence, e.Count); err != nil {
      return err
    }
  }
  return tx.Commit()
}

func (s *Store) EdgeCount() (int, error) {
  if err := s.ensureEdges(); err != nil {
    return 0, err
  }
  var n int
  err := s.db.QueryRow("SELECT COUNT(*) FROM edges").Scan(&n)
  return n, err
}

func scanEdges(rows *sql.Rows) ([]model.Edge, error) {
  defer rows.Close()
  var out []model.Edge
  for rows.Next() {
    var e model.Edge
    if err := rows.Scan(&e.Src, &e.Dst, &e.DstName, &e.Type, &e.Status, &e.Confidence, &e.Line, &e.File, &e.Evidence, &e.Count); err != nil {
      return nil, err
    }
    out = append(out, e)
  }
  return out, rows.Err()
}

func (s *Store) edgesWhere(col string, ids, types []string) ([]model.Edge, error) {
  if len(ids) == 0 {
    return nil, nil
  }
  if err := s.ensureEdges(); err != nil {
    return nil, err
  }
  q := "SELECT " + edgeCols + " FROM edges WHERE " + col + " IN (" + placeholders(len(ids)) + ")"
  args := make([]any, 0, len(ids)+len(types))
  for _, id := range ids {
    args = append(args, id)
  }
  if len(types) > 0 {
    q += " AND type IN (" + placeholders(len(types)) + ")"
    for _, t := range types {
      args = append(args, t)
    }
  }
  q += " ORDER BY file, line"
  rows, err := s.db.Query(q, args...)
  if err != nil {
    return nil, err
  }
  return scanEdges(rows)
}

func (s *Store) EdgesBySrc(ids, types []string) ([]model.Edge, error) {
  return s.edgesWhere("src", ids, types)
}

func (s *Store) EdgesByDst(ids, types []string) ([]model.Edge, error) {
  return s.edgesWhere("dst", ids, types)
}

func (s *Store) EdgesByStatus(statuses []string, limit int) ([]model.Edge, error) {
  if err := s.ensureEdges(); err != nil {
    return nil, err
  }
  args := make([]any, 0, len(statuses)+1)
  for _, st := range statuses {
    args = append(args, st)
  }
  args = append(args, limit)
  rows, err := s.db.Query("SELECT "+edgeCols+" FROM edges WHERE status IN ("+placeholders(len(statuses))+") ORDER BY file, line LIMIT ?", args...)
  if err != nil {
    return nil, err
  }
  return scanEdges(rows)
}

func (s *Store) EdgeStats() (map[string]int, error) {
  if err := s.ensureEdges(); err != nil {
    return nil, err
  }
  rows, err := s.db.Query("SELECT type, status, COUNT(*) FROM edges GROUP BY type, status")
  if err != nil {
    return nil, err
  }
  defer rows.Close()
  out := map[string]int{}
  for rows.Next() {
    var t, st string
    var n int
    if err := rows.Scan(&t, &st, &n); err != nil {
      return nil, err
    }
    out[t+"/"+st] = n
  }
  return out, rows.Err()
}

// FindSymbols matches an exact symbol ID, name, or Parent.Name.
func (s *Store) FindSymbols(q string, limit int) ([]model.Symbol, error) {
  rows, err := s.db.Query("SELECT "+symCols+" FROM symbols WHERE id=? OR name=? OR (parent<>'' AND parent||'.'||name=?) ORDER BY file, start_line LIMIT ?", q, q, q, limit)
  if err != nil {
    return nil, err
  }
  return scanSymbols(rows)
}
