package main

import (
	"io"

	"github.com/coreos/go-systemd/unit"
)

// OCIGenConfig is the configurations for generating systemd unit files from OCI image layouts
type OCIGenConfig struct {
	ImageLayoutDir string
	ExtractDir     string
}

// LoadConfigFromOptions reads from an INI style set of options
func LoadConfigFromOptions(r io.Reader) (*OCIGenConfig, error) {
	options, err := unit.Deserialize(r)
	if err != nil {
		return nil, err
	}
	cfg := OCIGenConfig{}
	for _, opt := range options {
		if opt.Section == "system" {
			switch opt.Name {
			case "imagelayoutdir":
				cfg.ImageLayoutDir = opt.Value
			case "extractdir":
				cfg.ExtractDir = opt.Value
			}
		}
	}

	return &cfg, nil
}
