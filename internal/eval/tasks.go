package eval

import "strings"

// Task describes one realistic request. Must lists substrings that have to appear in the context: in a
// node id (for example "Routes/Login2.go#Login") or in the module-scope section (a file path). Noise lists
// substrings of node ids that should NOT be in the context.
type Task struct {
  Name   string   `json:"name"`
  Task   string   `json:"task"`
  Must   []string `json:"must"`
  Noise  []string `json:"noise,omitempty"`
  Budget int      `json:"budget,omitempty"`
}

type TaskFile struct {
  Name  string `json:"name"`
  Tasks []Task `json:"tasks"`
}

type TaskResult struct {
  Name      string   `json:"name"`
  Recall    float64  `json:"recall"`
  Found     int      `json:"found"`
  Total     int      `json:"total"`
  Missing   []string `json:"missing,omitempty"`
  NoiseHits []string `json:"noise_hits,omitempty"`
  Nodes     int      `json:"nodes"`
  Tokens    int      `json:"tokens"`
}

// ContextIDs extracts the node ids listed in a task context (lines that end with a backticked id).
func ContextIDs(md string) []string {
  var ids []string
  for _, line := range strings.Split(md, "\n") {
    if !strings.HasPrefix(line, "- ") || !strings.HasSuffix(line, "`") {
      continue
    }
    body := line[:len(line)-1]
    if j := strings.LastIndex(body, "`"); j >= 0 {
      ids = append(ids, body[j+1:])
    }
  }
  return ids
}

func moduleSection(md string) string {
  i := strings.Index(md, "## Module-scope code")
  if i < 0 {
    return ""
  }
  rest := md[i+3:]
  if j := strings.Index(rest, "\n## "); j >= 0 {
    rest = rest[:j]
  }
  return rest
}

func ScoreTask(t Task, md string) TaskResult {
  ids := ContextIDs(md)
  mod := moduleSection(md)
  r := TaskResult{Name: t.Name, Total: len(t.Must), Nodes: len(ids), Tokens: len(md) / 4}
  for _, m := range t.Must {
    hit := strings.Contains(mod, m)
    for _, id := range ids {
      if hit {
        break
      }
      if strings.Contains(id, m) {
        hit = true
      }
    }
    if hit {
      r.Found++
    } else {
      r.Missing = append(r.Missing, m)
    }
  }
  for _, id := range ids {
    for _, n := range t.Noise {
      if strings.Contains(id, n) {
        r.NoiseHits = append(r.NoiseHits, id)
        break
      }
    }
  }
  r.Recall = 1
  if r.Total > 0 {
    r.Recall = float64(r.Found) / float64(r.Total)
  }
  return r
}
