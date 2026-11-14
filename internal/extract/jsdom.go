package extract

import (
  "regexp"
  "strings"

  "pidx/internal/model"
)

func init() {
  registry["html"] = extractHTML
  registry["css"] = extractCSS
  registry["javascript"] = jsWithDOM
  registry["typescript"] = jsWithDOM
}

var (
  reDomAssign = regexp.MustCompile(`([A-Za-z_$][\w$]*)\s*=\s*document\.(getElementById|querySelector)\(\s*['"\x60]([^'"\x60]+)['"\x60]\s*\)`)
  reDomChain  = regexp.MustCompile(`document\.(getElementById|querySelector)\(\s*['"\x60]([^'"\x60]+)['"\x60]\s*\)\s*\.(?:addEventListener\(\s*['"]([^'"]+)['"]\s*,|on([a-z]+)\s*=)\s*([A-Za-z_$][\w$.]*)?`)
  reVarListen = regexp.MustCompile(`([A-Za-z_$][\w$]*)\s*\.(?:addEventListener\(\s*['"]([^'"]+)['"]\s*,|on([a-z]+)\s*=)\s*([A-Za-z_$][\w$.]*)?`)
  reClassList = regexp.MustCompile(`\.classList\.(add|remove|toggle)\(\s*['"]([^'"]+)['"]`)
)

func selOf(fn, arg string) string {
  if fn == "getElementById" {
    return "#" + arg
  }
  return arg
}

// handlerName returns "" for anonymous handlers (function expressions / arrows).
func handlerName(h string) string {
  if h == "function" || h == "async" {
    return ""
  }
  return h
}

// jsWithDOM runs the JS extractor, then adds DOM binding refs:
//   dom_listen  Name=selector Target="event:handler"   (element -> handler binding)
//   dom_class   Name=class    Target=add|remove|toggle
// Variables assigned from getElementById/querySelector are tracked per file (flow-insensitive).
func jsWithDOM(path string, src []byte) model.FileFacts {
  f := extractJS(path, src)
  lines := strings.Split(strings.ReplaceAll(string(src), "\r\n", "\n"), "\n")
  vars := map[string]string{}
  for _, l := range lines {
    if isCommentLine(strings.TrimSpace(l)) {
      continue
    }
    for _, m := range reDomAssign.FindAllStringSubmatch(l, -1) {
      vars[m[1]] = selOf(m[2], m[3])
    }
  }
  for i, l := range lines {
    if isCommentLine(strings.TrimSpace(l)) {
      continue
    }
    ln := i + 1
    from := enclosing(f.Symbols, ln)
    add := func(kind, name, target string) {
      f.Refs = append(f.Refs, model.RawRef{Kind: kind, Name: name, Target: target, Line: ln, From: from})
    }
    for _, m := range reDomChain.FindAllStringSubmatch(l, -1) {
      ev := m[3]
      if ev == "" {
        ev = m[4]
      }
      add("dom_listen", selOf(m[1], m[2]), ev+":"+handlerName(m[5]))
    }
    for _, m := range reVarListen.FindAllStringSubmatch(l, -1) {
      sel, ok := vars[m[1]]
      if !ok {
        continue
      }
      ev := m[2]
      if ev == "" {
        ev = m[3]
      }
      add("dom_listen", sel, ev+":"+handlerName(m[4]))
    }
    for _, m := range reClassList.FindAllStringSubmatch(l, -1) {
      for _, c := range strings.Fields(m[2]) {
        add("dom_class", c, m[1])
      }
    }
  }
  return f
}
