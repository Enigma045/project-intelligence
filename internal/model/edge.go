package model

// Edge is a directed relationship between two nodes.
// Status values: resolved, inferred, ambiguous, unresolved, external.
// Type values: calls, listens, imports.
type Edge struct {
  Src        string  `json:"src"`
  Dst        string  `json:"dst,omitempty"`
  DstName    string  `json:"dst_name"`
  Type       string  `json:"type"`
  Status     string  `json:"status"`
  Confidence float64 `json:"confidence"`
  Line       int     `json:"line,omitempty"`
  File       string  `json:"file,omitempty"`
  Evidence   string  `json:"evidence,omitempty"`
  Count      int     `json:"count"`
}

// FileRow is the minimal per-file data the resolver needs.
type FileRow struct {
  Path     string
  Language string
  Imports  []string
}
