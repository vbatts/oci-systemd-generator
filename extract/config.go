package extract

import (
	"fmt"
	"strings"

	"github.com/opencontainers/image-spec/specs-go/v1"
)

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
