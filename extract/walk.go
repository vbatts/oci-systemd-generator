package extract

import (
	"os"
	"path/filepath"

	"github.com/vbatts/oci-systemd-generator/layout"
)

// WalkForExtracts walks a rootpath looking for all directories that match an
// extracted OCI image reference.
func WalkForExtracts(rootpath string) (extracts []Layout, err error) {
	namespath := filepath.Join(rootpath, nameNames)
	if _, err := os.Stat(namespath); err != nil && os.IsNotExist(err) {
		return nil, ErrNoExtracts
	} else if err != nil {
		return nil, err
	}
	extracts = []Layout{}
	err = filepath.Walk(namespath, func(path string, info os.FileInfo, err error) error {
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
		case nameRef:
			altDir = filepath.Join(dirname, nameRootfs)
		case nameRootfs:
			altDir = filepath.Join(dirname, nameRef)
		default:
			return nil
		}

		if altInfo, err := os.Lstat(altDir); err != nil || !altInfo.IsDir() {
			// either this is an error OR it is nil because the directory is not a directory,
			// so just skip it
			return nil
		}

		l, err := filepath.Rel(namespath, dirname)
		if err != nil {
			return err
		}
		extracts = append(extracts, Layout{Root: rootpath, Name: l})
		return nil
	})
	return extracts, err
}

// DetermineNotExtracted returns only the list of manifests that are not
// present in the provided list of extracted layouts.
func DetermineNotExtracted(extracts []Layout, manifests []*layout.Manifest) ([]*layout.Manifest, error) {
	ne := []*layout.Manifest{}
	for _, manifest := range manifests {
		found := false
		for _, el := range extracts {
			if manifest.Layout.Name != el.Name {
				continue
			}
			eRefs, err := el.Refs()
			if err != nil {
				return nil, err
			}
			for _, eRef := range eRefs {
				if manifest.Layout.Name == el.Name && manifest.Ref == eRef {
					found = true
				}
			}
		}
		if !found {
			ne = append(ne, manifest)
		}
	}
	return ne, nil
}
