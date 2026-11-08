package supervisor

import (
	"crypto/sha256"
	"encoding/hex"
)

const windowsPipePrefix = `\\.\pipe\dim-rust-`

// windowsPipeName derives the stable Windows named-pipe endpoint from the
// canonical native home: `\\.\pipe\dim-rust-<sha256("dimcode:"+home)[:32]>`
// (protocol-contract §18).
func windowsPipeName(nativeHome string) string {
	sum := sha256.Sum256([]byte("dimcode:" + nativeHome))
	return windowsPipePrefix + hex.EncodeToString(sum[:])[:32]
}
