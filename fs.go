package main

import (
	"io/fs"
	"os"
	"path/filepath"
)

// fsCopyDir copies source directory to the destination.
// Dst is the parent that will contain the new src.
// NOTE: Very quick implementation. There are much better
// ones on the net.
func fsCopyDir(src, dst string) error {
	dstRoot := filepath.Join(dst, filepath.Base(src))
	err := os.MkdirAll(dstRoot, os.ModePerm)
	if err != nil {
		return err
	}
	f := os.DirFS(src)
	err = fs.WalkDir(f, ".", func(path string, d fs.DirEntry, err error) error {
		if path == "." {
			return nil
		}
		dstPath := filepath.Join(dstRoot, path)
		if d.IsDir() {
			return os.MkdirAll(dstPath, os.ModePerm)
		} else {
			data, err := fs.ReadFile(f, path)
			if err != nil {
				return err
			}
			return os.WriteFile(dstPath, data, 0644)
		}
	})
	return err
}

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
