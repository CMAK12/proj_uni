package service

import (
	"crypto/rand"
	"encoding/hex"
)

// newID returns a random 128-bit identifier as a 32-char hex string. crypto/rand
// makes collisions effectively impossible without a database sequence.
func newID() string {
	var b [16]byte
	// rand.Read from crypto/rand never returns an error on supported platforms.
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
