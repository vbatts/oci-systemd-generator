package layout

import (
	"crypto/sha256"
	"fmt"

	"github.com/opencontainers/image-spec/specs-go/v1"
)

// Config carries the layout and ref name, plus the full structure for the OCI
// image manifest
type Config struct {
	Layout      *Layout   // the layout that references this image config
	Ref         string    // the reference name that references this image config
	Manifest    *Manifest // the manifest that references this image config
	ImageConfig *v1.Image // the actual OCI image config
}

func (c Config) diffIDsDigests() []DigestRef {
	if c.ImageConfig.RootFS.Type != "layers" {
		return nil
	}
	dLen := len(c.ImageConfig.RootFS.DiffIDs)
	if dLen == 0 {
		return nil
	}
	digestRefs := make([]DigestRef, dLen)
	for i, diffid := range c.ImageConfig.RootFS.DiffIDs {
		digestRefs[i] = DigestRef{Name: diffid, Layout: c.Layout}
	}
	return digestRefs
}

// ChainID is the calculated identification of the culmination of an OCI image config's diff_ids.
// See https://github.com/opencontainers/image-spec/blob/master/config.md#layer-chainid
func (c Config) ChainID() (*DigestRef, error) {
	if c.ImageConfig == nil {
		return nil, fmt.Errorf("no ImageConfig present")
	}
	digestRefs := c.diffIDsDigests()
	if digestRefs == nil {
		return nil, fmt.Errorf("failed to get the diff_ids")
	}

	return chainID(nil, digestRefs...), nil
}

func chainID(prev *DigestRef, diffIDs ...DigestRef) *DigestRef {
	if diffIDs == nil || len(diffIDs) == 0 {
		return prev
	}
	if prev == nil {
		return chainID(&diffIDs[0], diffIDs[1:]...)
	}
	sum := sha256.Sum256([]byte(prev.Name + " " + diffIDs[0].Name))
	return chainID(&DigestRef{Name: fmt.Sprintf("sha256:%x", sum), Layout: prev.Layout}, diffIDs[1:]...)
}
