package main

import (
	"os"
	"path/filepath"
)

// Layouts is a collections OCI image layouts
type Layouts map[string]Layout

// Layout is an OCI image layout that includes descriptor refs and the content
// addressible objects pointed to by the descriptors.
type Layout struct {
	Root string
	Name string
}

// Refs gives the path to all regular files or symlinks in this layout's "refs" directory
func (l Layout) Refs() ([]string, error) {
	return findFilesOrSymlink(filepath.Join(l.Root, l.Name, "refs"))
}

// Blobs gives the path to all regular files or symlinks in this layout's "blobs" directory
func (l Layout) Blobs() ([]string, error) {
	return findFilesOrSymlink(filepath.Join(l.Root, l.Name, "blobs"))
}

func findFilesOrSymlink(basename string) ([]string, error) {
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

// WalkForLayouts looks through rootpath for OCI image-layout directories. Namely a directory that has "refs" and "blobs" directory, and an oci-layout file.
func WalkForLayouts(rootpath string) (layouts Layouts, err error) {
	layouts = Layouts{}
	err = filepath.Walk(rootpath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return nil
		}
		var (
			altDir   string
			basename = filepath.Base(path)
			dirname  = filepath.Dir(path)
		)
		switch basename {
		case "refs":
			altDir = filepath.Join(dirname, "blobs")
		case "blobs":
			altDir = filepath.Join(dirname, "refs")
		default:
			return nil
		}

		if altInfo, err := os.Lstat(altDir); err != nil || !altInfo.IsDir() {
			// either this is an error OR it is nil because the directory is not a directory,
			// so just skip it
			return nil
		}
		if _, err := os.Stat(filepath.Join(dirname, "oci-layout")); os.IsNotExist(err) {
			// does not have oci version file, so skip it.
			Debugf("%q does not have an oci-layout file", dirname)
			return nil
		}

		l, err := filepath.Rel(rootpath, dirname)
		if err != nil {
			return err
		}
		if _, ok := layouts[l]; !ok {
			layouts[l] = Layout{Root: rootpath, Name: l}
		}
		return nil
	})
	return layouts, err
}
