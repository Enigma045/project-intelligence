package extract

import "pidx/internal/model"

// Extractor is a pure function: (path, bytes) -> facts. Swap in Tree-sitter adapters here later.
type Extractor func(path string, src []byte) model.FileFacts

var registry = map[string]Extractor{
  "go":         extractGo,
  "javascript": extractJS,
  "typescript": extractJS,
  "python":     extractPython,
}

func Supported(lang string) bool {
  _, ok := registry[lang]
  return ok
}

func Run(lang, path string, src []byte) model.FileFacts {
  if ex, ok := registry[lang]; ok {
    return finalize(path, ex(path, src))
  }
  return model.FileFacts{Path: path, Language: lang, ParseStatus: "unsupported"}
}
