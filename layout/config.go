package layout

import (
	"crypto/sha256"
	"fmt"
	"strings"

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
