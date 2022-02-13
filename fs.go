package main

import (
	"io/fs"
)

// fsReadBytes answers the bytes for the given file at path.
func fsReadBytes(dir fs.FS, path string) ([]byte, error) {
	data, err := fs.ReadFile(dir, path)
	if err != nil {
		return nil, err
	}
	return data, nil
}
