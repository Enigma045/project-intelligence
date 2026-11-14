package index

import (
  "strconv"

  "pidx/internal/resolve"
  "pidx/internal/store"
)

// RebuildGraph re-resolves all references into edges: code edges (calls/imports/listens), web edges
// (HTML/CSS/JS links) and backend edges (API routes, SQL tables). Resolution is global but cheap relative
// to parsing (no file I/O); per-file parsing remains incremental.
func RebuildGraph(st *store.Store) (int, error) {
  files, err := st.LoadFileRows()
  if err != nil {
    return 0, err
  }
  syms, err := st.LoadSymbols()
  if err != nil {
    return 0, err
  }
  refs, err := st.LoadRefs()
  if err != nil {
    return 0, err
  }
  in := resolve.Input{Files: files, Symbols: syms, Refs: refs}
  edges := resolve.Resolve(in)
  edges = append(edges, resolve.ResolveWeb(in)...)
  edges = append(edges, resolve.ResolveBackend(in)...)
  if err := st.ReplaceEdges(edges); err != nil {
    return 0, err
  }
  _ = st.SetMeta("edges", strconv.Itoa(len(edges)))
  return len(edges), nil
}
