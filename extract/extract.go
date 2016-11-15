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
	"syscall"

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
	if err := el.SetRefConfig(m.Ref, configFH); err != nil && err != os.ErrExist {
		return nil, err
	}
	configFH.Close()

	// 3) apply the layers referenced to the layer's chanID dir
	// which will require marshalling the manifest to get the config object

	config, err := m.Config()
	if err != nil {
		return nil, err
	}
	chainIDRef, err := config.ChainID()
	if err != nil {
		return nil, err
	}
	destpath := el.chainIDPath(chainIDRef.HashName(), chainIDRef.Sum())
	if _, err := os.Stat(destpath); err != nil && os.IsNotExist(err) {
		if err := os.MkdirAll(destpath, os.FileMode(0755)); err != nil {
			return nil, fmt.Errorf("error preparing chainID dir for %s/%s: %s", chainIDRef.HashName(), chainIDRef.Sum(), err)
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

				util.Debugf("Applying %q to chainID %q", desc.Digest, chainIDRef.Name)
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
		util.Debugf("chainID %q already exists. Not applying.", chainIDRef.Name)
	}

	// 4) symlink to that chainID dir
	if _, err := os.Lstat(el.chainIDPath(chainIDRef.HashName(), chainIDRef.Sum())); err != nil && os.IsNotExist(err) {
		if err := os.Symlink(el.chainIDPath(chainIDRef.HashName(), chainIDRef.Sum()), el.rootfsPath(m.Ref)); err != nil {
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

	// for good measure
	destpath = filepath.Clean(destpath)

	// Both of the above checked mediatypes are gzip compressed.
	gz, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gz.Close()
	// gz is now the tar stream
	tr := tar.NewReader(gz)

	whiteouts := []string{}
	filepaths := map[string]interface{}{}
	for {
		hdr, err := tr.Next()
		if err != nil && err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		hdr.Name = filepath.Clean(hdr.Name)
		if escapedPath(destpath, hdr.Name) {
			util.Debugf("%q attempts to escape %q!! Skipping", hdr.Name, destpath)
			//return ErrPathEscapes
			continue
		}
		if hdr.Linkname != "" {
			linkpath := filepath.Join(filepath.Dir(hdr.Name), hdr.Linkname)
			if escapedPath(destpath, linkpath) {
				util.Debugf("%q -> %q attempts to escape %q!!", hdr.Name, hdr.Linkname, destpath)
				//return ErrPathEscapes

				// TODO experiment with this idea?
				//hdr.Linkname = "/dev/null"
				//hdr.Typeflag = tar.TypeSymlink
				// XXX this is not right. It can chmod /dev/null on the host?
				continue
			}
		}
		if strings.HasPrefix(filepath.Base(hdr.Name), whiteoutPrefix) {
			whiteouts = append(whiteouts, hdr.Name)
			// delete all at the whiteout path _except_ the filepaths that have
			// been extracted so far.
			pathToDelete := pathFromWhiteout(hdr.Name)
			if pathToDelete == "" {
				continue
			}
			err := filepath.Walk(filepath.Join(destpath, pathToDelete), func(p string, info os.FileInfo, err error) error {
				if err != nil {
					if os.IsNotExist(err) {
						return nil
					}
					return err
				}
				if _, exists := filepaths[p]; !exists {
					if err := os.RemoveAll(p); err != nil && !os.IsNotExist(err) {
						return err
					}
					//fmt.Printf("DELETE %q\n", p)
				}
				return nil
			})
			if err != nil {
				return err
			}

			continue
		}

		// First ensure that the directory of this entry exists
		dirpath := filepath.Join(destpath, filepath.Dir(hdr.Name))
		if _, err := os.Lstat(dirpath); err != nil && os.IsNotExist(err) {
			// Assuming a default mode. When/if this directories entry is encountered, we'll chmod it.
			if err := os.MkdirAll(dirpath, os.FileMode(0755)); err != nil {
				return err
			}
		}
		entrypath := filepath.Join(destpath, hdr.Name)
		switch hdr.Typeflag {
		case tar.TypeDir:
			if _, err := os.Lstat(entrypath); err != nil && os.IsNotExist(err) {
				// Assuming a default mode. When/if this directories entry is encountered, we'll chmod it.
				if err := os.MkdirAll(entrypath, os.FileMode(hdr.Mode)); err != nil {
					// should fail
					return err
				}
			}
		case tar.TypeFifo:
			if err := os.Remove(entrypath); err != nil && !os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "failed to remove %q\n", entrypath)
			}
			if syscall.Mkfifo(entrypath, uint32(hdr.Mode)); err != nil {
				// should fail
				return err
			}
		case tar.TypeChar:
			if err := os.Remove(entrypath); err != nil && !os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "failed to remove %q\n", entrypath)
			}
			// should not fail
			if err := syscall.Mknod(entrypath, syscall.S_IFCHR, mkdev(hdr.Devmajor, hdr.Devminor)); err != nil {
				fmt.Fprintf(os.Stderr, "%q failed to mknod: %s\n", entrypath, err)
			}
		case tar.TypeBlock:
			if err := os.Remove(entrypath); err != nil && !os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "failed to remove %q\n", entrypath)
			}
			// should not fail
			if err := syscall.Mknod(entrypath, syscall.S_IFBLK, mkdev(hdr.Devmajor, hdr.Devminor)); err != nil {
				fmt.Fprintf(os.Stderr, "%q failed to mknod: %s\n", entrypath, err)
			}
		case tar.TypeSymlink:
			if err := os.Remove(entrypath); err != nil && !os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "failed to remove %q\n", entrypath)
			}
			// should fail?
			if err := os.Symlink(hdr.Linkname, entrypath); err != nil {
				fmt.Fprintf(os.Stderr, "INFO: failed to symlink to %q: %s\n", hdr.Linkname, err)
			}
		case tar.TypeLink:
			if err := os.Remove(entrypath); err != nil && !os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "failed to remove %q\n", entrypath)
			}
			// should fail? or should just copy from the original?
			if err := os.Link(filepath.Join(destpath, hdr.Linkname), entrypath); err != nil {
				fmt.Fprintf(os.Stderr, "INFO: failed to link to %q: %s\n", hdr.Linkname, err)
			}
		case tar.TypeReg:
			if err := os.Remove(entrypath); err != nil && !os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "failed to remove %q\n", entrypath)
			}
			fh, err := os.Create(entrypath)
			if err != nil {
				return err
			}
			if _, err := io.Copy(fh, tr); err != nil {
				fh.Close()
				return err
			}
			fh.Close()
		default:
			util.Debugf("unknown tar type: %q", hdr.Typeflag)
		}
		// these are generic for all types except symlinks
		// most of these ought not be fatal, but log to stderr
		if info, err := os.Lstat(entrypath); err == nil && info.Mode()&os.ModeSymlink == 0 {
			if err := os.Chmod(entrypath, os.FileMode(hdr.Mode)); err != nil {
				fmt.Fprintf(os.Stderr, "INFO: failed to set mode: %s\n", err)
			}
			if err := os.Chown(entrypath, hdr.Uid, hdr.Gid); err != nil {
				fmt.Fprintf(os.Stderr, "INFO: failed to set owner: %s\n", err)
			}
			if err := os.Chtimes(entrypath, hdr.ModTime, hdr.ModTime); err != nil {
				fmt.Fprintf(os.Stderr, "INFO: failed to set times: %s\n", err)
			}
			for k, v := range hdr.Xattrs {
				if err := syscall.Setxattr(entrypath, k, []byte(v), 0); err != nil {
					fmt.Fprintf(os.Stderr, "%q failed to set xattr %q: %s\n", entrypath, k, err)
				}
			}
		}

		// whiteouts. I hate them. Since they are not ordered, and technically
		// apply to "lower layers", then effective they must be applied first,
		// regardless of when in this stream they show up.
		filepaths[hdr.Name] = nil
	}
	util.Debugf("  extracted %d files", len(filepaths))
	if len(whiteouts) > 0 {
		util.Debugf("   whiteouts: %q", whiteouts)
	}
	return nil
}

func mkdev(major, minor int64) int {
	return int(uint32(((minor & 0xfff00) << 12) | ((major & 0xfff) << 8) | (minor & 0xff)))
}

var whiteoutPrefix = ".wh."

func pathFromWhiteout(path string) string {
	if filepath.Base(path) == whiteoutPrefix+whiteoutPrefix+".opq" {
		return filepath.Dir(path)
	}

	// who knows what to do for .plnk and .aufs ?
	if strings.Contains(path, whiteoutPrefix+whiteoutPrefix) {
		return ""
	}

	// else should only be whiteoutPrefix+<filename>
	return filepath.Join(filepath.Dir(path), strings.TrimPrefix(filepath.Base(path), whiteoutPrefix))
}

// ErrPathEscapes is when a path in an archive is attempting to escape the destination path
var ErrPathEscapes = errors.New("path in archive attempts to escape root")

func escapedPath(rootpath, relpath string) bool {
	clean := filepath.Clean(filepath.Join(rootpath, relpath))
	return !strings.HasPrefix(clean, filepath.Clean(rootpath))
}

// TODO Perhaps have an easy streamer for calculating the checksum of a stream
