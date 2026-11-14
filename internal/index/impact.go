package index

import (
  "fmt"
  "sort"
  "strings"

  "pidx/internal/model"
  "pidx/internal/store"
)

// ExtractorVersion: bump whenever an extractor changes what it emits; forces a one-time re-extract of all files.
const ExtractorVersion = "13"

type Affected struct{ ID, Reason string }

type SymDiff struct {
  Added       []string `json:"added,omitempty"`
  Removed     []string `json:"removed,omitempty"`
  SigChanged  []string `json:"signature_changed,omitempty"`
  BodyChanged []string `json:"body_changed,omitempty"`
}

func (d SymDiff) Empty() bool {
  return len(d.Added)+len(d.Removed)+len(d.SigChanged)+len(d.BodyChanged) == 0
}

type FileDiff struct {
  File string  `json:"file"`
  Diff SymDiff `json:"diff"`
}

type Impact struct {
  Symbol     string   `json:"symbol"`
  Reason     string   `json:"reason"`
  Dependents []string `json:"dependents"`
}

// DiffSymbols compares stored symbols with freshly extracted ones by stable ID, signature hash and body hash.
func DiffSymbols(old, cur []model.Symbol) SymDiff {
  oldBy := make(map[string]model.Symbol, len(old))
  for _, s := range old {
    oldBy[s.ID] = s
  }
  curBy := make(map[string]bool, len(cur))
  var d SymDiff
  for _, s := range cur {
    curBy[s.ID] = true
    o, ok := oldBy[s.ID]
    switch {
    case !ok:
      d.Added = append(d.Added, s.ID)
    case o.SigHash != s.SigHash:
      d.SigChanged = append(d.SigChanged, s.ID)
    case o.BodyHash != s.BodyHash:
      d.BodyChanged = append(d.BodyChanged, s.ID)
    }
  }
  for _, s := range old {
    if !curBy[s.ID] {
      d.Removed = append(d.Removed, s.ID)
    }
  }
  sort.Strings(d.Added)
  sort.Strings(d.Removed)
  sort.Strings(d.SigChanged)
  sort.Strings(d.BodyChanged)
  return d
}

// idFile extracts the file path from a symbol or file ID (kind:path#name).
func idFile(id string) string {
  i := strings.Index(id, ":")
  if i < 0 {
    return id
  }
  rest := id[i+1:]
  if j := strings.Index(rest, "#"); j >= 0 {
    return rest[:j]
  }
  return rest
}

// ComputeImpact uses the OLD graph (still in the store) to list callers in other files of symbols
// whose signature changed or that were removed. Dependents are reported, not re-parsed.
func ComputeImpact(st *store.Store, aff []Affected) ([]Impact, error) {
  if len(aff) == 0 {
    return nil, nil
  }
  reason := map[string]string{}
  var ids []string
  for _, a := range aff {
    if _, ok := reason[a.ID]; !ok {
      ids = append(ids, a.ID)
    }
    reason[a.ID] = a.Reason
  }
  sort.Strings(ids)
  deps := map[string][]string{}
  for i := 0; i < len(ids); i += 400 {
    j := i + 400
    if j > len(ids) {
      j = len(ids)
    }
    es, err := st.EdgesByDst(ids[i:j], []string{"calls", "listens"})
    if err != nil {
      return nil, err
    }
    for _, e := range es {
      if e.File == idFile(e.Dst) {
        continue
      }
      deps[e.Dst] = append(deps[e.Dst], fmt.Sprintf("%s:%d (%s)", e.File, e.Line, e.Src))
    }
  }
  var out []Impact
  for _, id := range ids {
    if d := deps[id]; len(d) > 0 {
      out = append(out, Impact{Symbol: id, Reason: reason[id], Dependents: d})
    }
  }
  return out, nil
}
