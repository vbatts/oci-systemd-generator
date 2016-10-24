package layout

import (
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
func (c Config) ExecStart() (string, error) {
	if c.ImageConfig == nil {
		return "", fmt.Errorf("Config: no ImageConfig present")
	}

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
	// if neither are set, then `/sbin/init` (check for '/sbin/init' existance first?)

	if cmd == nil || len(cmd) == 0 {
		return "/sbin/init", nil
	}

	// if the command is not an absolute path
	if !strings.HasPrefix(cmd[0], "/") {
		return fmt.Sprintf(`/bin/sh -c %q`, strings.Join(cmd, " ")), nil
	}

	return strings.Join(cmd, " "), nil
}
