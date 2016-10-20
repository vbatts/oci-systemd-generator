package main

import (
	"crypto"
	_ "crypto/sha256"
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

// Extract is the extracted content of an OCI image reference.
type Extract struct {
	Root string
	Name string
}

// WalkForExtracts walks a rootpath looking for all directories that match an
// extracted OCI image reference.
func WalkForExtracts(rootpath string) (extracts []Extract, err error) {
	return nil, nil
}

// TODO Perhaps have an easy streamer for calculating the checksum of a stream
