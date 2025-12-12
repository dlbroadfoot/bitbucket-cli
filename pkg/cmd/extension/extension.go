// Package extension provides types for managing CLI extensions.
// This is a minimal package for Bitbucket CLI compatibility.
package extension

// ExtensionKind represents the type of extension
type ExtensionKind int

const (
	// LocalKind represents a locally developed extension
	LocalKind ExtensionKind = iota
	// GitKind represents a git-based extension
	GitKind
	// BinaryKind represents a pre-compiled binary extension
	BinaryKind
)
