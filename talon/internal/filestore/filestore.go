package filestore

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

type Store struct {
	root string
}

func New(root string) (*Store, error) {
	if root == "" {
		return nil, fmt.Errorf("filestore path is required")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	return &Store{root: root}, nil
}

func Name(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func (s *Store) Path(name string) string {
	if len(name) < 4 {
		return filepath.Join(s.root, name)
	}
	return filepath.Join(s.root, name[:2], name[2:4], name)
}

func (s *Store) Save(data []byte) (string, error) {
	name := Name(data)
	path := s.Path(name)
	if _, err := os.Stat(path); err == nil {
		return name, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return "", err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return "", err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return "", err
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return "", err
	}
	return name, nil
}

func (s *Store) Get(name string) ([]byte, error) {
	if name == "" {
		return nil, fmt.Errorf("file name is required")
	}
	return os.ReadFile(s.Path(name))
}

func (s *Store) Delete(name string) error {
	if name == "" {
		return fmt.Errorf("file name is required")
	}
	err := os.Remove(s.Path(name))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (s *Store) Exists(name string) bool {
	if name == "" {
		return false
	}
	_, err := os.Stat(s.Path(name))
	return err == nil
}

func (s *Store) Root() string {
	return s.root
}
