package extract

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/vbatts/oci-systemd-generator/layout"
	"github.com/vbatts/oci-systemd-generator/util"
)

//var DefaultRootDir = RootDir{ Path: "/var/lib/oci" }

// NewRootDir produces a handler for the root directory of extracted OCI images
func NewRootDir(path string) (*RootDir, error) {
	// otherwise it just needs to be populated
	if err := populateRootDir(path, os.FileMode(0755)); err != nil {
		return nil, err
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
	_, err := os.Stat(filepath.Dir(el.refPath(m.Ref)))
	if err != nil && os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(el.refPath(m.Ref)), os.FileMode(0755)); err != nil {
			return nil, fmt.Errorf("error preparing %s/%s: %s", el.Name, m.Ref, err)
		}
	}
	// 2) copy over the manifest's config to nameConfigs dir
	// This first ref is a descriptor to a manifest
	configFH, err := m.ConfigReader()
	if err != nil {
		return nil, err
	}
	if err := el.SetRef(m.Ref, configFH); err != nil && err != os.ErrExist {
		return nil, err
	}
	configFH.Close()

	// 3) apply the layers referenced to the layer's chanID dir
	// which will require marshalling the manifest to get the config object

	config, err := m.Config()
	if err != nil {
		return nil, err
	}
	ref, err := config.ChainID()
	if err != nil {
		return nil, err
	}
	destpath := el.chainIDPath(ref.HashName(), ref.Sum())
	if _, err := os.Stat(destpath); err != nil && os.IsNotExist(err) {
		if err := os.MkdirAll(destpath, os.FileMode(0755)); err != nil {
			return nil, fmt.Errorf("error preparing chainID dir for %s/%s: %s", ref.HashName(), ref.Sum(), err)
		}
		// ugh, here we'll have to access the objects in order from the manifest, but
		// only when they're the right media type.
		// also, for correctness, they'll have to cross-reference the checksum of each
		// _uncompressed_ layer against the m.Layout.ImageConfig.RootFS.DiffIDs
		// XXX
		for _, desc := range m.Manifest.Layers {
			err := func() error {
				brdr, err := m.Layout.GetBlob(layout.DigestRef{Name: desc.Digest, Layout: m.Layout})
				if err != nil {
					return err
				}
				defer brdr.Close()

				util.Debugf("Extracting %q", desc.Digest)
				err = ApplyImageLayer(destpath, desc.MediaType, brdr)
				if err != nil && err == layout.ErrUnsupportedMediaType {
					util.Debugf("%q is unsupported. Skipping...", desc.MediaType)
				} else if err != nil {
					if err == ErrPathEscapes {
						util.Debugf(" removing %q", destpath)
						os.RemoveAll(destpath)
					}
					return err
				}

				return nil
			}()
			if err != nil {
				return nil, err
			}
		}
	} else {
		util.Debugf("%q already exists. Not extracting.", destpath)
	}

	// 4) symlink to that chainID dir
	if _, err := os.Lstat(el.chainIDPath(ref.HashName(), ref.Sum())); err != nil && os.IsNotExist(err) {
		if err := os.Symlink(el.chainIDPath(ref.HashName(), ref.Sum()), el.rootfsPath(m.Ref)); err != nil {
			return nil, err
		}
	}
	return &el, nil
}

// ErrNoExtracts is returned when the extracts root (`/var/lib/oci/extracts`)
// does not have the expected directories. This is true the first time any
// layouts are extracted.
var ErrNoExtracts = errors.New("extracts directory not populated")

func populateRootDir(rootpath string, perm os.FileMode) error {
	for _, path := range []string{nameNames, nameConfigs, nameChainIDDir} {
		if _, err := os.Stat(filepath.Join(rootpath, path)); err != nil && os.IsNotExist(err) {
			if err := os.MkdirAll(filepath.Join(rootpath, path), perm); err != nil {
				return err
			}
		}
	}
	return nil
}

// ApplyImageLayer extracts the typed stream to destpath.
// For OCI image layer, this means accommodating the whiteout file entries as well.
// When applying uid/gid, it will attempt to chown the file if EPERM will default to current uid/gid.
func ApplyImageLayer(destpath string, mediatype string, r io.Reader) error {
	if mediatype != v1.MediaTypeImageLayer && mediatype != v1.MediaTypeImageLayerNonDistributable {
		return layout.ErrUnsupportedMediaType
	}
	// Both of the above checked mediatypes are gzip compressed.
	gz, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gz.Close()
	// gz is now the tar stream
	tr := tar.NewReader(gz)

	var sum int64
	for {
		hdr, err := tr.Next()
		if err != nil && err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		if escapedPath(destpath, hdr.Name) {
			util.Debugf("%q attempts to escape %q!!", hdr.Name, destpath)
			return ErrPathEscapes
		}
		if hdr.Linkname != "" {
			linkpath := filepath.Join(filepath.Dir(hdr.Name), hdr.Linkname)
			if escapedPath(destpath, linkpath) {
				util.Debugf("%q -> %q attempts to escape %q!!", hdr.Name, hdr.Linkname, destpath)
				return ErrPathEscapes
			}
		}

		// XXX whiteouts
		// XXX actually extract the hdr entry

		sum++
	}
	util.Debugf("  extracted %d files", sum)
	return nil
}

// ErrPathEscapes is when a path in an archive is attempting to escape the destination path
var ErrPathEscapes = errors.New("path in archive attempts to escape root")

func escapedPath(rootpath, relpath string) bool {
	clean := filepath.Clean(filepath.Join(rootpath, relpath))
	return !strings.HasPrefix(clean, filepath.Clean(rootpath))
}

// TODO Perhaps have an easy streamer for calculating the checksum of a stream
