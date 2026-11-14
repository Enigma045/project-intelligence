package model

import (
  "crypto/sha256"
  "encoding/hex"
)

type Param struct {
  Name       string `json:"name"`
  Type       string `json:"type,omitempty"`
  TypeSource string `json:"type_source"` // declared | unknown
}

type Symbol struct {
  ID            string  `json:"id"`
  Kind          string  `json:"kind"`
  Name          string  `json:"name"`
  Parent        string  `json:"parent,omitempty"`
  File          string  `json:"file"`
  StartLine     int     `json:"start_line"`
  EndLine       int     `json:"end_line"`
  Params        []Param `json:"params,omitempty"`
  Returns       string  `json:"returns,omitempty"`
  ReturnsSource string  `json:"returns_source,omitempty"`
  Visibility    string  `json:"visibility"`
  Async         bool    `json:"async,omitempty"`
  BodyHash      string  `json:"body_hash"`
  SigHash       string  `json:"signature_hash"`
  Provenance    string  `json:"provenance"` // verified | inferred
}

func (s Symbol) Qualified() string {
  if s.Parent != "" {
    return s.Parent + "." + s.Name
  }
  return s.Name
}

// RawRef is an unresolved reference emitted by an extractor (resolved in Phase 2).
type RawRef struct {
  File   string `json:"file,omitempty"`
  Kind   string `json:"kind"` // call | dom_id | dom_selector | event | api_request
  Name   string `json:"name"`
  Target string `json:"target,omitempty"`
  Line   int    `json:"line"`
  From   string `json:"from,omitempty"` // enclosing symbol, empty = module scope
}

type FileFacts struct {
  Path        string   `json:"path"`
  Language    string   `json:"language"`
  ParseStatus string   `json:"parse_status"` // ok | partial | error | heuristic | unsupported
  ParseErrors []string `json:"parse_errors,omitempty"`
  Imports     []string `json:"imports,omitempty"`
  Symbols     []Symbol `json:"symbols"`
  Refs        []RawRef `json:"refs"`
}

type FileInfo struct {
  Path     string
  Name     string
  Ext      string
  Language string
  Size     int64
  MTimeNS  int64
  Hash     string
}

func HashBytes(b []byte) string {
  s := sha256.Sum256(b)
  return "sha256:" + hex.EncodeToString(s[:])
}

func HashString(s string) string { return HashBytes([]byte(s)) }
