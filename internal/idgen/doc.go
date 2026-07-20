// Package idgen generates unique, unpredictable session identifiers for
// terminal sessions.
//
// IDs use the "term_" prefix followed by 16 lowercase hex chars (8 bytes from
// crypto/rand). Zero dependencies, stdlib only. This is a flat package (no
// sub-packages) because it owns a single, small responsibility.
package idgen
