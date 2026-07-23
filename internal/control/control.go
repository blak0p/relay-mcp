package control

import (
	"errors"
	"strings"
)

// ErrUnsupportedKey is returned when input is not one of the supported control
// keys.
var ErrUnsupportedKey = errors.New("unsupported control key")

// Sequence is a canonical control key and the bytes sent to the terminal.
type Sequence struct {
	Key   string
	Bytes []byte
}

type entry struct {
	key   string
	bytes string
}

var entries = []entry{
	{"ctrl+c", "\x03"},
	{"ctrl+d", "\x04"},
	{"ctrl+z", "\x1a"},
	{"ctrl+\\", "\x1c"},
	{"ctrl+l", "\x0c"},
	{"ctrl+a", "\x01"},
	{"ctrl+e", "\x05"},
	{"ctrl+k", "\x0b"},
	{"ctrl+u", "\x15"},
	{"ctrl+w", "\x17"},
	{"ctrl+r", "\x12"},
	{"tab", "\x09"},
	{"enter", "\x0d"},
	{"escape", "\x1b"},
	{"backspace", "\x7f"},
	{"up", "\x1b[A"},
	{"down", "\x1b[B"},
	{"right", "\x1b[C"},
	{"left", "\x1b[D"},
	{"home", "\x1b[H"},
	{"end", "\x1b[F"},
	{"delete", "\x1b[3~"},
	{"page_up", "\x1b[5~"},
	{"page_down", "\x1b[6~"},
}

// Resolve normalizes input and returns the corresponding finite control
// sequence. The returned byte slice does not share backing storage with the
// allowlist.
func Resolve(input string) (Sequence, error) {
	key := strings.ToLower(strings.TrimSpace(input))
	for _, entry := range entries {
		if entry.key == key {
			return Sequence{Key: entry.key, Bytes: []byte(entry.bytes)}, nil
		}
	}
	return Sequence{}, ErrUnsupportedKey
}

// SupportedKeys returns the canonical control keys in deterministic order.
func SupportedKeys() []string {
	keys := make([]string, len(entries))
	for i, entry := range entries {
		keys[i] = entry.key
	}
	return keys
}
