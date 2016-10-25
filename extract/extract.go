package extract

import (
	"crypto"
	_ "crypto/sha256" // this is for the HashMap
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// SumContent calculates the hexidecimal digest of the content in r, with
// hashing functionality of hashName.
// `hashName` string will be referenced through HashMap for its hashing functionality.
func SumContent(hashName string, r io.Reader) (string, error) {
	hash, ok := HashMap[hashName]
	if !ok {
		return "", ErrNoHash
	}
	h := hash.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// ErrNoHash if the hashName provided does not exist
var ErrNoHash = errors.New("no such hash in HashMap")

// HashMap is the mapping between the string form, found in the digest
// "sha256:ea2bedaf251...", to the crypto.Hash that provides the hash.
var HashMap = map[string]crypto.Hash{
	"sha256": crypto.SHA256,
}

/*
Layout is the extracted content of an OCI image reference.

The attributes of an extracted image ref are:
- the "name" - derived of the relative path from presumably /var/lib/oci/imagelayouts/
- the ref name - derived from the `./refs/<name>` file

The ref is a descriptor pointing to a checksum of a manifest.  Multiple refs
may point to the same checksum, so citing this per the _checksum_ would be
cleaner, and then just symlink the <name> and <refname> to a checksummed
directory.

The /var/lib/oci/extract/ hierarchy is:
    |- dirs/
    |  |- chainID/
    |     |- sha256/
    |        |- ba/
    |           |- baabaab1acc24ee9/
    |- manifest/
    |  |- sha256/
    |     |- ea/
    |        |- ea7beefea7beefd0ee7
    |- names/
       |- example.com/myapp/
          |- refs/
          |  |- stable -> ../../../manifest/sha256/ea/ea7beefea7beefd0ee7
          |- rootfs/
             |- stable -> ../../../dirs/chainID/sha256/ba/baabaab1acc24ee9/

*/
type Layout struct {
	Root string
	Name string
}

// WalkForExtracts walks a rootpath looking for all directories that match an
// extracted OCI image reference.
func WalkForExtracts(rootpath string) (extracts []Layout, err error) {
	namespath := filepath.Join(rootpath, "names")
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
		case "refs":
			altDir = filepath.Join(dirname, "rootfs")
		case "rootfs":
			altDir = filepath.Join(dirname, "refs")
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

// Refs provides the names of the refs, which are themselves symlinks to the
// corresponding OCI manifest object.
//
// TODO this might better return structs, than just string list?
func (l Layout) Refs() ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(l.Root, "names", l.Name, "refs", "*"))
	if err != nil {
		return nil, err
	}
	refs := make([]string, len(matches))
	for i := range matches {
		refs[i] = filepath.Base(matches[i])
	}
	return refs, nil
}

// GetRef returns a handle to the ref to the OCI manifest. Caller is
// responsible for closing the handle.
func (l Layout) GetRef(ref string) (io.ReadCloser, error) {
	filename := filepath.Join(l.Root, "names", l.Name, "refs", ref)
	if _, err := os.Stat(filename); err != nil && os.IsNotExist(err) {
		return nil, err
	}
	return os.Open(filename)
}

// ErrNoExtracts is returned when the extracts root (`/var/lib/oci/extracts`)
// does not have the expected directories. This is true the first time any
// layouts are extracted.
var ErrNoExtracts = errors.New("extracts directory not populated")

func populateRootDir(rootpath string, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Join(rootpath, "dirs", "chainID"), perm); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(rootpath, "manifest"), perm); err != nil {
		return err
	}
	return os.MkdirAll(filepath.Join(rootpath, "names"), perm)
}

// TODO Perhaps have an easy streamer for calculating the checksum of a stream
