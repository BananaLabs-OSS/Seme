package goupb08portruntime

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

type DirectoryStore struct{ Root string }

func (s DirectoryStore) Put(key, value []byte) error {
	if len(key) != 32 || !filepath.IsAbs(s.Root) {
		return fmt.Errorf("go_upb08_store.key")
	}
	name := fmt.Sprintf("%x", key)
	path := filepath.Join(s.Root, name)
	f, e := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if os.IsExist(e) {
		old, ok := s.Get(key)
		if !ok || !bytes.Equal(old, value) {
			return fmt.Errorf("go_upb08_store.collision")
		}
		return nil
	}
	if e != nil {
		return e
	}
	ok := false
	defer func() {
		_ = f.Close()
		if !ok {
			_ = os.Remove(path)
		}
	}()
	if _, e = f.Write(value); e != nil {
		return e
	}
	if e = f.Sync(); e != nil {
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	ok = true
	return nil
}
func (s DirectoryStore) Get(key []byte) ([]byte, bool) {
	if len(key) != 32 {
		return nil, false
	}
	p := filepath.Join(s.Root, fmt.Sprintf("%x", key))
	i, e := os.Lstat(p)
	if e != nil || !i.Mode().IsRegular() || i.Mode()&os.ModeSymlink != 0 {
		return nil, false
	}
	b, e := os.ReadFile(p)
	return b, e == nil
}
