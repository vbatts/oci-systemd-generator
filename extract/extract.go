package extract

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/opencontainers/image-spec/specs-go/v1"
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

// Extract an OCI image manifest and its layers to the provided rootpath directory
func Extract(rootpath string, m *layout.Manifest) (*Layout, error) {
	// 1) mkdir for the layout name, and ref name
	if m.Layout.Name == "" {
		return nil, fmt.Errorf("image layout name cannot be empty")
	}
	el := Layout{
		Root:     rootpath,
		Name:     m.Layout.Name,
		HashName: DefaultHashName,
	}
	if err := os.MkdirAll(el.refPath(m.Ref), os.FileMode(0755)); err != nil {
		return nil, fmt.Errorf("error preparing %s/%s: %s", el.Name, m.Ref, err)
	}

	// 2) copy over the manifest's config to nameConfigs dir
	// This first ref is a descriptor to a manifest
	configFH, err := m.ConfigReader()
	if err != nil {
		return nil, err
	}
	if err := el.SetRef(m.Ref, configFH); err != nil {
		return nil, err
	}
	configFH.Close()

	// 3) apply the layers referenced to the layer's chanID dir
	// which will require marshalling the manifest to get the config object

	// XXX

	// 4) symlink to that chainID dir

	// XXX

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

// ApplyImageLayer extracts the typed stream to destpath.
// For OCI image layer, this means accommodating the whiteout file entries as well.
// When applying uid/gid, it will attempt to chown the file if EPERM will default to current uid/gid.
func ApplyImageLayer(mediatype string, r io.Reader, destpath string) error {
	if mediatype != v1.MediaTypeImageLayer || mediatype != v1.MediaTypeImageLayerNonDistributable {
		return layout.ErrUnsupportedMediaType
	}
	// XXX
	return nil
}

// TODO Perhaps have an easy streamer for calculating the checksum of a stream
