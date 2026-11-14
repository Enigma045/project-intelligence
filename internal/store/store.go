package store

import (
  "database/sql"
  "encoding/json"
  "os"
  "path/filepath"

  _ "modernc.org/sqlite"

  "pidx/internal/model"
)

const schema = `
CREATE TABLE IF NOT EXISTS files (
  path TEXT PRIMARY KEY, name TEXT, ext TEXT, language TEXT, size INTEGER,
  mtime_ns INTEGER, hash TEXT, parse_status TEXT, parse_errors TEXT, imports TEXT);
CREATE TABLE IF NOT EXISTS symbols (
  id TEXT PRIMARY KEY, file TEXT, kind TEXT, name TEXT, parent TEXT,
  start_line INTEGER, end_line INTEGER, params TEXT, returns TEXT, returns_source TEXT,
  visibility TEXT, async INTEGER, body_hash TEXT, sig_hash TEXT, provenance TEXT);
CREATE INDEX IF NOT EXISTS idx_sym_file ON symbols(file);
CREATE INDEX IF NOT EXISTS idx_sym_name ON symbols(name);
CREATE TABLE IF NOT EXISTS raw_refs (
  id INTEGER PRIMARY KEY AUTOINCREMENT, file TEXT, kind TEXT, name TEXT,
  target TEXT, line INTEGER, from_sym TEXT);
CREATE INDEX IF NOT EXISTS idx_ref_file ON raw_refs(file);
CREATE INDEX IF NOT EXISTS idx_ref_name ON raw_refs(name);
CREATE TABLE IF NOT EXISTS meta (k TEXT PRIMARY KEY, v TEXT);
`

const symCols = "id,file,kind,name,parent,start_line,end_line,params,returns,returns_source,visibility,async,body_hash,sig_hash,provenance"

type Store struct{ db *sql.DB }

type Stats struct {
  Files      int
  Symbols    int
  Refs       int
  ByLanguage map[string]int
}

func Open(dbPath string) (*Store, error) {
  if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
    return nil, err
  }
  db, err := sql.Open("sqlite", dbPath)
  if err != nil {
    return nil, err
  }
  db.SetMaxOpenConns(1)
  if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
    db.Close()
    return nil, err
  }
  if _, err := db.Exec(schema); err != nil {
    db.Close()
    return nil, err
  }
  return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) ListFiles() (map[string]model.FileInfo, error) {
  rows, err := s.db.Query("SELECT path,name,ext,language,size,mtime_ns,hash FROM files")
  if err != nil {
    return nil, err
  }
  defer rows.Close()
  out := map[string]model.FileInfo{}
  for rows.Next() {
    var fi model.FileInfo
    if err := rows.Scan(&fi.Path, &fi.Name, &fi.Ext, &fi.Language, &fi.Size, &fi.MTimeNS, &fi.Hash); err != nil {
      return nil, err
    }
    out[fi.Path] = fi
  }
  return out, rows.Err()
}

func (s *Store) GetFile(path string) (model.FileInfo, string, []string, bool, error) {
  var fi model.FileInfo
  var status, imps string
  err := s.db.QueryRow("SELECT path,name,ext,language,size,mtime_ns,hash,parse_status,imports FROM files WHERE path=?", path).
    Scan(&fi.Path, &fi.Name, &fi.Ext, &fi.Language, &fi.Size, &fi.MTimeNS, &fi.Hash, &status, &imps)
  if err == sql.ErrNoRows {
    return fi, "", nil, false, nil
  }
  if err != nil {
    return fi, "", nil, false, err
  }
  var im []string
  _ = json.Unmarshal([]byte(imps), &im)
  return fi, status, im, true, nil
}

func (s *Store) ReplaceFile(info model.FileInfo, f model.FileFacts) error {
  tx, err := s.db.Begin()
  if err != nil {
    return err
  }
  defer tx.Rollback()
  for _, q := range []string{"DELETE FROM symbols WHERE file=?", "DELETE FROM raw_refs WHERE file=?"} {
    if _, err := tx.Exec(q, info.Path); err != nil {
      return err
    }
  }
  imps, _ := json.Marshal(f.Imports)
  errs, _ := json.Marshal(f.ParseErrors)
  if _, err := tx.Exec(`INSERT OR REPLACE INTO files(path,name,ext,language,size,mtime_ns,hash,parse_status,parse_errors,imports) VALUES(?,?,?,?,?,?,?,?,?,?)`,
    info.Path, info.Name, info.Ext, info.Language, info.Size, info.MTimeNS, info.Hash, f.ParseStatus, string(errs), string(imps)); err != nil {
    return err
  }
  for _, sy := range f.Symbols {
    ps, _ := json.Marshal(sy.Params)
    async := 0
    if sy.Async {
      async = 1
    }
    if _, err := tx.Exec(`INSERT OR REPLACE INTO symbols(`+symCols+`) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
      sy.ID, sy.File, sy.Kind, sy.Name, sy.Parent, sy.StartLine, sy.EndLine, string(ps), sy.Returns, sy.ReturnsSource,
      sy.Visibility, async, sy.BodyHash, sy.SigHash, sy.Provenance); err != nil {
      return err
    }
  }
  for _, r := range f.Refs {
    if _, err := tx.Exec(`INSERT INTO raw_refs(file,kind,name,target,line,from_sym) VALUES(?,?,?,?,?,?)`,
      info.Path, r.Kind, r.Name, r.Target, r.Line, r.From); err != nil {
      return err
    }
  }
  return tx.Commit()
}

func (s *Store) TouchFile(path string, mtimeNS, size int64) error {
  _, err := s.db.Exec("UPDATE files SET mtime_ns=?, size=? WHERE path=?", mtimeNS, size, path)
  return err
}

func (s *Store) DeleteFile(path string) error {
  tx, err := s.db.Begin()
  if err != nil {
    return err
  }
  defer tx.Rollback()
  for _, q := range []string{"DELETE FROM symbols WHERE file=?", "DELETE FROM raw_refs WHERE file=?", "DELETE FROM files WHERE path=?"} {
    if _, err := tx.Exec(q, path); err != nil {
      return err
    }
  }
  return tx.Commit()
}

func scanSymbols(rows *sql.Rows) ([]model.Symbol, error) {
  defer rows.Close()
  var out []model.Symbol
  for rows.Next() {
    var sy model.Symbol
    var ps string
    var async int
    if err := rows.Scan(&sy.ID, &sy.File, &sy.Kind, &sy.Name, &sy.Parent, &sy.StartLine, &sy.EndLine, &ps, &sy.Returns,
      &sy.ReturnsSource, &sy.Visibility, &async, &sy.BodyHash, &sy.SigHash, &sy.Provenance); err != nil {
      return nil, err
    }
    sy.Async = async == 1
    _ = json.Unmarshal([]byte(ps), &sy.Params)
    out = append(out, sy)
  }
  return out, rows.Err()
}

func (s *Store) Outline(path string) ([]model.Symbol, error) {
  rows, err := s.db.Query("SELECT "+symCols+" FROM symbols WHERE file=? ORDER BY start_line, end_line DESC", path)
  if err != nil {
    return nil, err
  }
  return scanSymbols(rows)
}

func (s *Store) Search(q string, limit int) ([]model.Symbol, error) {
  like := "%" + q + "%"
  rows, err := s.db.Query("SELECT "+symCols+" FROM symbols WHERE name LIKE ? OR (parent || '.' || name) LIKE ? ORDER BY name, file LIMIT ?", like, like, limit)
  if err != nil {
    return nil, err
  }
  return scanSymbols(rows)
}

func (s *Store) FindRefs(name string) ([]model.RawRef, error) {
  rows, err := s.db.Query("SELECT file,kind,name,target,line,from_sym FROM raw_refs WHERE name=? OR name LIKE ? OR target=? ORDER BY file, line", name, "%."+name, name)
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

func (s *Store) SetMeta(k, v string) error {
  _, err := s.db.Exec("INSERT OR REPLACE INTO meta(k,v) VALUES(?,?)", k, v)
  return err
}

func (s *Store) Stats() (Stats, error) {
  st := Stats{ByLanguage: map[string]int{}}
  if err := s.db.QueryRow("SELECT COUNT(*) FROM files").Scan(&st.Files); err != nil {
    return st, err
  }
  if err := s.db.QueryRow("SELECT COUNT(*) FROM symbols").Scan(&st.Symbols); err != nil {
    return st, err
  }
  if err := s.db.QueryRow("SELECT COUNT(*) FROM raw_refs").Scan(&st.Refs); err != nil {
    return st, err
  }
  rows, err := s.db.Query("SELECT language, COUNT(*) FROM files GROUP BY language")
  if err != nil {
    return st, err
  }
  defer rows.Close()
  for rows.Next() {
    var l string
    var n int
    if err := rows.Scan(&l, &n); err != nil {
      return st, err
    }
    st.ByLanguage[l] = n
  }
  return st, rows.Err()
}
