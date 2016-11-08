package layout

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/opencontainers/image-spec/specs-go/v1"
)

// Manifest carries the layout and ref name, plus the full structure for the
// OCI image manifest
type Manifest struct {
	Layout   *Layout
	Ref      string
	Manifest *v1.Manifest
}

// ConfigReader gives access to the raw body of the config for this manifest.
// The caller is responsible to close the io.ReadCloser
func (m Manifest) ConfigReader() (io.ReadCloser, error) {
	if m.Manifest.Config.MediaType != v1.MediaTypeImageConfig {
		return nil, fmt.Errorf("expected %q; got %q", v1.MediaTypeImageConfig, m.Manifest.Config.MediaType)
	}
	digestRef := DigestRef{Name: m.Manifest.Config.Digest}
	return m.Layout.GetBlob(digestRef)
}

// Config provides the structure for this particular view of this layout
// reference of the config
func (m *Manifest) Config() (*Config, error) {
	configFH, err := m.ConfigReader()
	if err != nil {
		return nil, err
	}
	defer configFH.Close()
	dec := json.NewDecoder(configFH)
	var imageConfig *v1.Image
	if err := dec.Decode(imageConfig); err != nil {
		return nil, err
	}

	config := Config{
		Manifest:    m,
		Layout:      m.Layout,
		Ref:         m.Ref,
		ImageConfig: imageConfig,
	}
	return &config, nil
}

// Some common errors
var (
	ErrObjectNil            = fmt.Errorf("object is nil")
	ErrUnsupportedMediaType = fmt.Errorf("unsupported Medatype")
)

// ManifestFromDescriptor simplifies the reaching of manifest as the
// descriptors are accessed
func ManifestFromDescriptor(l *Layout, d *v1.Descriptor) (*Manifest, error) {
	if l == nil || d == nil {
		return nil, ErrObjectNil
	}
	if d.MediaType != v1.MediaTypeImageManifest && d.MediaType != v1.MediaTypeImageManifestList {
		return nil, ErrUnsupportedMediaType
	}
	if d.MediaType == v1.MediaTypeImageManifestList {
		return nil, fmt.Errorf("TODO: add support for manifest list")
	}
	manifestFH, err := l.GetBlob(DigestRef{Name: d.Digest})
	if err != nil {
		return nil, err
	}
	defer manifestFH.Close()
	manifest := v1.Manifest{}
	dec := json.NewDecoder(manifestFH)
	if err := dec.Decode(&manifest); err != nil {
		return nil, err
	}
	return &Manifest{Layout: l, Manifest: &manifest}, nil
}
