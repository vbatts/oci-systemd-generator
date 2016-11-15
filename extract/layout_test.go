package extract

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/opencontainers/image-spec/specs-go/v1"
)

func TestLayoutManifest(t *testing.T) {
	tmp, err := ioutil.TempDir("", "testing.")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)
	l := Layout{
		Root:     tmp,
		Name:     "example.com/test/myapp",
		HashName: DefaultHashName,
	}

	// 8fe85f31aa6fa5ff4e5cb7d03497e9a78c0f8492ec2da21279adedbc312976cf
	r := strings.NewReader("slartibartfast")
	if err := l.SetRefConfig("stable", r); err != nil {
		t.Fatal(err)
	}

	rh, err := l.GetRef("stable")
	if err != nil {
		t.Fatal(err)
	}

	fh := rh.(*os.File)
	manifestName, err := os.Readlink(fh.Name())
	if err != nil {
		t.Fatal(err)
	}
	expected := "8fe85f31aa6fa5ff4e5cb7d03497e9a78c0f8492ec2da21279adedbc312976cf"
	if filepath.Base(manifestName) != expected {
		t.Fatalf("expected ref to be at %q; but got %q", expected, filepath.Base(manifestName))
	}
}

func TestConfig(t *testing.T) {
	c := Config{
		ImageConfig: &v1.Image{},
	}

	cmd := c.ExecStart()

	// Test default
	expect := ""
	if cmd != expect {
		t.Errorf("expected %q; got %q", expect, cmd)
	}

	// test cmd, with absolute path
	c.ImageConfig.Config.Cmd = []string{"/usr/bin/tail", "-f", "/dev/null"}
	cmd = c.ExecStart()
	expect = `/usr/bin/tail -f /dev/null`
	if cmd != expect {
		t.Errorf("expected %q; got %q", expect, cmd)
	}

	// test cmd, with no absolute path
	c.ImageConfig.Config.Cmd = []string{"tail", "-f", "/dev/null"}
	cmd = c.ExecStart()
	expect = `/bin/sh -c "tail -f /dev/null"`
	if cmd != expect {
		t.Errorf("expected %q; got %q", expect, cmd)
	}

	// test Entrypoint, with no absolute path
	c.ImageConfig.Config.Entrypoint = []string{"tail", "-f", "/dev/null"}
	c.ImageConfig.Config.Cmd = nil
	cmd = c.ExecStart()
	expect = `/bin/sh -c "tail -f /dev/null"`
	if cmd != expect {
		t.Errorf("expected %q; got %q", expect, cmd)
	}

	// test Entrypoint, with absolute path
	c.ImageConfig.Config.Entrypoint = []string{"/usr/bin/tail"}
	c.ImageConfig.Config.Cmd = []string{"-f", "/dev/null"}
	cmd = c.ExecStart()
	expect = `/usr/bin/tail -f /dev/null`
	if cmd != expect {
		t.Errorf("expected %q; got %q", expect, cmd)
	}
}
