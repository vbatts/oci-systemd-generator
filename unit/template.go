package unit

import "github.com/coreos/go-systemd/unit"

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
