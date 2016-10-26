package extract

import (
	"io"
	"os"
	"path/filepath"
)

/*
Layout is the extracted content of an OCI image reference.

The attributes of an extracted image ref are:
- the "name" - derived of the relative path from presumably /var/lib/oci/imagelayouts/
- the ref name - derived from the `./<name>/ref` symlink
- the rootfs directory - derived from the `./<name>/rootfs symlink

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
          |- stable/
          |  |- ref -> ../../../manifest/sha256/ea/ea7beefea7beefd0ee7
          |  |- rootfs -> ../../../dirs/chainID/sha256/ba/baabaab1acc24ee9/
          |- v1.0.0/
             |- ref -> ../../../manifest/sha256/ea/ea7beefea7beefd0ee7
             |- rootfs -> ../../../dirs/chainID/sha256/ba/baabaab1acc24ee9/

*/
type Layout struct {
	Root string
	Name string
}

// Refs provides the names of the refs, which are themselves symlinks to the
// corresponding OCI manifest object.
//
// TODO this might better return structs, than just string list?
func (l Layout) Refs() ([]string, error) {
	// /var/lib/oci/extracts/names/example.com/myapp/stable/ref
	matches, err := filepath.Glob(l.refPath("*"))
	if err != nil {
		return nil, err
	}
	refs := make([]string, len(matches))
	for i := range matches {
		refs[i] = filepath.Base(filepath.Dir(matches[i]))
	}
	return refs, nil
}

// GetRef returns a handle to the ref to the OCI manifest. Caller is
// responsible for closing the handle.
func (l Layout) GetRef(ref string) (io.ReadCloser, error) {
	if _, err := os.Stat(l.refPath(ref)); err != nil && os.IsNotExist(err) {
		return nil, err
	}
	return os.Open(l.refPath(ref))
}

func (l Layout) refPath(ref string) string {
	return filepath.Join(l.Root, nameNames, l.Name, ref, nameRef)
}
