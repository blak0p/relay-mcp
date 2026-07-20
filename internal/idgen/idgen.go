package idgen

import (
	"crypto/rand"
	"encoding/hex"
)

// New returns a new unique session id of the form "term_" + 16 hex chars.
// Panics if the system's cryptographic random source is unavailable, which
// is preferable to silently returning a duplicate or weak id.
func New() string {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		panic("idgen: crypto/rand unavailable: " + err.Error())
	}
	return "term_" + hex.EncodeToString(buf[:])
}
