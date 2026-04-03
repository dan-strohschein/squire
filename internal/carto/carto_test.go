package carto

import (
	"testing"

	"github.com/dan-strohschein/squire/internal/testutil"
)

func TestRun_Stats(t *testing.T) {
	aidDir := testutil.AidDir(t)

	err := Run(aidDir, []string{"stats"})
	if err != nil {
		t.Fatalf("Run(stats) error: %v", err)
	}
}

func TestRun_Callstack(t *testing.T) {
	aidDir := testutil.AidDir(t)

	err := Run(aidDir, []string{"callstack", "Server.Start", "--down"})
	if err != nil {
		t.Fatalf("Run(callstack) error: %v", err)
	}
}

func TestRun_Depends(t *testing.T) {
	aidDir := testutil.AidDir(t)

	err := Run(aidDir, []string{"depends", "Config"})
	if err != nil {
		t.Fatalf("Run(depends) error: %v", err)
	}
}

func TestRun_Search(t *testing.T) {
	aidDir := testutil.AidDir(t)

	err := Run(aidDir, []string{"search", "Server*"})
	if err != nil {
		t.Fatalf("Run(search) error: %v", err)
	}
}

func TestRun_List(t *testing.T) {
	aidDir := testutil.AidDir(t)

	err := Run(aidDir, []string{"list", "testpkg"})
	if err != nil {
		t.Fatalf("Run(list) error: %v", err)
	}
}

func TestRun_Effects(t *testing.T) {
	aidDir := testutil.AidDir(t)

	err := Run(aidDir, []string{"effects", "Server.Start"})
	if err != nil {
		t.Fatalf("Run(effects) error: %v", err)
	}
}

func TestRun_NoCommand(t *testing.T) {
	aidDir := testutil.AidDir(t)

	err := Run(aidDir, []string{})
	if err == nil {
		t.Fatal("expected error for empty command")
	}
}

func TestRun_UnknownCommand(t *testing.T) {
	aidDir := testutil.AidDir(t)

	err := Run(aidDir, []string{"bogus"})
	if err == nil {
		t.Fatal("expected error for unknown command")
	}
}

func TestRun_InvalidDir(t *testing.T) {
	err := Run("/nonexistent", []string{"stats"})
	if err == nil {
		t.Fatal("expected error for nonexistent directory")
	}
}
