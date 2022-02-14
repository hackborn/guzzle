package main

import (
	"io/fs"
	"os"
)

// fsExists returns true if the path exists
func fsExists(path string) bool {
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return true
	} else {
		return false
	}
}

// fsNotExists returns true if the path does not exist
func fsNotExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return true
	} else {
		return false
	}
}

// fsReadBytes answers the bytes for the given file at path.
func fsReadBytes(dir fs.FS, path string) ([]byte, error) {
	data, err := fs.ReadFile(dir, path)
	if err != nil {
		return nil, err
	}
	return data, nil
}
