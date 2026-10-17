package system

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseStatus(t *testing.T) {
	input := `
	Name: test-process
	State: R (running)
	Pid: 1234
	PPid: 5678
	VmRSS: 2048 kB
	`

	expected := Process{
		Name:  "test-process",
		State: "R (running)",
		PID:   1234,
		PPid:  5678,
		RSSKB: 2048,
	}
	got, err := parseStatus(strings.NewReader(input), 1234)
	if err != nil {
		t.Fatalf("parseStatus() error = %v", err)
	}
	if got.PID != expected.PID {
		t.Errorf("parseStatus() got = %v, want %v", got, expected)
	}

	if got.Name != expected.Name {
		t.Errorf("parseStatus() got = %v, want %v", got, expected)
	}

	if got.State != expected.State {
		t.Errorf("parseStatus() got = %v, want %v", got, expected)
	}

	if got.PPid != expected.PPid {
		t.Errorf("parseStatus() got = %v, want %v", got, expected)
	}

	if got.RSSKB != expected.RSSKB {
		t.Errorf("parseStatus() got = %v, want %v", got, expected)
	}
}

func TestParseStatusZeroRSS(t *testing.T) {
	input := "PPid: 201\nName: worker\nState: R (running)\nVmRSS: 0 kB\n"

	got, err := parseStatus(strings.NewReader(input), 42)
	if err != nil {
		t.Fatalf("zero RSS should be valid : %v", err)
	}

	if !got.RSSAvailable {
		t.Error("RSSAvailable = false expected true")
	}

	if got.RSSKB != 0 {
		t.Errorf("RSSKB = %d, want 0", got.RSSKB)
	}
}

func TestParseStatusMissingRSS(t *testing.T) {
	input := "PPid: 201\nName: worker\nState: R (running)\n"
	got, err := parseStatus(strings.NewReader(input), 42)
	if err != nil {
		t.Fatalf("mssing RSS should be allowed : %v", err)
	}

	if got.RSSAvailable {
		t.Error("RSSAvailable = true expected false")
	}

	if got.RSSKB != 0 {
		t.Errorf("RSSKB = %d expected 0", got.RSSKB)
	}
}

func TestReadProcessSkipDisappearedProcess(t *testing.T) {
	root := t.TempDir()
	// create a test directory for a process that will disappear
	pidDir := filepath.Join(root, "1234")
	// the dir will be : /tmp/TestReadProcessSkipDisappearedProcess123456789/1234 but we deliberately
	// don't create the status file to simulate a disappeared process
	if err := os.Mkdir(pidDir, 0755); err != nil {
		t.Fatalf("failed to create test process directory: %v", err)
	}

	processes, err := ReadProcessesFrom(root)
	if err != nil {
		t.Fatalf("readProcessesFrom() error = %v", err)
	}

	if len(processes) != 0 {
		t.Errorf("expected 0 processes, got %d", len(processes))
	}
}

func TestReadProcessMissingPID(t *testing.T) {
	_, err := ReadProcessAt(t.TempDir(), 999999) // assuming this PID does not exist
	if err == nil {
		t.Fatalf("expected error for missing PID")
	}

	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("error =%v, want an error wrapping os,ErrNotExist", err)
	}
}
