package config

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigLoad(t *testing.T) {
	cfg, err := LoadConfigFromOptions(strings.NewReader(DefaultConfig))
	if err != nil {
		t.Fatal(err)
	}

	expect := "/home/vbatts/oci/layouts"
	got := filepath.Clean(cfg.ImageLayoutDir)
	if got != expect {
		t.Errorf("expected %q; got %q", expect, got)
	}
	expect = "/home/vbatts/oci/extracts"
	got = filepath.Clean(cfg.ExtractsDir)
	if got != expect {
		t.Errorf("expected %q; got %q", expect, got)
	}
}
