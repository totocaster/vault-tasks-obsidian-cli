package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestRunVersionCommand(t *testing.T) {
	originalVersion, originalCommit, originalDate := version, commit, date
	version = "1.2.3"
	commit = "abc123"
	date = "2026-04-08T12:00:00Z"
	defer func() {
		version, commit, date = originalVersion, originalCommit, originalDate
	}()

	output := captureStdout(t, func() {
		if err := run([]string{"version"}); err != nil {
			t.Fatalf("run version: %v", err)
		}
	})

	if !strings.Contains(output, "vault-tasks 1.2.3") {
		t.Fatalf("unexpected version output: %q", output)
	}
}

func TestRunVersionFlag(t *testing.T) {
	output := captureStdout(t, func() {
		if err := run([]string{"--version"}); err != nil {
			t.Fatalf("run --version: %v", err)
		}
	})

	if !strings.Contains(output, "vault-tasks ") {
		t.Fatalf("unexpected version flag output: %q", output)
	}
}

func TestRunHelpCommand(t *testing.T) {
	output := captureStdout(t, func() {
		if err := run([]string{"help"}); err != nil {
			t.Fatalf("run help: %v", err)
		}
	})

	if !strings.Contains(output, "vault-tasks version") {
		t.Fatalf("expected help to mention version command: %q", output)
	}
}

func TestRunUnknownCommand(t *testing.T) {
	err := run([]string{"nope"})
	if err == nil {
		t.Fatal("expected unknown command error")
	}
	if !strings.Contains(err.Error(), `unknown command "nope"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	originalStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}

	os.Stdout = writer
	defer func() {
		os.Stdout = originalStdout
	}()

	fn()

	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close: %v", err)
	}

	var buffer bytes.Buffer
	if _, err := io.Copy(&buffer, reader); err != nil {
		t.Fatalf("io.Copy: %v", err)
	}
	if err := reader.Close(); err != nil {
		t.Fatalf("reader.Close: %v", err)
	}

	return buffer.String()
}
