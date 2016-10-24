package layout

import (
	"testing"

	"github.com/opencontainers/image-spec/specs-go/v1"
)

func TestConfig(t *testing.T) {
	c := Config{
		ImageConfig: &v1.Image{},
	}

	cmd, err := c.ExecStart()
	if err != nil {
		t.Fatal(err)
	}

	// Test default
	expect := "/sbin/init"
	if cmd != expect {
		t.Errorf("expected %q; got %q", expect, cmd)
	}

	// test cmd, with absolute path
	c.ImageConfig.Config.Cmd = []string{"/usr/bin/tail", "-f", "/dev/null"}
	cmd, err = c.ExecStart()
	if err != nil {
		t.Fatal(err)
	}
	expect = `/usr/bin/tail -f /dev/null`
	if cmd != expect {
		t.Errorf("expected %q; got %q", expect, cmd)
	}

	// test cmd, with no absolute path
	c.ImageConfig.Config.Cmd = []string{"tail", "-f", "/dev/null"}
	cmd, err = c.ExecStart()
	if err != nil {
		t.Fatal(err)
	}
	expect = `/bin/sh -c "tail -f /dev/null"`
	if cmd != expect {
		t.Errorf("expected %q; got %q", expect, cmd)
	}

	// test Entrypoint, with no absolute path
	c.ImageConfig.Config.Entrypoint = []string{"tail", "-f", "/dev/null"}
	c.ImageConfig.Config.Cmd = nil
	cmd, err = c.ExecStart()
	if err != nil {
		t.Fatal(err)
	}
	expect = `/bin/sh -c "tail -f /dev/null"`
	if cmd != expect {
		t.Errorf("expected %q; got %q", expect, cmd)
	}

	// test Entrypoint, with absolute path
	c.ImageConfig.Config.Entrypoint = []string{"/usr/bin/tail"}
	c.ImageConfig.Config.Cmd = []string{"-f", "/dev/null"}
	cmd, err = c.ExecStart()
	if err != nil {
		t.Fatal(err)
	}
	expect = `/usr/bin/tail -f /dev/null`
	if cmd != expect {
		t.Errorf("expected %q; got %q", expect, cmd)
	}
}
