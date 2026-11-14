package extract

import (
  "regexp"
  "strings"

  "pidx/internal/model"
)

// This file is named zz_* so its init runs after the other files' init functions and wins.
func init() {
  registry["go"] = goWithBackend
  registry["javascript"] = jsWithAPI
  registry["typescript"] = jsWithAPI
  registry["sql"] = extractSQL
}

func goWithBackend(path string, src []byte) model.FileFacts {
  f := extractGo(path, src)
  syms, refs := goBackend(path, src)
  f.Symbols = append(f.Symbols, syms...)
  f.Refs = append(f.Refs, refs...)
  f.Refs = append(f.Refs, goPackageRefs(src)...)
  return f
}

var reFetchMethod = regexp.MustCompile(`(?i)method\s*:\s*['"]([A-Za-z]+)['"]`)

// jsWithAPI = DOM extractor + fetch methods + cleanup of refs the line-based extractor gets wrong.
func jsWithAPI(path string, src []byte) model.FileFacts {
  f := jsWithDOM(path, src)
  lines := strings.Split(strings.ReplaceAll(string(src), "\r\n", "\n"), "\n")
  addFetchMethods(&f, lines)
  cleanJSRefs(&f, lines)
  addJSConsts(&f, lines)
  return f
}

// addFetchMethods adds the HTTP method to api_request refs (Target) by looking at the fetch call options
// within the next three lines, stopping at the next fetch( so neighbouring calls are not mixed up.
func addFetchMethods(f *model.FileFacts, lines []string) {
  for i := range f.Refs {
    r := &f.Refs[i]
    if r.Kind != "api_request" {
      continue
    }
    end := r.Line + 3
    if end > len(lines) {
      end = len(lines)
    }
    if r.Line < 1 || r.Line-1 >= end {
      continue
    }
    text := strings.Join(lines[r.Line-1:end], "\n")
    if first := strings.Index(text, "fetch("); first >= 0 {
      if rest := strings.Index(text[first+6:], "fetch("); rest >= 0 {
        text = text[:first+6+rest]
      }
    }
    if m := reFetchMethod.FindStringSubmatch(text); m != nil {
      r.Target = strings.ToUpper(m[1])
    }
  }
}

var anonWords = map[string]bool{"function": true, "async": true}

func lineOf(lines []string, n int) string {
  if n < 1 || n > len(lines) {
    return ""
  }
  return lines[n-1]
}

func identByte(c byte) bool {
  return c == '_' || c == '$' || (c >= '0' && c <= '9') || (c|32) >= 'a' && (c|32) <= 'z'
}

// callForms reports whether line contains name( as a standalone call, and/or as a method call (.name().
func callForms(line, name string) (standalone, dotted bool) {
  from := 0
  for from < len(line) {
    i := strings.Index(line[from:], name)
    if i < 0 {
      break
    }
    i += from
    from = i + 1
    if i > 0 && identByte(line[i-1]) {
      continue
    }
    j := i + len(name)
    if j < len(line) && identByte(line[j]) {
      continue
    }
    for j < len(line) && (line[j] == ' ' || line[j] == '\t') {
      j++
    }
    if j >= len(line) || line[j] != '(' {
      continue
    }
    if i > 0 && line[i-1] == '.' {
      dotted = true
    } else {
      standalone = true
    }
  }
  return
}

// isArrowParam reports whether name appears in line as the single parameter of an arrow function (e => ...).
func isArrowParam(line, name string) bool {
  if name == "" || strings.Contains(name, ".") {
    return false
  }
  from := 0
  for from < len(line) {
    i := strings.Index(line[from:], name)
    if i < 0 {
      return false
    }
    i += from
    from = i + 1
    if i > 0 && identByte(line[i-1]) {
      continue
    }
    rest := strings.TrimLeft(line[i+len(name):], " \t")
    if strings.HasPrefix(rest, "=>") {
      return true
    }
  }
  return false
}

// cleanJSRefs fixes three systematic mistakes of the line-based JS extractor:
//  - a call right after a dot (x.filter(..).map(..)) is a method call, not a plain function: rename to ?.name
//  - async / await are keywords, not calls
//  - function expressions and arrow params are anonymous handlers, not handler names
func cleanJSRefs(f *model.FileFacts, lines []string) {
  out := make([]model.RawRef, 0, len(f.Refs))
  for _, r := range f.Refs {
    line := lineOf(lines, r.Line)
    switch r.Kind {
    case "call":
      if r.Name == "async" || r.Name == "await" {
        continue
      }
      if !strings.Contains(r.Name, ".") {
        if sa, dt := callForms(line, r.Name); dt && !sa {
          r.Name = "?." + r.Name
        }
      }
    case "event":
      if anonWords[r.Target] || isArrowParam(line, r.Target) {
        r.Target = ""
      }
    case "dom_listen":
      if i := strings.Index(r.Target, ":"); i >= 0 {
        h := r.Target[i+1:]
        if h != "" && (anonWords[h] || isArrowParam(line, h)) {
          r.Target = r.Target[:i+1]
        }
      }
    }
    out = append(out, r)
  }
  f.Refs = out
}
