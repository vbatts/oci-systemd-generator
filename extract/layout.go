package extract

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/vbatts/oci-systemd-generator/util"
)

/*
Layout is the extracted content of an OCI image reference.

The attributes of an extracted image ref are:
- the "name" - derived of the relative path from presumably /var/lib/oci/imagelayouts/
- the ref name - derived from the `./<name>/ref` symlink
- the rootfs directory - derived from the `./<name>/rootfs symlink

The config is a descriptor pointing to a checksum of a config.  Multiple refs
may point to the same checksum, so citing this per the _checksum_ would be
cleaner, and then just symlink the <name> and <refname> to a checksummed
directory.

The /var/lib/oci/extract/ hierarchy is:
    |- dirs/
    |  |- chainID/
    |     |- sha256/
    |        |- ba/
    |           |- baabaab1acc24ee9/
    |- configs/
    |  |- sha256/
    |     |- ea/
    |        |- ea7beefea7beefd0ee7
    |- names/
       |- example.com/myapp/
          |- stable/
          |  |- config -> ../../../configs/sha256/ea/ea7beefea7beefd0ee7
          |  |- rootfs -> ../../../dirs/chainID/sha256/ba/baabaab1acc24ee9/
          |- v1.0.0/
             |- config -> ../../../configs/sha256/ea/ea7beefea7beefd0ee7
             |- rootfs -> ../../../dirs/chainID/sha256/ba/baabaab1acc24ee9/

*/
type Layout struct {
	Root     string
	Name     string
	HashName string
}

// DefaultHashName is the name of the hash to use for the calculating the
// objects in layouts.
//
// See util.HashMap.
var DefaultHashName = "sha256"

// Refs provides the refs, which contain the symlinks to the corresponding OCI
// image config object and root filesystem.
func (l *Layout) Refs() ([]*Ref, error) {
	// /var/lib/oci/extracts/names/example.com/myapp/stable/ref
	matches, err := filepath.Glob(l.refPath("*"))
	if err != nil {
		return nil, err
	}
	refs := make([]*Ref, len(matches))
	for i := range matches {
		refs[i] = &Ref{
			Name:   filepath.Base(filepath.Dir(matches[i])),
			Layout: l,
		}
	}
	return refs, nil
}

// GetRef returns a handle to the ref to the OCI image config.
// Caller is responsible for closing the handle.
func (l Layout) GetRef(name string) (io.ReadCloser, error) {
	if _, err := os.Stat(l.refPath(name)); err != nil && os.IsNotExist(err) {
		return nil, err
	}
	return os.Open(l.refPath(name))
}

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

// Config is the extracted representation of the OCI image
type Config struct {
	Layout      *Layout
	Ref         *Ref
	ImageConfig *v1.Image // the actual OCI image config
}

// ExecStart provides the command to be executed, like on the ExecStart= option of a systemd unit file.
func (c Config) ExecStart() string {
	if c.ImageConfig == nil {
		return ""
	}
	// TODO it may be interesting to instead have an annotation, like com.example.systemd.unit.service.execstart=

	cmd := []string{}
	if c.ImageConfig.Config.Entrypoint != nil || len(c.ImageConfig.Config.Entrypoint) > 0 {
		cmd = append(cmd, c.ImageConfig.Config.Entrypoint...)
	}
	if c.ImageConfig.Config.Cmd != nil || len(c.ImageConfig.Config.Cmd) > 0 {
		cmd = append(cmd, c.ImageConfig.Config.Cmd...)
	}

	// c.ImageConfig.Config.Entrypoint
	// c.ImageConfig.Config.Cmd
	// If Entrypoint is set, it is first, and Cmd is appended as args
	// If Entrypoint is "", then Cmd is the exec
	// if the result is not absolute, then it needs a shell exec (`/bin/sh -c "args"`) (check for '/bin/sh' existance first?)

	if cmd == nil || len(cmd) == 0 {
		return ""
	}

	// if the command is not an absolute path
	if !strings.HasPrefix(cmd[0], "/") {
		return fmt.Sprintf(`/bin/sh -c %q`, strings.Join(cmd, " "))
	}

	return strings.Join(cmd, " ")
}

// SetRefConfig for name `ref` takes a reader. Reader `r` is read and written to it's
// content addressed mapping, and a symbolic link for `ref` is created pointing
// to this content addressed data.
func (l Layout) SetRefConfig(refname string, r io.Reader) error {
	// using Stat to follow symlink
	if _, err := os.Stat(l.refPath(refname)); err == nil {
		return os.ErrExist
	}
	if _, ok := util.HashMap[l.HashName]; !ok {
		return fmt.Errorf("HashName does not exist: %q", l.HashName)
	}

	tmp, err := l.tmpPath()
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	fh, err := ioutil.TempFile(tmp, "extract-layout.")
	if err != nil {
		return err
	}

	h := util.HashMap[l.HashName].New()
	tr := io.TeeReader(r, h)

	if _, err := io.Copy(fh, tr); err != nil {
		fh.Close()
		return err
	}
	if err := fh.Close(); err != nil {
		return err
	}

	dest := l.configPath(l.HashName, fmt.Sprintf("%x", h.Sum(nil)))
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}
	if err := os.Rename(fh.Name(), dest); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(l.refPath(refname)), 0755); err != nil {
		return err
	}
	// TODO rather than explicit paths, this would be nice to symlink to
	// ../../../configs/sha256/fc/fc744f333c05dd872a52509e0e4b9da7eed14d7a4d7df1f21fb6f2f3b16d31b4
	if _, err := os.Lstat(l.refPath(refname)); err != nil && os.IsNotExist(err) {
		if err := os.Symlink(dest, l.refPath(refname)); err != nil {
			return err
		}
	}
	return nil
}

func (l Layout) tmpPath() (string, error) {
	if err := os.MkdirAll(filepath.Join(l.Root, "tmp"), 0700); err != nil {
		return "", err
	}
	return ioutil.TempDir(filepath.Join(l.Root, "tmp"), "tmp")
}

func (l Layout) chainIDPath(hashName, sum string) string {
	return filepath.Join(l.Root, nameChainIDDir, hashName, sum[0:2], sum)
}
func (l Layout) configPath(hashName, sum string) string {
	return filepath.Join(l.Root, nameConfigs, hashName, sum[0:2], sum)
}
func (l Layout) rootfsPath(ref string) string {
	return filepath.Join(l.Root, nameNames, l.Name, ref, nameRootfs)
}
func (l Layout) refPath(ref string) string {
	return filepath.Join(l.Root, nameNames, l.Name, ref, nameRef)
}
