package layout

import "github.com/opencontainers/image-spec/specs-go/v1"

// Manifest carries the layout and ref name, plus the full structure for the OCI image manifest
type Manifest struct {
	Layout   *Layout
	Ref      string
	Manifest *v1.Manifest
}
