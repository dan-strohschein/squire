package refactor

import (
	"testing"

	"github.com/dan-strohschein/squire/internal/testutil"
)

func TestNewGraphQuerier_ValidDir(t *testing.T) {
	aidDir := testutil.AidDir(t)

	querier, err := NewGraphQuerier(aidDir)
	if err != nil {
		t.Fatalf("NewGraphQuerier(%s) returned error: %v", aidDir, err)
	}
	if querier == nil {
		t.Fatal("NewGraphQuerier returned nil querier")
	}
	if querier.Engine == nil {
		t.Error("querier.Engine is nil")
	}
	if querier.Graph == nil {
		t.Error("querier.Graph is nil")
	}
}

func TestNewGraphQuerier_InvalidDir(t *testing.T) {
	_, err := NewGraphQuerier("/nonexistent/path")
	if err == nil {
		t.Fatal("expected error for nonexistent directory, got nil")
	}
}

func TestNewGraphQuerier_Query(t *testing.T) {
	aidDir := testutil.AidDir(t)

	querier, err := NewGraphQuerier(aidDir)
	if err != nil {
		t.Fatalf("NewGraphQuerier: %v", err)
	}

	// Test a callstack query — Server.Start calls Config.Validate
	result, err := querier.Query("callstack", []string{"Server.Start", "--down"}, aidDir)
	if err != nil {
		t.Fatalf("Query(callstack) error: %v", err)
	}
	if result == nil {
		t.Fatal("Query(callstack) returned nil result")
	}
	if result.NodeCount == 0 {
		t.Error("expected non-zero NodeCount for Server.Start callstack")
	}
}

func TestNewGraphQuerier_ErrorProducers(t *testing.T) {
	aidDir := testutil.AidDir(t)

	querier, err := NewGraphQuerier(aidDir)
	if err != nil {
		t.Fatalf("NewGraphQuerier: %v", err)
	}

	// ErrorProducers should work without panicking even if no matches
	result, err := querier.ErrorProducers("error")
	if err != nil {
		// Not all fixtures produce errors — a clean error is fine
		t.Logf("ErrorProducers returned error (expected for simple fixture): %v", err)
		return
	}
	if result == nil {
		t.Error("ErrorProducers returned nil result without error")
	}
}
