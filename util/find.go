package util

import (
	"os"
	"path/filepath"
)

// FindFilesOrSymlink does a walk, looking only for regular files or symlinks
func FindFilesOrSymlink(basename string) ([]string, error) {
	files := []string{}
	err := filepath.Walk(basename, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() || (info.Mode()&os.ModeSymlink) != 0 {
			return nil
		}
		ref, err := filepath.Rel(basename, path)
		if err != nil {
			return err
		}
		files = append(files, ref)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}
