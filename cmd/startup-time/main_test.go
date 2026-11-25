package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func withState(t *testing.T) string {
	t.Helper()
	old := stateFile
	stateFile = filepath.Join(t.TempDir(), "startup-time.state")
	t.Cleanup(func() { stateFile = old })
	return stateFile
}

func TestParseTime(t *testing.T) {
	got, err := parseTime("2030-01-02 03:04:05 UTC")
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	if _, err := parseTime("not a time at all"); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseTimeIgnoresCallerPath(t *testing.T) {
	// A fake "date" first in PATH must not be used.
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "date"), []byte("#!/bin/sh\necho 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+":"+os.Getenv("PATH"))
	got, err := parseTime("2030-01-01 00:00:00 UTC")
	if err != nil || got.Unix() == 1 {
		t.Fatalf("PATH hijack: got %v, %v", got, err)
	}
}

func TestSetRejectsPast(t *testing.T) {
	withState(t)
	if err := set("2001-01-01"); err == nil {
		t.Fatal("expected error for a past time")
	}
}

func TestSetWritesRFC3339AndReadsBack(t *testing.T) {
	p := withState(t)
	if err := set("now + 2 hours"); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(p)
	if _, err := time.Parse(time.RFC3339, strings.TrimSpace(string(data))); err != nil {
		t.Fatalf("state %q is not RFC3339: %v", data, err)
	}
	lock, err := readLock()
	if err != nil || time.Until(lock) < 110*time.Minute {
		t.Fatalf("readLock = %v, %v", lock, err)
	}
}

func TestNonRootCannotShorten(t *testing.T) {
	if isRoot() {
		t.Skip("running as root")
	}
	withState(t)
	if err := set("now + 3 hours"); err != nil {
		t.Fatal(err)
	}
	if err := set("now + 1 hour"); err == nil {
		t.Fatal("non-root shortened an active lock")
	}
	if err := set("now + 5 hours"); err != nil {
		t.Fatalf("extending should work: %v", err)
	}
	if err := clear(); err == nil {
		t.Fatal("non-root cleared the lock")
	}
}

func TestCorruptStateIsAnError(t *testing.T) {
	p := withState(t)
	os.WriteFile(p, []byte("garbage\n"), 0o644)
	if _, err := readLock(); err == nil {
		t.Fatal("expected error for corrupt state")
	}
}
