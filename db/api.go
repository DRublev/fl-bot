package db

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
)

type DB struct{}

func (d *DB) getStoragePath(storageName []string) (string, error) {
	_, base, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.New("Not ok getting info about caller")
	}
	dir := path.Join(path.Dir(base))
	rootDir := filepath.Dir(dir)

	paths := append([]string{rootDir, "storage"}, storageName...)
	p := path.Join(paths...)
	return p, nil
}

func (d *DB) Persist(storageName []string, content []byte) error {
	return nil
}

func (d *DB) Append(storageName []string, content []byte) error {
	storageFile, err := d.getStoragePath(storageName)
	fmt.Println("Appending to ", storageFile)
	if err != nil {
		log.Default().Println("Failed to get storage path: ", err)
		return err
	}

	if _, err := os.Stat(storageFile); os.IsNotExist(err) {
		dir, _ := d.getStoragePath(storageName[:len(storageName)-1])
		os.MkdirAll(dir, 0700)
	}

	file, err := os.OpenFile(storageFile, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0660)
	if err != nil {
		log.Default().Println("Failed to open file: ", err)
		return err
	}
	defer file.Close()

	log.Default().Println("Appending to file: ", storageFile)
	_, err = file.Write(content)

	return err
}

func (d *DB) Get(storageName []string) ([]byte, error) {
	var result []byte
	return result, nil
}
