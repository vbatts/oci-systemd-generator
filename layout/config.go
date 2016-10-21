package layout

import "github.com/opencontainers/image-spec/specs-go/v1"

// Config carries the layout and ref name, plus the full structure for the OCI
// image manifest
type Config struct {
	Layout      *Layout   // the layout that references this image config
	Ref         string    // the reference name that references this image config
	Manifest    *Manifest // the manifest that references this image config
	ImageConfig *v1.Image // the actual OCI image config
}
