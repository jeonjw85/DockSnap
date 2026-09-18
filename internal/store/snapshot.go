package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Store struct {
	root string
}

func New(cwd string) *Store {
	return &Store{root: filepath.Join(cwd, ".dosnap", "snapshots")}
}

func (s *Store) Prepare(tag Tag) (string, error) {
	if err := os.MkdirAll(s.root, 0o755); err != nil {
		return "", fmt.Errorf("mkdir snapshots: %w", err)
	}
	name := tag.String()
	tmp := filepath.Join(s.root, name+".tmp")
	if err := os.RemoveAll(tmp); err != nil {
		return "", fmt.Errorf("clear %s.tmp: %w", name, err)
	}
	if err := os.Mkdir(tmp, 0o755); err != nil {
		return "", fmt.Errorf("mkdir %s.tmp: %w", name, err)
	}
	return tmp, nil
}

func (s *Store) Commit(m Meta) (err error) {
	tag := m.Tag.String()
	tmp := filepath.Join(s.root, tag+".tmp")
	dest := filepath.Join(s.root, tag)
	if err := os.MkdirAll(tmp, 0o755); err != nil {
		return fmt.Errorf("mkdir %s.tmp: %w", tag, err)
	}
	committed := false
	defer func() {
		if !committed {
			err = errors.Join(err, os.RemoveAll(tmp))
		}
	}()
	data, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshal %s: %w", tag, err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "meta.json"), data, 0o644); err != nil {
		return fmt.Errorf("write %s meta: %w", tag, err)
	}
	if err := os.RemoveAll(dest); err != nil {
		return fmt.Errorf("replace %s: %w", tag, err)
	}
	if err := os.Rename(tmp, dest); err != nil {
		return fmt.Errorf("rename %s: %w", tag, err)
	}
	committed = true
	return nil
}

func (s *Store) Save(m Meta) error {
	if _, err := s.Prepare(m.Tag); err != nil {
		return err
	}
	return s.Commit(m)
}

func (s *Store) Load(tag Tag) (Meta, error) {
	m, err := loadMeta(filepath.Join(s.root, tag.String(), "meta.json"))
	if err != nil {
		return Meta{}, fmt.Errorf("%s: %w", tag.String(), err)
	}
	return m, nil
}

func (s *Store) List() ([]Meta, error) {
	entries, err := os.ReadDir(s.root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("list snapshots: %w", err)
	}
	var out []Meta
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".tmp") {
			continue
		}
		m, err := loadMeta(filepath.Join(s.root, name, "meta.json"))
		if err != nil {
			continue
		}
		out = append(out, m)
	}
	return out, nil
}

func loadMeta(path string) (Meta, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Meta{}, err
	}
	var m Meta
	if err := json.Unmarshal(data, &m); err != nil {
		return Meta{}, err
	}
	return m, nil
}
