package scan

import (
  "bufio"
  "io/fs"
  "os"
  "path"
  "path/filepath"
  "sort"
  "strings"
)

type Entry struct {
  Rel      string
  Abs      string
  Language string
  Size     int64
  MTimeNS  int64
}

const MaxFileSize = 2 << 20

var langByExt = map[string]string{
  ".go": "go", ".js": "javascript", ".mjs": "javascript", ".cjs": "javascript", ".jsx": "javascript",
  ".ts": "typescript", ".tsx": "typescript", ".py": "python", ".html": "html", ".htm": "html",
  ".css": "css", ".sql": "sql", ".json": "json", ".md": "markdown",
}

var skipDirs = map[string]bool{
  ".git": true, "node_modules": true, ".project-intelligence": true, "vendor": true, "dist": true,
  "build": true, "__pycache__": true, ".venv": true, "venv": true, ".idea": true, ".vscode": true,
}

func Detect(name string) string { return langByExt[strings.ToLower(filepath.Ext(name))] }

func loadIgnore(root string) []string {
  var pats []string
  for _, n := range []string{".gitignore", ".pidxignore"} {
    f, err := os.Open(filepath.Join(root, n))
    if err != nil {
      continue
    }
    sc := bufio.NewScanner(f)
    for sc.Scan() {
      l := strings.TrimSpace(sc.Text())
      if l == "" || strings.HasPrefix(l, "#") || strings.HasPrefix(l, "!") {
        continue
      }
      pats = append(pats, l)
    }
    f.Close()
  }
  return pats
}

func ignored(pats []string, rel string) bool {
  base := path.Base(rel)
  for _, p := range pats {
    p = strings.TrimPrefix(strings.TrimSuffix(p, "/"), "/")
    if p == "" {
      continue
    }
    if strings.Contains(p, "/") {
      if ok, _ := path.Match(p, rel); ok {
        return true
      }
    } else if ok, _ := path.Match(p, base); ok {
      return true
    }
  }
  return false
}

// Walk lists indexable files (recognised language only), sorted by relative path.
func Walk(root string) ([]Entry, error) {
  pats := loadIgnore(root)
  var out []Entry
  err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
    if err != nil {
      return nil
    }
    rel, rerr := filepath.Rel(root, p)
    if rerr != nil || rel == "." {
      return nil
    }
    rel = filepath.ToSlash(rel)
    if d.IsDir() {
      if skipDirs[d.Name()] || ignored(pats, rel) {
        return filepath.SkipDir
      }
      return nil
    }
    lang := Detect(d.Name())
    if lang == "" || ignored(pats, rel) {
      return nil
    }
    info, ierr := d.Info()
    if ierr != nil || info.Size() > MaxFileSize {
      return nil
    }
    out = append(out, Entry{Rel: rel, Abs: p, Language: lang, Size: info.Size(), MTimeNS: info.ModTime().UnixNano()})
    return nil
  })
  sort.Slice(out, func(i, j int) bool { return out[i].Rel < out[j].Rel })
  return out, err
}
