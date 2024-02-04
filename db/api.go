package db

import (
	"errors"
	"os"
	"path"
	"path/filepath"
	"runtime"
)

type DB struct{}

func (d *DB) getStoragePath(storageName []string) (string, error) {
	_, base, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.New("Not ok getting info aboul caller")
	}
	dir := path.Join(path.Dir(base))
	rootDir := filepath.Dir(dir)

	paths := append([]string{rootDir, "db"}, storageName...)
	p := path.Join(paths...)
	return p, nil
}

func (d *DB) Persist(storageName []string, content []byte) error {
	return nil
}

func (d *DB) Append(storageName []string, content []byte) error {
	return nil
}

func (d *DB) Get(storageName []string) ([]byte, error) {
	var result []byte
	return result, nil
}
