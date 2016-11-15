package extract

import (
	"encoding/json"
	"io"
	"os"
	"strings"

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
func (r Ref) RootFS() string {
	return r.Layout.rootfsPath(r.Name)
}

// ReverseDomainNotation provides a name for this extracted OCI image based on
// the relative path name of the OCI image layout, and the reference
// (`./refs/`) name.
// In the format `$reversedomain.$path.ref.$ref` with the literal word "ref"
// before the reference name.
func (r Ref) ReverseDomainNotation() string {
	var basename, path string
	if strings.Contains(r.Layout.Name, pathDelimiter) {
		parts := strings.SplitN(r.Layout.Name, pathDelimiter, 2)
		basename, path = parts[0], parts[1]
	} else {
		basename = r.Layout.Name
	}
	if strings.Contains(basename, domainDelimiter) {
		chunks := strings.Split(basename, domainDelimiter)
		for i := 0; i < len(chunks)/2; i++ {
			end := len(chunks) - 1
			chunks[i], chunks[end-i] = chunks[end-i], chunks[i]
		}
		basename = strings.Join(chunks, domainDelimiter)
	}
	path = strings.Replace(path, pathDelimiter, domainDelimiter, -1)
	return strings.Join([]string{basename, path, "ref", r.Name}, domainDelimiter)
}

var (
	domainDelimiter = "."
	pathDelimiter   = "/"
)
