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
