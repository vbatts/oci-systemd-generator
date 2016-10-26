package extract

import (
	"errors"
	"os"
	"path/filepath"
)

// ErrNoExtracts is returned when the extracts root (`/var/lib/oci/extracts`)
// does not have the expected directories. This is true the first time any
// layouts are extracted.
var ErrNoExtracts = errors.New("extracts directory not populated")

func populateRootDir(rootpath string, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Join(rootpath, nameDirs, nameChainID), perm); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(rootpath, nameManifest), perm); err != nil {
		return err
	}
	return os.MkdirAll(filepath.Join(rootpath, nameNames), perm)
}

// TODO Perhaps have an easy streamer for calculating the checksum of a stream
