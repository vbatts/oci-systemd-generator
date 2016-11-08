package layout

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/vbatts/oci-systemd-generator/util"
)

// Layouts is a collections OCI image layouts
type Layouts map[string]*Layout

// Layout is an OCI image layout that includes descriptor refs and the content
// addressible objects pointed to by the descriptors.
type Layout struct {
	Root string
	Name string
}

const (
	nameLayout = "oci-layout"
	nameBlobs  = "blobs"
	nameRefs   = "refs"

	digestSeparator = ":"
)

// GetBlob returns the stream for a blob addressed by it's digest (`sha256:abcde123456...`)
func (l Layout) GetBlob(digest DigestRef) (io.ReadCloser, error) {
	path := filepath.Join(l.Root, l.Name, nameBlobs, digestToPath(digest))
	if _, err := os.Stat(path); err != nil && os.IsNotExist(err) {
		return nil, err
	}
	return os.Open(path)
}

// GetRef loads the descriptor reference for this OCI image
func (l Layout) GetRef(name string) (*v1.Descriptor, error) {
	buf, err := ioutil.ReadFile(filepath.Join(l.Root, l.Name, nameRefs, name))
	if err != nil {
		return nil, err
	}
	var desc v1.Descriptor
	if err := json.Unmarshal(buf, &desc); err != nil {
		return nil, err
	}
	return &desc, nil
}

// OCIVersion reads the OCI image layout version for this layout
func (l Layout) OCIVersion() (string, error) {
	buf, err := ioutil.ReadFile(filepath.Join(l.Root, l.Name, nameLayout))
	if err != nil {
		return "", err
	}

	var ociImageLayout v1.ImageLayout
	if err := json.Unmarshal(buf, &ociImageLayout); err != nil {
		return "", err
	}

	return ociImageLayout.Version, nil
}

// Refs gives the path to all regular files or symlinks in this layout's "refs" directory
func (l Layout) Refs() ([]string, error) {
	return util.FindFilesOrSymlink(filepath.Join(l.Root, l.Name, nameRefs))
}

// Blobs gives the path to all regular files or symlinks in this layout's "blobs" directory
func (l *Layout) Blobs() ([]DigestRef, error) {
	paths, err := util.FindFilesOrSymlink(filepath.Join(l.Root, l.Name, nameBlobs))
	if err != nil {
		return nil, err
	}
	digests := []DigestRef{}
	for _, path := range paths {
		digest := pathToDigest(path)
		if digest == nil {
			continue
		}
		digest.Layout = l
		digests = append(digests, *digest)
	}
	return digests, nil
}

// WalkForLayouts looks through rootpath for OCI image-layout directories.
// Namely a directory that has "refs" and "blobs" directory, and an oci-layout
// file.
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
		case nameRefs:
			altDir = filepath.Join(dirname, nameBlobs)
		case nameBlobs:
			altDir = filepath.Join(dirname, nameRefs)
		default:
			return nil
		}

		if altInfo, err := os.Lstat(altDir); err != nil || !altInfo.IsDir() {
			// either this is an error OR it is nil because the directory is not a directory,
			// so just skip it
			return nil
		}
		if _, err := os.Stat(filepath.Join(dirname, nameLayout)); os.IsNotExist(err) {
			// does not have oci version file, so skip it.
			util.Debugf("%q does not have an oci-layout file", dirname)
			return nil
		}

		l, err := filepath.Rel(rootpath, dirname)
		if err != nil {
			return err
		}
		if _, ok := layouts[l]; !ok {
			layouts[l] = &Layout{Root: rootpath, Name: l}
		}
		return nil
	})
	return layouts, err
}
