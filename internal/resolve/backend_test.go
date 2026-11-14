package resolve

import (
  "testing"

  "pidx/internal/model"
)

func TestBackendChain(t *testing.T) {
  idRoute := "route:api/h.go#POST /api/register"
  idItems := "route:api/h.go#GET /api/items/{id}"
  idReg := "method:api/h.go#Handler.Register"
  idSubmit := "function:web/app.js#submit"
  idUsers := "table:db/schema.sql#users"
  in := Input{
    Files: []model.FileRow{
      {Path: "web/app.js", Language: "javascript"},
      {Path: "api/h.go", Language: "go"},
      {Path: "db/schema.sql", Language: "sql"},
    },
    Symbols: []model.Symbol{
      {ID: idRoute, Kind: "route", Name: "POST /api/register", File: "api/h.go", StartLine: 12},
      {ID: idItems, Kind: "route", Name: "GET /api/items/{id}", File: "api/h.go", StartLine: 13},
      {ID: idReg, Kind: "method", Name: "Register", Parent: "Handler", File: "api/h.go", StartLine: 20},
      {ID: idSubmit, Kind: "function", Name: "submit", File: "web/app.js", StartLine: 1},
      {ID: idUsers, Kind: "table", Name: "users", File: "db/schema.sql", StartLine: 1},
    },
    Refs: []model.RawRef{
      {File: "api/h.go", Kind: "route_handler", Name: "POST /api/register", Target: "h.Register", Line: 12, From: "Routes"},
      {File: "web/app.js", Kind: "api_request", Name: "/api/register", Target: "POST", Line: 2, From: "submit"},
      {File: "web/app.js", Kind: "api_request", Name: "/api/other", Line: 3, From: "submit"},
      {File: "web/app.js", Kind: "api_request", Name: "https://x.example.com/a", Line: 4, From: "submit"},
      {File: "web/app.js", Kind: "api_request", Name: "/api/items/42", Target: "GET", Line: 5, From: "submit"},
      {File: "api/h.go", Kind: "sql_use", Name: "users", Target: "insert", Line: 21, From: "Handler.Register"},
      {File: "api/h.go", Kind: "sql_use", Name: "ghosts", Target: "select", Line: 22, From: "Handler.Register"},
    },
  }
  es := ResolveBackend(in)
  check := func(typ, src, name, status, dst string) {
    t.Helper()
    e := find(es, typ, src, name)
    if e == nil || e.Status != status || e.Dst != dst {
      t.Fatalf("%s %s -> %s: want %s %q, got %+v", typ, src, name, status, dst, e)
    }
  }
  check("handles_route", idRoute, "h.Register", "inferred", idReg)
  check("requests_api", idSubmit, "/api/register", "resolved", idRoute)
  check("requests_api", idSubmit, "/api/other", "unresolved", "")
  check("requests_api", idSubmit, "https://x.example.com/a", "external", "")
  check("requests_api", idSubmit, "/api/items/42", "resolved", idItems)
  check("queries_table", idReg, "users", "resolved", idUsers)
  check("queries_table", idReg, "ghosts", "external", "")
}

func TestPackageQualifiedHandler(t *testing.T) {
  idFn := "function:Routes/r.go#Register"
  idAlias := "var:Routes/r.go#Alias"
  mk := func(n string) model.Symbol {
    return model.Symbol{ID: "route:cmd/main.go#" + n, Kind: "route", Name: n, File: "cmd/main.go", StartLine: 5}
  }
  in := Input{
    Files: []model.FileRow{
      {Path: "cmd/main.go", Language: "go", Imports: []string{"net/http", "mod/Routes"}},
      {Path: "Routes/r.go", Language: "go"},
    },
    Symbols: []model.Symbol{
      mk("ANY /register"), mk("ANY /alias"), mk("ANY /nf"), mk("ANY /ext"),
      {ID: idFn, Kind: "function", Name: "Register", File: "Routes/r.go", StartLine: 1},
      {ID: idAlias, Kind: "var", Name: "Alias", File: "Routes/r.go", StartLine: 9},
    },
    Refs: []model.RawRef{
      {File: "cmd/main.go", Kind: "route_handler", Name: "ANY /register", Target: "Routes.Register", Line: 5, From: "main"},
      {File: "cmd/main.go", Kind: "route_handler", Name: "ANY /alias", Target: "Routes.Alias", Line: 6, From: "main"},
      {File: "cmd/main.go", Kind: "route_handler", Name: "ANY /nf", Target: "Routes.Missing", Line: 7, From: "main"},
      {File: "cmd/main.go", Kind: "route_handler", Name: "ANY /ext", Target: "http.NotFound", Line: 8, From: "main"},
    },
  }
  es := ResolveBackend(in)
  check := func(route, name, status, dst string) {
    t.Helper()
    e := find(es, "handles_route", "route:cmd/main.go#"+route, name)
    if e == nil || e.Status != status || e.Dst != dst {
      t.Fatalf("%s -> %s: want %s %q, got %+v", route, name, status, dst, e)
    }
  }
  check("ANY /register", "Routes.Register", "resolved", idFn)
  check("ANY /alias", "Routes.Alias", "inferred", idAlias)
  check("ANY /nf", "Routes.Missing", "unresolved", "")
  check("ANY /ext", "http.NotFound", "external", "")
}

func TestTemplatePaths(t *testing.T) {
  idLoad := "function:web/app.js#load"
  idUsers := "route:srv/m.go#ANY /getuserdetails"
  idItems := "route:srv/m.go#GET /api/items/{id}"
  in := Input{
    Files: []model.FileRow{{Path: "web/app.js", Language: "javascript"}, {Path: "srv/m.go", Language: "go"}},
    Symbols: []model.Symbol{
      {ID: idLoad, Kind: "function", Name: "load", File: "web/app.js", StartLine: 1},
      {ID: idUsers, Kind: "route", Name: "ANY /getuserdetails", File: "srv/m.go", StartLine: 3},
      {ID: idItems, Kind: "route", Name: "GET /api/items/{id}", File: "srv/m.go", StartLine: 4},
    },
    Refs: []model.RawRef{
      {File: "web/app.js", Kind: "api_request", Name: "/getuserdetails?page=${page}&limit=${limit}", Line: 2, From: "load"},
      {File: "web/app.js", Kind: "api_request", Name: "/api/items/${id}", Target: "GET", Line: 3, From: "load"},
      {File: "web/app.js", Kind: "api_request", Name: "${base}/api/x", Line: 4, From: "load"},
    },
  }
  es := ResolveBackend(in)
  check := func(name, status, dst string) {
    t.Helper()
    e := find(es, "requests_api", idLoad, name)
    if e == nil || e.Status != status || e.Dst != dst {
      t.Fatalf("%s: want %s %q, got %+v", name, status, dst, e)
    }
  }
  check("/getuserdetails?page=${page}&limit=${limit}", "resolved", idUsers)
  check("/api/items/${id}", "inferred", idItems)
  check("${base}/api/x", "dynamic", "")
}

func TestRouteMatch(t *testing.T) {
  yes := [][2]string{{"/a/b", "/a/b"}, {"/a/{id}", "/a/9"}, {"/a/:id/c", "/a/9/c"}, {"/static/", "/static/x/y"}, {"/a/b", "/a/${x}"}}
  no := [][2]string{{"/a/b", "/a/c"}, {"/a/{id}", "/a/9/c"}, {"/a/b", "/a"}}
  for _, c := range yes {
    if !routeMatch(c[0], c[1]) {
      t.Errorf("%s should match %s", c[0], c[1])
    }
  }
  for _, c := range no {
    if routeMatch(c[0], c[1]) {
      t.Errorf("%s should not match %s", c[0], c[1])
    }
  }
}
