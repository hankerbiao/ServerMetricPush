package main

import (
	"os"
	"testing"
)

func TestDefaultConfigPath(t *testing.T) {
	if defaultConfigPath != "./config.yml" {
		t.Fatalf("defaultConfigPath = %q, want %q", defaultConfigPath, "./config.yml")
	}
}

func TestEffectivePushInstance_PrefersConfiguredValue(t *testing.T) {
	if got := effectivePushInstance("custom-instance"); got != "custom-instance" {
		t.Fatalf("effectivePushInstance() = %q, want %q", got, "custom-instance")
	}
}

func TestEffectivePushInstance_FallsBackWhenEmpty(t *testing.T) {
	got := effectivePushInstance("")
	if got == "" {
		t.Fatal("effectivePushInstance() = empty, want detected ip or hostname")
	}
}

func TestNormalizeMachineID_TrimsWhitespace(t *testing.T) {
	got := normalizeMachineID("abc123\n")
	if got != "abc123" {
		t.Fatalf("normalizeMachineID() = %q, want %q", got, "abc123")
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
