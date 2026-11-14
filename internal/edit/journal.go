package edit

import (
  "encoding/json"
  "fmt"
  "os"
  "path/filepath"
  "sort"
  "strings"
  "time"
)

type JEntry struct {
  Path    string `json:"path"`
  Existed bool   `json:"existed"`
}

type Manifest struct {
  ID    string   `json:"id"`
  Files []JEntry `json:"files"`
}

func journalDir(root string) string { return filepath.Join(root, ".project-intelligence", "journal") }

func absPath(root, rel string) string { return filepath.Join(root, filepath.FromSlash(rel)) }

// WriteJournal snapshots pre-images of every file about to change and returns the journal id.
func WriteJournal(root string, st []Staged) (string, error) {
  id := strings.ReplaceAll(time.Now().UTC().Format("20060102-150405.000"), ".", "-")
  dir := filepath.Join(journalDir(root), id)
  if err := os.MkdirAll(dir, 0o755); err != nil {
    return "", err
  }
  m := Manifest{ID: id}
  for _, s := range st {
    m.Files = append(m.Files, JEntry{Path: s.Path, Existed: s.Existed})
    if !s.Existed {
      continue
    }
    p := filepath.Join(dir, "files", filepath.FromSlash(s.Path))
    if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
      return "", err
    }
    if err := os.WriteFile(p, s.Before, 0o644); err != nil {
      return "", err
    }
  }
  b, err := json.MarshalIndent(m, "", "  ")
  if err != nil {
    return "", err
  }
  return id, os.WriteFile(filepath.Join(dir, "manifest.json"), b, 0o644)
}

// Commit writes staged files (temp file + rename). On error the caller rolls back from the journal.
func Commit(root string, st []Staged) error {
  for _, s := range st {
    full := absPath(root, s.Path)
    if s.Delete {
      if err := os.Remove(full); err != nil && !os.IsNotExist(err) {
        return err
      }
      continue
    }
    if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
      return err
    }
    tmp := full + ".pidx-tmp"
    if err := os.WriteFile(tmp, s.After, 0o644); err != nil {
      return err
    }
    if err := os.Rename(tmp, full); err != nil {
      _ = os.Remove(tmp)
      return err
    }
  }
  return nil
}

// Rollback restores pre-images (and removes files the edit created), then marks the entry as rolled back.
func Rollback(root, id string) error {
  dir := filepath.Join(journalDir(root), id)
  b, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
  if err != nil {
    return err
  }
  var m Manifest
  if err := json.Unmarshal(b, &m); err != nil {
    return err
  }
  for _, f := range m.Files {
    full := absPath(root, f.Path)
    if f.Existed {
      data, err := os.ReadFile(filepath.Join(dir, "files", filepath.FromSlash(f.Path)))
      if err != nil {
        return err
      }
      if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
        return err
      }
      if err := os.WriteFile(full, data, 0o644); err != nil {
        return err
      }
    } else if err := os.Remove(full); err != nil && !os.IsNotExist(err) {
      return err
    }
  }
  return os.WriteFile(filepath.Join(dir, "rolled_back"), []byte("1"), 0o644)
}

// Latest returns the newest journal entry that has not been rolled back.
func Latest(root string) (string, error) {
  ents, err := os.ReadDir(journalDir(root))
  if err != nil {
    return "", fmt.Errorf("no journal entries")
  }
  var ids []string
  for _, e := range ents {
    if !e.IsDir() {
      continue
    }
    if _, err := os.Stat(filepath.Join(journalDir(root), e.Name(), "rolled_back")); err == nil {
      continue
    }
    ids = append(ids, e.Name())
  }
  if len(ids) == 0 {
    return "", fmt.Errorf("nothing to undo")
  }
  sort.Strings(ids)
  return ids[len(ids)-1], nil
}
