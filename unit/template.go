package unit

import (
	"html/template"

	"github.com/coreos/go-systemd/unit"
)

var DefaultConfig = []*unit.UnitOption{
	&unit.UnitOption{
		Section: "Unit",
		Name:    "Description",
		Value:   "OCI: %n",
	},
	&unit.UnitOption{
		Section: "Service",
		Name:    "Slice",
		Value:   "oci.slice",
	},
}

var shellExecTemplate = template.Must(template.New("shellExec").Parse(`/bin/sh -c "{{.}}"`))
