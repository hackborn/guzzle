package main

import (
	"errors"
	"fmt"
	"io"
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

func fsDirEmpty(sys fs.FS, path string) (bool, error) {
	f, err := sys.Open(path)
	if err != nil {
		err = fmt.Errorf("open err %v %w", path, err)
		return false, err
	}
	defer f.Close()
	rd, ok := f.(fs.ReadDirFile)
	if !ok {
		return false, fmt.Errorf("ReadDirFS not supported")
	}
	entries, err := rd.ReadDir(1)
	// EOF indicates is the main way we know it's
	// empty, but we'll also take an empty response.
	if errors.Is(err, io.EOF) || (err == nil && len(entries) < 1) {
		return true, nil
	}
	return false, err
}
