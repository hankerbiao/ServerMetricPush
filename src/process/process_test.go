package process

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"
)

func TestResolvePath_UsesCurrentDirectoryBinary(t *testing.T) {
	wd := t.TempDir()
	t.Setenv("PATH", "")
	t.Chdir(wd)

	if err := createExecutable("node_exporter", "#!/bin/sh\nexit 0\n"); err != nil {
		t.Fatalf("createExecutable() error = %v", err)
	}

	got, err := resolvePath("node_exporter")
	if err != nil {
		t.Fatalf("resolvePath() error = %v", err)
	}

	if !strings.HasSuffix(got, "/node_exporter") {
		t.Fatalf("resolvePath() = %q, want path ending with /node_exporter", got)
	}
}

func TestStart_ReturnsErrorWhenProcessExitsImmediately(t *testing.T) {
	wd := t.TempDir()
	t.Setenv("PATH", "")
	t.Chdir(wd)

	script := "#!/bin/sh\n" +
		"echo 'mock node_exporter failed to start' >&2\n" +
		"exit 1\n"
	if err := createExecutable("node_exporter", script); err != nil {
		t.Fatalf("createExecutable() error = %v", err)
	}

	_, err := Start(Config{ExecutablePath: "node_exporter", Port: 9100})
	if err == nil {
		t.Fatal("Start() error = nil, want startup failure")
	}
	if !strings.Contains(err.Error(), "mock node_exporter failed to start") {
		t.Fatalf("Start() error = %q, want stderr output", err)
	}
}

func TestFormatError_IncludesStderr(t *testing.T) {
	stderr := bytes.NewBufferString("boom")

	err := formatError("prefix", errors.New("failed"), stderr)
	if err == nil {
		t.Fatal("formatError() error = nil, want wrapped error")
	}
	if !strings.Contains(err.Error(), "stderr: boom") {
		t.Fatalf("formatError() = %q, want stderr content", err)
	}
}

func createExecutable(name, content string) error {
	return os.WriteFile(name, []byte(content), 0o755)
}
