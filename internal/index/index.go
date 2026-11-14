package index

import (
  "os"
  "path/filepath"
  "runtime"
  "sync"
  "time"

  "pidx/internal/extract"
  "pidx/internal/model"
  "pidx/internal/scan"
  "pidx/internal/store"
)

type Report struct {
  Added, Changed, Deleted, Unchanged, Touched, Failed int
  Elapsed                                             time.Duration
  Impact                                              []Impact
  Diffs                                               []FileDiff
}

type job struct {
  e     scan.Entry
  old   *model.FileInfo
  force bool
}

type result struct {
  kind  string
  info  model.FileInfo
  facts model.FileFacts
  err   error
}

func process(j job) result {
  src, err := os.ReadFile(j.e.Abs)
  if err != nil {
    return result{err: err}
  }
  h := model.HashBytes(src)
  info := model.FileInfo{
    Path: j.e.Rel, Name: filepath.Base(j.e.Rel), Ext: filepath.Ext(j.e.Rel), Language: j.e.Language,
    Size: int64(len(src)), MTimeNS: j.e.MTimeNS, Hash: h,
  }
  if j.old != nil && !j.force && j.old.Hash == h {
    return result{kind: "touch", info: info}
  }
  kind := "added"
  if j.old != nil {
    kind = "changed"
  }
  return result{kind: kind, info: info, facts: extract.Run(j.e.Language, j.e.Rel, src)}
}

// Run performs an incremental scan: only files whose (mtime,size) differ are read and hashed;
// only files whose hash differs are re-extracted. A changed ExtractorVersion forces a full re-extract once.
func Run(root string, st *store.Store) (Report, error) {
  start := time.Now()
  var rep Report
  entries, err := scan.Walk(root)
  if err != nil {
    return rep, err
  }
  stored, err := st.ListFiles()
  if err != nil {
    return rep, err
  }
  force := st.GetMeta("extractor_version") != ExtractorVersion
  var affected []Affected
  seen := make(map[string]bool, len(entries))
  var jobs []job
  for _, e := range entries {
    seen[e.Rel] = true
    old, ok := stored[e.Rel]
    if ok && !force && old.MTimeNS == e.MTimeNS && old.Size == e.Size {
      rep.Unchanged++
      continue
    }
    j := job{e: e, force: force}
    if ok {
      o := old
      j.old = &o
    }
    jobs = append(jobs, j)
  }

  results := make([]result, len(jobs))
  ch := make(chan int)
  var wg sync.WaitGroup
  for w := 0; w < runtime.NumCPU(); w++ {
    wg.Add(1)
    go func() {
      defer wg.Done()
      for idx := range ch {
        results[idx] = process(jobs[idx])
      }
    }()
  }
  for i := range jobs {
    ch <- i
  }
  close(ch)
  wg.Wait()

  for _, r := range results {
    switch {
    case r.err != nil:
      rep.Failed++
    case r.kind == "touch":
      if err := st.TouchFile(r.info.Path, r.info.MTimeNS, r.info.Size); err != nil {
        return rep, err
      }
      rep.Touched++
    default:
      if r.kind == "changed" {
        if oldSyms, oerr := st.Outline(r.info.Path); oerr == nil {
          df := DiffSymbols(oldSyms, r.facts.Symbols)
          if !df.Empty() {
            rep.Diffs = append(rep.Diffs, FileDiff{File: r.info.Path, Diff: df})
          }
          for _, id := range df.SigChanged {
            affected = append(affected, Affected{id, "signature_changed"})
          }
          for _, id := range df.Removed {
            affected = append(affected, Affected{id, "removed"})
          }
        }
      }
      if err := st.ReplaceFile(r.info, r.facts); err != nil {
        return rep, err
      }
      if r.kind == "added" {
        rep.Added++
      } else {
        rep.Changed++
      }
    }
  }
  for p := range stored {
    if !seen[p] {
      if oldSyms, oerr := st.Outline(p); oerr == nil {
        for _, s := range oldSyms {
          affected = append(affected, Affected{s.ID, "removed"})
        }
      }
      if err := st.DeleteFile(p); err != nil {
        return rep, err
      }
      rep.Deleted++
    }
  }

  // Impact uses the OLD graph, which is still in the store until RebuildGraph below.
  if imp, ierr := ComputeImpact(st, affected); ierr == nil {
    rep.Impact = imp
  }
  need := force || rep.Added+rep.Changed+rep.Deleted > 0
  if !need {
    if n, cerr := st.EdgeCount(); cerr == nil && n == 0 {
      need = true
    }
  }
  if need {
    if _, err := RebuildGraph(st); err != nil {
      return rep, err
    }
  }
  _ = st.SetMeta("extractor_version", ExtractorVersion)
  _ = st.SetMeta("last_scan", time.Now().UTC().Format(time.RFC3339))
  rep.Elapsed = time.Since(start)
  return rep, nil
}
