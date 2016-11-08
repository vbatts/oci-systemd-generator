package extract

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/vbatts/oci-systemd-generator/layout"
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
	// 1) mkdir for the layout name, and ref name
	if il.Name == "" {
		return nil, fmt.Errorf("image layout name cannot be empty")
	}
	el := Layout{
		Root: rd.Path,
		Name: il.Name,
	}
	refs, err := il.Refs()
	if err != nil {
		return nil, err
	}
	for _, ref := range refs {
		if err := os.MkdirAll(el.refPath(ref), os.FileMode(0755)); err != nil {
			return nil, fmt.Errorf("error preparing %s/%s: %s", il.Name, ref, err)
		}

		// 2) copy over the manifest's config to nameConfigs dir
		// This first ref is a descriptor to a manifest
		desc, err := il.GetRef(ref)
		if err != nil {
			return nil, fmt.Errorf("failed getting descriptor for %q", ref)
		}
		manifest, err := layout.ManifestFromDescriptor(&il, desc)
		if err != nil {
			return nil, err
		}
		configFH, err := manifest.ConfigReader()
		if err != nil {
			return nil, err
		}
		if err := el.SetRef(ref, configFH); err != nil {
			return nil, err
		}
		configFH.Close()
	}

	// 3) apply the layers referenced to the layer's chanID dir
	// which will require marshalling the manifest to get the config object

	// 4) symlink to that chainID dir
	return &el, nil
}

// ErrNoExtracts is returned when the extracts root (`/var/lib/oci/extracts`)
// does not have the expected directories. This is true the first time any
// layouts are extracted.
var ErrNoExtracts = errors.New("extracts directory not populated")

func populateRootDir(rootpath string, perm os.FileMode) error {
	for _, path := range []string{nameNames, nameConfigs, nameChainIDDir} {
		if err := os.MkdirAll(filepath.Join(rootpath, path), perm); err != nil {
			return err
		}
	}
	return nil
}

func checkBasicRootDir(rootpath string) error {
	for _, path := range []string{nameNames, nameConfigs, nameChainIDDir} {
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
