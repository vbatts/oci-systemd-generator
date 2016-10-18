package main

import (
	"os"
	"path/filepath"
)

type Layouts map[string]Layout
type Layout struct {
	Root string
	Name string
}

func (l Layout) Refs() ([]string, error) {
	refs := []string{}
	basename := filepath.Join(l.Root, l.Name, "refs")
	err := filepath.Walk(basename, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		ref, err := filepath.Rel(basename, path)
		if err != nil {
			return err
		}
		refs = append(refs, ref)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return refs, nil
}

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
			// either this is an error OR it is nil because the directory is not a directory
			return err
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
