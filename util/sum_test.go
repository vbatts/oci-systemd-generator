package util

import (
	"os"
	"testing"
)

func TestSum(t *testing.T) {
	expect := "ad629855562843eab84ef000f05d4d801774c48a7996dd5b38d03c5b0c0e9e72"
	fh, err := os.Open("./testdata/rando.img")
	if err != nil {
		t.Fatal(err)
	}

	got, err := SumContent("sha256", fh)
	if err != nil {
		t.Fatal(err)
	}

	if got != expect {
		t.Errorf("expected %q; got %q", expect, got)
	}
}
