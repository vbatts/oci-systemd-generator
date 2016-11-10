package extract

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestRootDir(t *testing.T) {
	dir, err := ioutil.TempDir("", "test-rootdir.")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	if err := populateRootDir(dir, os.FileMode(0755)); err != nil {
		t.Fatal(err)
	}

	if err := os.RemoveAll(filepath.Join(dir, nameDirs)); err != nil {
		t.Fatal(err)
	}

	if err := populateRootDir(dir, os.FileMode(0755)); err != nil {
		t.Fatal(err)
	}
}
