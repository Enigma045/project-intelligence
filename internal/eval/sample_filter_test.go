package eval

import (
  "testing"

  "pidx/internal/model"
)

func TestSampleSkipsChainCalls(t *testing.T) {
  edges := []model.Edge{
    ed("a.go", 1, "calls", "?.Get", "", "dynamic"),
    ed("a.go", 2, "calls", "user_logs.Create_user_log", "", "dynamic"),
  }
  got := SampleEdges(edges, DefaultTypes, 10, "s", nil)
  if len(got) != 1 || got[0].DstName != "user_logs.Create_user_log" {
    t.Fatalf("chain calls must not be sampled: %+v", got)
  }
}
