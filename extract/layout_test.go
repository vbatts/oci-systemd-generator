package extract

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	if err := l.SetRef("stable", r); err != nil {
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
