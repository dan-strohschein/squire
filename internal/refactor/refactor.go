// Package refactor wraps chisel's refactoring engine for squire.
package refactor

import (
	"github.com/dan-strohschein/chisel/resolve"
)

// NewGraphQuerier creates a chisel LibraryGraphQuerier that uses the
// embedded cartograph engine directly. This replaces the old
// EmbeddedGraphQuerier which used a JSON serialization bridge.
func NewGraphQuerier(aidDir string) (*resolve.LibraryGraphQuerier, error) {
	return resolve.NewLibraryGraphQuerier(aidDir)
}
