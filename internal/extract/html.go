package extract

import (
  "strconv"
  "strings"

  "pidx/internal/model"
)

var voidTags = map[string]bool{
  "area": true, "base": true, "br": true, "col": true, "embed": true, "hr": true, "img": true, "input": true,
  "link": true, "meta": true, "param": true, "source": true, "track": true, "wbr": true,
}

type htmlAttr struct{ name, val string }

type htmlTag struct {
  name       string
  attrs      []htmlAttr
  closing    bool
  selfClose  bool
  start, end int
}

func (t htmlTag) attr(n string) (string, bool) {
  for _, a := range t.attrs {
    if a.name == n {
      return a.val, true
    }
  }
  return "", false
}

// parseTag parses the tag starting at s[i]=='<'. Returns ok=false if it is not a well-formed tag.
func parseTag(s string, i int) (htmlTag, bool) {
  t := htmlTag{start: i}
  j := i + 1
  if j < len(s) && s[j] == '/' {
    t.closing = true
    j++
  }
  k := j
  for k < len(s) && !isWS(s[k]) && s[k] != '/' && s[k] != '>' {
    k++
  }
  if k == j || !isLetter(s[j]) {
    return t, false
  }
  t.name = asciiLower(s[j:k])
  j = k
  for j < len(s) {
    for j < len(s) && isWS(s[j]) {
      j++
    }
    if j >= len(s) {
      return t, false
    }
    if s[j] == '>' {
      t.end = j + 1
      return t, true
    }
    if s[j] == '/' {
      if j+1 < len(s) && s[j+1] == '>' {
        t.selfClose = true
        t.end = j + 2
        return t, true
      }
      j++
      continue
    }
    a := j
    for j < len(s) && !isWS(s[j]) && s[j] != '=' && s[j] != '>' && s[j] != '/' {
      j++
    }
    if a == j {
      j++
      continue
    }
    name := asciiLower(s[a:j])
    for j < len(s) && isWS(s[j]) {
      j++
    }
    val := ""
    if j < len(s) && s[j] == '=' {
      j++
      for j < len(s) && isWS(s[j]) {
        j++
      }
      if j < len(s) && (s[j] == '"' || s[j] == '\'') {
        q := s[j]
        j++
        b := j
        for j < len(s) && s[j] != q {
          j++
        }
        val = s[b:j]
        if j < len(s) {
          j++
        }
      } else {
        b := j
        for j < len(s) && !isWS(s[j]) && s[j] != '>' {
          j++
        }
        val = s[b:j]
      }
    }
    t.attrs = append(t.attrs, htmlAttr{name, val})
  }
  return t, false
}

type openEl struct {
  tag   string
  off   int
  sym   int
  form  bool
  label string
}

// extractHTML emits: element/form/field symbols, and refs: html_class, html_handler, html_link,
// script_src, style_href. Inline <script> and <style> blocks are extracted as embedded JS/CSS.
func extractHTML(path string, src []byte) model.FileFacts {
  facts := model.FileFacts{Path: path, Language: "html", ParseStatus: "heuristic"}
  s := string(src)
  lower := asciiLower(s)
  starts := lineStarts(s)
  var stack []openEl

  closeEl := func(e openEl, endOff, line int) {
    if e.sym < 0 {
      return
    }
    sy := &facts.Symbols[e.sym]
    sy.EndLine = line
    if endOff > len(s) {
      endOff = len(s)
    }
    sy.BodyHash = model.HashString(s[e.off:endOff])
  }

  i := 0
  for i < len(s) {
    j := strings.IndexByte(s[i:], '<')
    if j < 0 {
      break
    }
    i += j
    if strings.HasPrefix(s[i:], "<!--") {
      k := strings.Index(s[i+4:], "-->")
      if k < 0 {
        i = len(s)
      } else {
        i += 4 + k + 3
      }
      continue
    }
    if strings.HasPrefix(s[i:], "<!") || strings.HasPrefix(s[i:], "<?") {
      k := strings.IndexByte(s[i:], '>')
      if k < 0 {
        i = len(s)
      } else {
        i += k + 1
      }
      continue
    }
    t, ok := parseTag(s, i)
    if !ok {
      i++
      continue
    }
    i = t.end
    line := lineAt(starts, t.start)

    if t.closing {
      for n := len(stack) - 1; n >= 0; n-- {
        if stack[n].tag == t.name {
          for m := len(stack) - 1; m >= n; m-- {
            closeEl(stack[m], t.end, line)
          }
          stack = stack[:n]
          break
        }
      }
      continue
    }

    id, _ := t.attr("id")
    label := ""
    if id != "" {
      label = t.name + "#" + id
    }
    sym := -1
    if t.name == "form" || id != "" {
      name, kind := label, "element"
      if t.name == "form" {
        kind = "form"
        if name == "" {
          name = "form@L" + strconv.Itoa(line)
          label = name
        }
      }
      sym = len(facts.Symbols)
      facts.Symbols = append(facts.Symbols, model.Symbol{
        Kind: kind, Name: name, StartLine: line, EndLine: line, Visibility: "public",
        BodyHash: model.HashString(s[t.start:t.end]), Provenance: "verified",
      })
    }
    if cl, has := t.attr("class"); has {
      for _, c := range strings.Fields(cl) {
        facts.Refs = append(facts.Refs, model.RawRef{Kind: "html_class", Name: c, Target: t.name, Line: line, From: label})
      }
    }
    for _, a := range t.attrs {
      if len(a.name) > 2 && strings.HasPrefix(a.name, "on") && strings.TrimSpace(a.val) != "" {
        facts.Refs = append(facts.Refs, model.RawRef{Kind: "html_handler", Name: a.name, Target: strings.TrimSpace(a.val), Line: line, From: label})
      }
    }
    switch t.name {
    case "a":
      if h, has := t.attr("href"); has && h != "" && !strings.HasPrefix(h, "#") && !strings.HasPrefix(h, "javascript:") {
        facts.Refs = append(facts.Refs, model.RawRef{Kind: "html_link", Name: h, Line: line, From: label})
      }
    case "link":
      rel, _ := t.attr("rel")
      href, _ := t.attr("href")
      if href != "" && strings.Contains(strings.ToLower(rel), "stylesheet") {
        facts.Refs = append(facts.Refs, model.RawRef{Kind: "style_href", Name: href, Line: line})
      }
    case "input", "select", "textarea", "button":
      if nm, has := t.attr("name"); has && nm != "" {
        typ, hasType := t.attr("type")
        if !hasType {
          typ = map[string]string{"input": "text", "select": "select", "textarea": "textarea", "button": "submit"}[t.name]
        }
        parent := ""
        for n := len(stack) - 1; n >= 0; n-- {
          if stack[n].form {
            parent = stack[n].label
            break
          }
        }
        facts.Symbols = append(facts.Symbols, model.Symbol{
          Kind: "field", Name: nm, Parent: parent, StartLine: line, EndLine: line,
          Returns: strings.ToLower(typ), ReturnsSource: "declared", Visibility: "public",
          BodyHash: model.HashString(s[t.start:t.end]), Provenance: "verified",
        })
      }
    }

    if t.name == "script" || t.name == "style" {
      end := len(s)
      if k := strings.Index(lower[i:], "</"+t.name); k >= 0 {
        end = i + k
      }
      offset := lineAt(starts, i) - 1
      if t.name == "script" {
        if sv, has := t.attr("src"); has && sv != "" {
          facts.Refs = append(facts.Refs, model.RawRef{Kind: "script_src", Name: sv, Line: line})
        } else {
          typ, _ := t.attr("type")
          typ = strings.ToLower(typ)
          if typ == "" || strings.Contains(typ, "javascript") || typ == "module" {
            mergeInline(&facts, jsWithDOM("inline.js", []byte(s[i:end])), offset)
          }
        }
      } else {
        mergeInline(&facts, extractCSS("inline.css", []byte(s[i:end])), offset)
      }
      if sym >= 0 {
        facts.Symbols[sym].EndLine = lineAt(starts, end)
      }
      i = end
      continue
    }

    if voidTags[t.name] || t.selfClose {
      continue
    }
    stack = append(stack, openEl{tag: t.name, off: t.start, sym: sym, form: t.name == "form", label: label})
  }
  last := lineAt(starts, len(s))
  for _, e := range stack {
    closeEl(e, len(s), last)
  }
  return facts
}
