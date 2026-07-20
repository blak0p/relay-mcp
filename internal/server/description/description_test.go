package description

import "testing"

func TestCreateTerminalConstants_NonEmpty(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		value string
	}{
		{"CreateTerminalName", CreateTerminalName},
		{"CreateTerminalSummary", CreateTerminalSummary},
		{"CreateTerminalDescription", CreateTerminalDescription},
	}
	for _, c := range cases {
		if c.value == "" {
			t.Fatalf("%s is empty, want a non-empty string", c.name)
		}
	}
}

func TestCreateTerminalName_IsCreateTerminal(t *testing.T) {
	t.Parallel()
	if CreateTerminalName != "create_terminal" {
		t.Fatalf("CreateTerminalName = %q, want %q", CreateTerminalName, "create_terminal")
	}
}
