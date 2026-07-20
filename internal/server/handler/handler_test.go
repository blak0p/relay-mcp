package handler

import (
	"context"
	"errors"
	"os/exec"
	"regexp"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/blak0p/relay-mcp/internal/session/error"
	"github.com/blak0p/relay-mcp/internal/session/registry"
	"github.com/blak0p/relay-mcp/internal/session/session"
)

// sessionIDFormat matches the id format produced by idgen.New (term_ + 16 hex).
var sessionIDFormat = regexp.MustCompile(`^term_[0-9a-f]{16}$`)

// errorPayload is the JSON shape the handler writes into CallToolResult.Content
// for tool-level errors. It preserves the JSON-RPC-style code/message/data
// triple that the design mandates, even though mcp-go surfaces tool failures
// as CallToolResult{IsError:true} rather than as JSON-RPC error responses.
type errorPayload struct {
	Code    int            `json:"code"`
	Message string         `json:"message"`
	Data    map[string]any `json:"data,omitempty"`
}

// resultPayload is the JSON shape of a successful create_terminal result.
type resultPayload struct {
	ID        string `json:"id"`
	State     string `json:"state"`
	StartedAt string `json:"started_at"`
}

// extractResult parses the JSON text content of a successful CallToolResult.
func extractResult(t *testing.T, res *mcp.CallToolResult) resultPayload {
	t.Helper()
	if res == nil {
		t.Fatal("result is nil")
	}
	if res.IsError {
		t.Fatalf("result.IsError = true, want false; content=%v", res.Content)
	}
	if len(res.Content) == 0 {
		t.Fatal("result.Content is empty")
	}
	tc, ok := res.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("content[0] = %T, want mcp.TextContent", res.Content[0])
	}
	out, err := parseJSON[resultPayload](tc.Text)
	if err != nil {
		t.Fatalf("parse result: %v (raw=%q)", err, tc.Text)
	}
	return out
}

// extractError parses the JSON text content of an errored CallToolResult and
// returns the code, message, and data map.
func extractError(t *testing.T, res *mcp.CallToolResult) errorPayload {
	t.Helper()
	if res == nil {
		t.Fatal("result is nil")
	}
	if !res.IsError {
		t.Fatalf("result.IsError = false, want true; content=%v", res.Content)
	}
	if len(res.Content) == 0 {
		t.Fatal("result.Content is empty")
	}
	tc, ok := res.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("content[0] = %T, want mcp.TextContent", res.Content[0])
	}
	out, err := parseJSON[errorPayload](tc.Text)
	if err != nil {
		t.Fatalf("parse error: %v (raw=%q)", err, tc.Text)
	}
	return out
}

func TestCreateTerminalHandler_HappyPath(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not in PATH; skipping spawn test")
	}
	reg := registry.NewRegistry()
	h := New(reg)

	res, err := h(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("handler returned Go error: %v; want nil (tool errors go in IsError)", err)
	}
	out := extractResult(t, res)
	if !sessionIDFormat.MatchString(out.ID) {
		t.Fatalf("result.ID = %q, want match %s", out.ID, sessionIDFormat.String())
	}
	if out.State != string(session.StateRunning) {
		t.Fatalf("result.State = %q, want %q", out.State, session.StateRunning)
	}
	if out.StartedAt == "" {
		t.Fatal("result.StartedAt is empty")
	}

	// The session must be registered and retrievable.
	got, gerr := reg.Get()
	if gerr != nil {
		t.Fatalf("reg.Get after handler = %v, want nil", gerr)
	}
	if got.ID != out.ID {
		t.Fatalf("registered session ID = %q, want %q", got.ID, out.ID)
	}
	// Best-effort cleanup so later subtests don't hold the PTY open.
	_ = got.Close()
}

func TestCreateTerminalHandler_SessionAlreadyExists(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not in PATH; skipping spawn test")
	}
	reg := registry.NewRegistry()
	h := New(reg)

	// First call seeds the registry.
	first, err := h(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	firstOut := extractResult(t, first)
	t.Cleanup(func() {
		if s, gerr := reg.Get(); gerr == nil {
			_ = s.Close()
		}
	})

	// Second call must fail with -32001 and the existing id.
	second, err := h(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("second call returned Go error: %v; want nil (tool errors go in IsError)", err)
	}
	e := extractError(t, second)
	if e.Code != -32001 {
		t.Fatalf("error code = %d, want -32001", e.Code)
	}
	if !strings.Contains(e.Message, "session") || !strings.Contains(e.Message, "active") {
		t.Fatalf("error message = %q, want it to mention an active session", e.Message)
	}
	if e.Data == nil {
		t.Fatal("error.Data is nil, want existing session id")
	}
	existingID, _ := e.Data["existing_id"].(string)
	if existingID != firstOut.ID {
		t.Fatalf("error.Data.existing_id = %q, want %q", existingID, firstOut.ID)
	}

	// Sanity: the registry must still hold the original session.
	got, gerr := reg.Get()
	if gerr != nil {
		t.Fatalf("reg.Get after duplicate = %v, want nil", gerr)
	}
	if got.ID != firstOut.ID {
		t.Fatalf("registry was mutated by rejected call; got %q, want %q", got.ID, firstOut.ID)
	}
}

func TestCreateTerminalHandler_BashNotFound(t *testing.T) {
	// Not parallel: t.Setenv mutates the process environment.
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not in PATH; skipping bash_not_found test")
	}
	// Force PATH to a directory with no bash. /nonexistent is guaranteed absent.
	t.Setenv("PATH", "/nonexistent")

	reg := registry.NewRegistry()
	h := New(reg)

	res, err := h(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("handler returned Go error: %v; want nil (tool errors go in IsError)", err)
	}
	e := extractError(t, res)
	if e.Code != -32002 {
		t.Fatalf("error code = %d, want -32002 (bash_not_found)", e.Code)
	}
	if !strings.Contains(e.Message, "bash") {
		t.Fatalf("error message = %q, want it to mention bash", e.Message)
	}

	// No session must have been registered.
	if _, gerr := reg.Get(); !errors.Is(gerr, serror.ErrSessionNotFound) {
		t.Fatalf("reg.Get after bash_not_found = %v, want ErrSessionNotFound", gerr)
	}
}

func TestCreateTerminalHandler_SpawnFailed(t *testing.T) {
	t.Parallel()
	// Use a real, discoverable shell but force the spawn to fail by pointing
	// SHELL at a non-executable file. The handler uses exec.LookPath("bash");
	// when bash is present we instead override the spawn via the WithSpawner
	// option to force an error from pty.StartWithSize. This isolates the
	// spawn_failed path from the bash_not_found path.
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not in PATH; skipping spawn_failed test")
	}
	reg := registry.NewRegistry()
	h := New(reg, WithSpawner(funcaltySpawner))

	res, err := h(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("handler returned Go error: %v; want nil (tool errors go in IsError)", err)
	}
	e := extractError(t, res)
	if e.Code != -32003 {
		t.Fatalf("error code = %d, want -32003 (spawn_failed)", e.Code)
	}
	if !strings.Contains(e.Message, "spawn") {
		t.Fatalf("error message = %q, want it to mention spawn", e.Message)
	}

	// No session must have been registered.
	if _, gerr := reg.Get(); !errors.Is(gerr, serror.ErrSessionNotFound) {
		t.Fatalf("reg.Get after spawn_failed = %v, want ErrSessionNotFound", gerr)
	}
}
