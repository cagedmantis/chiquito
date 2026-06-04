package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersionFlag(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := run([]string{"-version"}, &out, &errBuf)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !strings.Contains(out.String(), "chiquito") {
		t.Errorf("version output = %q", out.String())
	}
}

func TestHelpFlag(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := run([]string{"-help"}, &out, &errBuf)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	// Usage goes to the error writer; it should mention key facts.
	u := errBuf.String()
	for _, want := range []string{"Usage", "C-x C-s", "config.toml"} {
		if !strings.Contains(u, want) {
			t.Errorf("usage missing %q:\n%s", want, u)
		}
	}
}

func TestUnknownFlag(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := run([]string{"-nope"}, &out, &errBuf)
	if code != 2 {
		t.Errorf("exit code = %d, want 2 for unknown flag", code)
	}
}
