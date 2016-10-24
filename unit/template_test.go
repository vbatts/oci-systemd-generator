package unit

import (
	"bytes"
	"testing"
)

func TestShellExec(t *testing.T) {
	expect := `/bin/sh -c "tail -f /dev/null"`
	buf := bytes.NewBuffer([]byte{})
	if err := shellExecTemplate.Execute(buf, "tail -f /dev/null"); err != nil {
		t.Fatal(err)
	}

	got := string(buf.Bytes())
	if got != expect {
		t.Errorf("expected %q; got %q", expect, got)
	}
}

func TestExecStart(t *testing.T) {
	expect := `/bin/sh -c "tail -f /dev/null"`
	u, err := ExecStart(`tail -f /dev/null`)
	if err != nil {
		t.Fatal(err)
	}

	if u.Section != "Service" {
		t.Errorf("Expected unit option in Service; got %q", u.Section)
	}
	if u.Name != "ExecStart" {
		t.Errorf("Expected unit option of ExecStart; got %q", u.Name)
	}
	if u.Value != expect {
		t.Errorf("Expected unit value of %q; got %q", expect, u.Value)
	}

	expect = `/usr/bin/tail -f /dev/null`
	u, err = ExecStart(expect)
	if err != nil {
		t.Fatal(err)
	}

	if u.Section != "Service" {
		t.Errorf("Expected unit option in Service; got %q", u.Section)
	}
	if u.Name != "ExecStart" {
		t.Errorf("Expected unit option of ExecStart; got %q", u.Name)
	}
	if u.Value != expect {
		t.Errorf("Expected unit value of %q; got %q", expect, u.Value)
	}
}
