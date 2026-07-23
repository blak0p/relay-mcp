package control

import (
	"bytes"
	"testing"
)

func TestResolve_AllowedKeys(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		key  string
		want []byte
	}{
		{"ctrl+c", []byte{0x03}},
		{"ctrl+d", []byte{0x04}},
		{"ctrl+z", []byte{0x1a}},
		{"ctrl+\\", []byte{0x1c}},
		{"ctrl+l", []byte{0x0c}},
		{"ctrl+a", []byte{0x01}},
		{"ctrl+e", []byte{0x05}},
		{"ctrl+k", []byte{0x0b}},
		{"ctrl+u", []byte{0x15}},
		{"ctrl+w", []byte{0x17}},
		{"ctrl+r", []byte{0x12}},
		{"tab", []byte{0x09}},
		{"enter", []byte{0x0d}},
		{"escape", []byte{0x1b}},
		{"backspace", []byte{0x7f}},
		{"up", []byte{0x1b, 0x5b, 0x41}},
		{"down", []byte{0x1b, 0x5b, 0x42}},
		{"right", []byte{0x1b, 0x5b, 0x43}},
		{"left", []byte{0x1b, 0x5b, 0x44}},
		{"home", []byte{0x1b, 0x5b, 0x48}},
		{"end", []byte{0x1b, 0x5b, 0x46}},
		{"delete", []byte{0x1b, 0x5b, 0x33, 0x7e}},
		{"page_up", []byte{0x1b, 0x5b, 0x35, 0x7e}},
		{"page_down", []byte{0x1b, 0x5b, 0x36, 0x7e}},
	} {
		t.Run(tt.key, func(t *testing.T) {
			got, err := Resolve(tt.key)
			if err != nil {
				t.Fatalf("Resolve(%q) error = %v", tt.key, err)
			}
			if got.Key != tt.key {
				t.Fatalf("Resolve(%q).Key = %q, want %q", tt.key, got.Key, tt.key)
			}
			if !bytes.Equal(got.Bytes, tt.want) {
				t.Fatalf("Resolve(%q).Bytes = %x, want %x", tt.key, got.Bytes, tt.want)
			}
		})
	}
}

func TestResolve_NormalizesAndRejectsUnsupportedInput(t *testing.T) {
	t.Parallel()

	got, err := Resolve("  CTRL+C  ")
	if err != nil {
		t.Fatalf("Resolve normalized key error = %v", err)
	}
	if got.Key != "ctrl+c" || !bytes.Equal(got.Bytes, []byte{0x03}) {
		t.Fatalf("Resolve normalized key = %#v, want ctrl+c and 03", got)
	}

	for _, input := range []string{"", "f1", "alt+x", "\\x03", "ctrl+c+ctrl+d"} {
		t.Run(input, func(t *testing.T) {
			if _, err := Resolve(input); err == nil {
				t.Fatalf("Resolve(%q) error = nil, want unsupported-key error", input)
			}
		})
	}
}

func TestSupportedKeys_AreOrderedAndCopySafe(t *testing.T) {
	t.Parallel()

	want := []string{
		"ctrl+c", "ctrl+d", "ctrl+z", "ctrl+\\", "ctrl+l", "ctrl+a", "ctrl+e", "ctrl+k", "ctrl+u", "ctrl+w", "ctrl+r",
		"tab", "enter", "escape", "backspace", "up", "down", "right", "left", "home", "end", "delete", "page_up", "page_down",
	}
	keys := SupportedKeys()
	if !equalStrings(keys, want) {
		t.Fatalf("SupportedKeys() = %#v, want %#v", keys, want)
	}
	keys[0] = "changed"
	if got := SupportedKeys()[0]; got != "ctrl+c" {
		t.Fatalf("SupportedKeys mutation leaked: first key = %q, want ctrl+c", got)
	}

	sequence, err := Resolve("up")
	if err != nil {
		t.Fatalf("Resolve(up) error = %v", err)
	}
	sequence.Bytes[0] = 0
	again, err := Resolve("up")
	if err != nil {
		t.Fatalf("Resolve(up) after mutation error = %v", err)
	}
	if !bytes.Equal(again.Bytes, []byte{0x1b, 0x5b, 0x41}) {
		t.Fatalf("Resolve(up) after mutation = %x, want 1b5b41", again.Bytes)
	}
}

func equalStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
