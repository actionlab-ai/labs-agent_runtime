package runstore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Store struct {
	RunID string
	Root  string
}

func New(runsDir string) (*Store, error) {
	id := "run-" + time.Now().Format("20060102-150405-000000000")
	root := filepath.Join(runsDir, id)
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	return &Store{RunID: id, Root: root}, nil
}

func (s *Store) Path(parts ...string) string {
	all := append([]string{s.Root}, parts...)
	return filepath.Join(all...)
}

func (s *Store) WriteText(rel string, content string) error {
	p := s.Path(split(rel)...)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, []byte(content), 0o644)
}

func (s *Store) WriteJSON(rel string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return s.WriteText(rel, string(b)+"\n")
}

func (s *Store) PrintSummary() string {
	return fmt.Sprintf("run_id=%s\nrun_dir=%s", s.RunID, s.Root)
}

func split(rel string) []string {
	rel = strings.TrimPrefix(rel, "/")
	if rel == "" {
		return nil
	}
	return strings.Split(rel, "/")
}
