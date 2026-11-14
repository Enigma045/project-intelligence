package api

import (
  "database/sql"
  "net/http"
)

type Handler struct{ DB *sql.DB }

var db *sql.DB

func Routes(h *Handler) *http.ServeMux {
  mux := http.NewServeMux()
  mux.HandleFunc("POST /api/register", h.Register)
  mux.HandleFunc("GET /api/users", ListUsers)
  return mux
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
  email := r.FormValue("email")
  if _, err := h.DB.Exec("INSERT INTO users (email) VALUES (?)", email); err != nil {
    http.Error(w, "failed", http.StatusInternalServerError)
    return
  }
  w.WriteHeader(http.StatusCreated)
}

func ListUsers(w http.ResponseWriter, r *http.Request) {
  rows, err := db.Query("SELECT id, email FROM users ORDER BY id")
  if err != nil {
    http.Error(w, "failed", http.StatusInternalServerError)
    return
  }
  defer rows.Close()
  w.WriteHeader(http.StatusOK)
}
