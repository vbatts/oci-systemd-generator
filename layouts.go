package main

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/opencontainers/image-spec/specs-go/v1"
)

// Layouts is a collections OCI image layouts
type Layouts map[string]Layout

// Layout is an OCI image layout that includes descriptor refs and the content
// addressible objects pointed to by the descriptors.
type Layout struct {
	Root string
	Name string
}

// "./sha256/ed2dca7ba0aa32384f2f5560513dbb0325c8e213b75eb662055e8bd1db7ac974" -> "sha256:ed2dca7ba0aa32384f2f5560513dbb0325c8e213b75eb662055e8bd1db7ac974"
func pathToDigest(path string) string {
	chunks := strings.Split(filepath.Clean(path), "/")
	if len(chunks) > 1 && chunks[0] == nameBlobs {
		chunks = chunks[1:]
	}
	if len(chunks) != 2 {
		return ""
	}
	return chunks[0] + digestSeparator + chunks[1]
}

// "sha256:ed2dca7ba0aa32384f2f5560513dbb0325c8e213b75eb662055e8bd1db7ac974" -> "./sha256/ed2dca7ba0aa32384f2f5560513dbb0325c8e213b75eb662055e8bd1db7ac974"
func digestToPath(digest string) string {
	chunks := strings.SplitN(digest, digestSeparator, 2)
	if len(chunks) != 2 {
		return ""
	}
	return filepath.Join(chunks[0], chunks[1])
}

const (
	nameLayout = "oci-layout"
	nameBlobs  = "blobs"
	nameRefs   = "refs"

	digestSeparator = ":"
)

// GetBlob returns the stream for a blob addressed by it's digest (`sha256:abcde123456...`)
func (l Layout) GetBlob(digest string) (io.ReadCloser, error) {
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

// ociImageLayoutVersion XXX this struct is not specified in the specs-go/ package?
// TODO remove after https://github.com/opencontainers/image-spec/pull/393
type ociImageLayoutVersion struct {
	ImageLayoutVersion string `json:"imageLayoutVersion"`
}

// OCIVersion reads the OCI image layout version for this layout
func (l Layout) OCIVersion() (string, error) {
	buf, err := ioutil.ReadFile(filepath.Join(l.Root, l.Name, nameLayout))
	if err != nil {
		return "", err
	}

	var ociVersion ociImageLayoutVersion
	if err := json.Unmarshal(buf, &ociVersion); err != nil {
		return "", err
	}

	return ociVersion.ImageLayoutVersion, nil
}

// Refs gives the path to all regular files or symlinks in this layout's "refs" directory
func (l Layout) Refs() ([]string, error) {
	return findFilesOrSymlink(filepath.Join(l.Root, l.Name, nameRefs))
}

// Blobs gives the path to all regular files or symlinks in this layout's "blobs" directory
func (l Layout) Blobs() ([]string, error) {
	paths, err := findFilesOrSymlink(filepath.Join(l.Root, l.Name, nameBlobs))
	if err != nil {
		return nil, err
	}
	digests := []string{}
	for _, path := range paths {
		digest := pathToDigest(path)
		if digest == "" {
			continue
		}
		digests = append(digests, digest)
	}
	return digests, nil
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
