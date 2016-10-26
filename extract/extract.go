package extract

import (
	"errors"
	"os"
	"path/filepath"
	"vb/oci-systemd-generator/layout"
)

//var DefaultRootDir = RootDir{ Path: "/var/lib/oci" }

// NewRootDir produces a handler for the root directory of extracted OCI images
func NewRootDir(path string) (*RootDir, error) {
	// initial check of the directory
	if err := checkBasicRootDir(path); err != nil {
		if err != ErrNoExtracts {
			return nil, err
		}
		// otherwise it just needs to be populated
		if err := populateRootDir(path, os.FileMode(0755)); err != nil {
			return nil, err
		}
	}
	return &RootDir{Path: path}, nil
}

// RootDir is the base of where OCI images will be extracted to.
// Typically this will be /var/lib/oci/extracts
type RootDir struct {
	Path string
}

// Extract an OCI image layout
func (rd RootDir) Extract(il layout.Layout) (*Layout, error) {
	return nil, nil
}

// ErrNoExtracts is returned when the extracts root (`/var/lib/oci/extracts`)
// does not have the expected directories. This is true the first time any
// layouts are extracted.
var ErrNoExtracts = errors.New("extracts directory not populated")

func populateRootDir(rootpath string, perm os.FileMode) error {
	for _, path := range []string{nameNames, nameManifest, nameChainIDDir} {
		if err := os.MkdirAll(filepath.Join(rootpath, path), perm); err != nil {
			return err
		}
	}
	return nil
}

func checkBasicRootDir(rootpath string) error {
	for _, path := range []string{nameNames, nameManifest, nameChainIDDir} {
		_, err := os.Stat(filepath.Join(rootpath, path))
		if err != nil && os.IsNotExist(err) {
			return ErrNoExtracts
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// TODO Perhaps have an easy streamer for calculating the checksum of a stream
