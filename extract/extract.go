package extract

import (
	"crypto"
	_ "crypto/sha256" // this is for the HashMap
	"errors"
	"fmt"
	"io"
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
          |- ref/
          |  |- stable -> ../../../manifest/sha256/ea/ea7beefea7beefd0ee7
          |- rootfs/ -> ../../dirs/chainID/sha256/ba/baabaab1acc24ee9/

Where the dir
*/
type Layout struct {
	Root string
	Name string
}

// WalkForExtracts walks a rootpath looking for all directories that match an
// extracted OCI image reference.
func WalkForExtracts(rootpath string) (extracts []Layout, err error) {
	return nil, nil
}

// TODO Perhaps have an easy streamer for calculating the checksum of a stream
