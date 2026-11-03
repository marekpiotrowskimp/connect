// Package idgen generates random identifiers used as primary keys for
// peers, messages, and files.
package idgen

import (
	"crypto/rand"
	"fmt"
)

// New returns a random UUID v4 string.
func New() string {
	var b [16]byte
	// crypto/rand.Read on the fixed-size array only fails if the OS RNG is
	// unavailable, which would make the whole process unusable anyway; a
	// panic here is preferable to threading an error through every caller.
	if _, err := rand.Read(b[:]); err != nil {
		panic(fmt.Sprintf("idgen: read random bytes: %v", err))
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
