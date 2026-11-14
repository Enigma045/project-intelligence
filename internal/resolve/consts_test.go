package resolve

import (
  "testing"

  "pidx/internal/model"
)

func TestConstantInRequestURL(t *testing.T) {
  idLoad := "function:web/l.js#load"
  idRoute := "route:srv/m.go#ANY /api/send-letter"
  in := Input{
    Files: []model.FileRow{{Path: "web/l.js", Language: "javascript"}, {Path: "srv/m.go", Language: "go"}},
    Symbols: []model.Symbol{
      {ID: idLoad, Kind: "function", Name: "load", File: "web/l.js", StartLine: 6},
      {ID: idRoute, Kind: "route", Name: "ANY /api/send-letter", File: "srv/m.go", StartLine: 3},
    },
    Refs: []model.RawRef{
      {File: "web/l.js", Kind: "js_const", Name: "API_BASE_URL", Target: "/api", Line: 2},
      {File: "web/l.js", Kind: "api_request", Name: "${API_BASE_URL}/send-letter?id=${id}", Target: "GET", Line: 8, From: "load"},
      {File: "web/l.js", Kind: "api_request", Name: "${OTHER}/send-letter", Target: "GET", Line: 9, From: "load"},
    },
  }
  es := ResolveBackend(in)
  e := find(es, "requests_api", idLoad, "${API_BASE_URL}/send-letter?id=${id}")
  if e == nil || e.Dst != idRoute || e.Status != "inferred" {
    t.Fatalf("constant URL must resolve as inferred: %+v", e)
  }
  if u := find(es, "requests_api", idLoad, "${OTHER}/send-letter"); u == nil || u.Status != "dynamic" {
    t.Fatalf("unknown constant must stay dynamic: %+v", u)
  }
}
