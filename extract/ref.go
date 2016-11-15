package extract

import (
	"encoding/json"
	"io"
	"os"

	"github.com/opencontainers/image-spec/specs-go/v1"
)

// Ref is the ref of an extracted OCI image layout.
// It consists primarly of a config and rootfs.
type Ref struct {
	Name   string
	Layout *Layout
}

// ConfigReader provides a file handle to the raw OCI image config for this
// extracted layout.
func (r Ref) ConfigReader() (io.ReadCloser, error) {
	if _, err := os.Stat(r.Layout.refPath(r.Name)); err != nil && os.IsNotExist(err) {
		return nil, err
	}
	return os.Open(r.Layout.refPath(r.Name))
}

// Config parses the OCI image config for this particular layout reference
func (r *Ref) Config() (*Config, error) {
	configFH, err := r.ConfigReader()
	if err != nil {
		return nil, err
	}
	defer configFH.Close()
	dec := json.NewDecoder(configFH)
	imageConfig := &v1.Image{}
	if err := dec.Decode(imageConfig); err != nil {
		return nil, err
	}
	return &Config{Layout: r.Layout, Ref: r, ImageConfig: imageConfig}, nil
}

// RootFS provides the path to this extracted image's root filesystem (at least
// the symlink to the path).
func (r Ref) RootFS() (string, error) {
	return r.Layout.rootfsPath(r.Name), nil
}
