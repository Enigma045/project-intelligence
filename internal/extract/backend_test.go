package extract

import (
  "strings"
  "testing"
)

func TestSQLExtract(t *testing.T) {
  src := `CREATE TABLE users (
  id INTEGER PRIMARY KEY,
  email TEXT NOT NULL
);

CREATE TABLE sessions (
  id INTEGER PRIMARY KEY,
  user_id INTEGER NOT NULL REFERENCES users(id),
  FOREIGN KEY (user_id) REFERENCES users(id)
);
CREATE INDEX idx_s ON sessions(user_id);
`
  f := Run("sql", "db/schema.sql", []byte(src))
  u := find(f, "users")
  if u == nil || u.Kind != "table" || u.StartLine != 1 || u.EndLine != 4 {
    t.Fatalf("bad users table: %+v", u)
  }
  s := find(f, "sessions")
  if s == nil || s.Kind != "table" || s.StartLine != 6 || s.EndLine != 10 {
    t.Fatalf("bad sessions table: %+v", s)
  }
  e := find(f, "email")
  if e == nil || e.Kind != "column" || e.Parent != "users" || e.Returns != "TEXT NOT NULL" || e.StartLine != 3 {
    t.Fatalf("bad email column: %+v", e)
  }
  uid := find(f, "user_id")
  if uid == nil || uid.Returns != "INTEGER NOT NULL FK->users" {
    t.Fatalf("bad user_id column: %+v", uid)
  }
  if !hasRef(f, "sql_fk", "users", "user_id") || !hasRef(f, "sql_fk", "users", "") {
    t.Fatalf("missing fk refs: %+v", f.Refs)
  }
  idx := find(f, "idx_s")
  if idx == nil || idx.Kind != "index" || idx.Parent != "sessions" {
    t.Fatalf("bad index: %+v", idx)
  }
}

func TestGoBackend(t *testing.T) {
  src := strings.Join([]string{
    "package api",
    "",
    "import (",
    "  \"database/sql\"",
    "  \"net/http\"",
    ")",
    "",
    "var db *sql.DB",
    "",
    "func Routes(h *Handler) {",
    "  mux := http.NewServeMux()",
    "  mux.HandleFunc(\"POST /api/register\", h.Register)",
    "  mux.HandleFunc(\"/api/users\", ListUsers)",
    "}",
    "",
    "func ListUsers(w http.ResponseWriter, r *http.Request) {",
    "  db.Query(\"SELECT id, email FROM users\")",
    "}",
    "",
    "const schema = \x60CREATE TABLE audit (",
    "  id INTEGER PRIMARY KEY",
    ");\x60",
    "",
  }, "\n")
  f := Run("go", "api/h.go", []byte(src))
  if r := find(f, "POST /api/register"); r == nil || r.Kind != "route" || r.StartLine != 12 {
    t.Fatalf("bad route: %+v", r)
  }
  if r := find(f, "ANY /api/users"); r == nil || r.Kind != "route" {
    t.Fatalf("bad any route: %+v", r)
  }
  if !hasRef(f, "route_handler", "POST /api/register", "h.Register") || !hasRef(f, "route_handler", "ANY /api/users", "ListUsers") {
    t.Fatalf("missing route handler refs: %+v", f.Refs)
  }
  if !hasRef(f, "sql_use", "users", "select") {
    t.Fatalf("missing sql_use: %+v", f.Refs)
  }
  if a := find(f, "audit"); a == nil || a.Kind != "table" || a.StartLine != 20 {
    t.Fatalf("bad embedded table: %+v", a)
  }
}

func TestJSApiMethod(t *testing.T) {
  src := "function a() {\n  fetch(\"/api/register\", { method: \"POST\" });\n}\nfunction b() {\n  fetch(\"/api/users\");\n}\n"
  f := Run("javascript", "a.js", []byte(src))
  if !hasRef(f, "api_request", "/api/register", "POST") || !hasRef(f, "api_request", "/api/users", "") {
    t.Fatalf("bad api_request refs: %+v", f.Refs)
  }
}
