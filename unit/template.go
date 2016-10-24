package unit

import (
	"fmt"
	"html/template"
	"strings"

	"github.com/coreos/go-systemd/unit"
)

// DefaultOptions includes the boiler plate options for generating OCI layout
// unit files.
var DefaultOptions = []*unit.UnitOption{
	&unit.UnitOption{Section: "Unit", Name: "Description", Value: "OCI: %n"},
	&unit.UnitOption{Section: "Service", Name: "Slice", Value: "oci.slice"},
	&unit.UnitOption{Section: "Service", Name: "PrivateTmp", Value: "yes"},
	&unit.UnitOption{Section: "Service", Name: "ProtectSystem", Value: "yes"},
	&unit.UnitOption{Section: "Service", Name: "ProtectHome", Value: "yes"},
	&unit.UnitOption{Section: "Service", Name: "Delegate", Value: "yes"},
	&unit.UnitOption{Section: "Service", Name: "DevicePolicy", Value: "closed"},
}

// ExecStart provides the unit file option for ExecStart=, given a command string
func ExecStart(cmd string) (*unit.UnitOption, error) {
	// if the command is not an abosulte path
	if !strings.HasPrefix(cmd, "/") {
		cmd = fmt.Sprintf(`/bin/sh -c %q`, cmd)
	}
	return unit.NewUnitOption("Service", "ExecStart", cmd), nil
}

var (
	shellExecTemplate = template.Must(template.New("shellExec").Parse(`/bin/sh -c "{{.}}"`))
)

/*
Needed:
from systmed.exec(5)
- RootDirectory=

*/
