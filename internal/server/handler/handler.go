package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/creack/pty"
	"github.com/mark3labs/mcp-go/mcp"

	"github.com/blak0p/relay-mcp/internal/server/description"
	"github.com/blak0p/relay-mcp/internal/session/error"
	"github.com/blak0p/relay-mcp/internal/session/registry"
	"github.com/blak0p/relay-mcp/internal/session/session"
)

// JSON-RPC-style error codes for create_terminal failures. These are the
// stable codes the design assigned to each failure mode; they travel inside
// the CallToolResult text payload (see package doc for why).
const (
	codeSessionAlreadyExists = -32001
	codeBashNotFound         = -32002
	codeSpawnFailed          = -32003
)

// PTY window size for spawned sessions. Fixed at 100x30 per the design; not
// configurable per call in v1.
const (
	ptyCols = 100
	ptyRows = 30
)

// CreateTerminalResult is the success payload returned by the create_terminal
// tool. It is JSON-marshalled into the CallToolResult text content.
type CreateTerminalResult struct {
	ID        string `json:"id"`
	State     string `json:"state"`
	StartedAt string `json:"started_at"`
}

// errorEnvelope is the failure payload returned by create_terminal. It mirrors
// the JSON-RPC error shape (code/message/data) so clients can branch on the
// stable code even though the failure is carried inside a CallToolResult.
type errorEnvelope struct {
	Code    int            `json:"code"`
	Message string         `json:"message"`
	Data    map[string]any `json:"data,omitempty"`
}

// Spawner starts a command in a PTY of the fixed 100x30 size and returns the
// PTY master and the started command. The default implementation calls
// pty.StartWithSize; tests can replace it via WithSpawner to exercise the
// spawn_failed path without depending on a missing binary.
type Spawner func(cmd *exec.Cmd) (*os.File, *exec.Cmd, error)

// Option configures a CreateTerminalHandler.
type Option func(*config)

type config struct {
	spawner Spawner
}

// WithSpawner overrides the default pty.StartWithSize-based spawner. Tests use
// it to force the spawn_failed error path deterministically.
func WithSpawner(s Spawner) Option {
	return func(c *config) { c.spawner = s }
}

// New returns a CreateTerminalHandler that spawns a bash PTY and registers the
// resulting session in reg. reg must be non-nil. The handler is safe for
// concurrent use as long as reg is.
func New(reg *registry.Registry, opts ...Option) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if reg == nil {
		panic("handler: New requires a non-nil registry")
	}
	cfg := config{spawner: defaultSpawner}
	for _, opt := range opts {
		opt(&cfg)
	}
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleCreateTerminal(ctx, reg, cfg.spawner)
	}
}

// handleCreateTerminal is the core of the create_terminal tool. It is split
// out of the returned closure so it can be unit-tested directly if needed;
// the public entry point is the closure returned by New.
func handleCreateTerminal(ctx context.Context, reg *registry.Registry, spawn Spawner) (*mcp.CallToolResult, error) {
	// 1. Reject up-front if a session is already active. We check the registry
	//    first so a duplicate call never spawns a second process.
	if existing, err := reg.Get(); err == nil {
		return errorResult(codeSessionAlreadyExists,
			fmt.Sprintf("a session is already active (id=%s); call close_terminal first", existing.ID),
			map[string]any{"existing_id": existing.ID},
		), nil
	} else if !errors.Is(err, serror.ErrSessionNotFound) {
		// Unexpected registry error — surface as a tool error with the
		// spawn_failed code (generic failure) so the client sees something.
		return errorResult(codeSpawnFailed, fmt.Sprintf("registry unavailable: %v", err), nil), nil
	}

	// 2. Resolve bash in PATH. No fallback shell.
	bashPath, err := exec.LookPath("bash")
	if err != nil {
		return errorResult(codeBashNotFound,
			"bash not found at /bin/bash. Set RELAY_MCP_SHELL or install bash.",
			nil,
		), nil
	}

	// 3. Spawn bash -i in a 100x30 PTY.
	cmd := exec.Command(bashPath, "-i")
	ptyFile, startedCmd, err := spawn(cmd)
	if err != nil {
		return errorResult(codeSpawnFailed, fmt.Sprintf("failed to spawn bash: %v", err), nil), nil
	}

	// 4. Build the session and register it. session.New populates the ID,
	//    state, and started-at timestamp; PID is read from the started cmd.
	s := session.New(startedCmd, ptyFile)
	if startedCmd.Process != nil {
		s.PID = startedCmd.Process.Pid
	}
	if err := reg.Put(s); err != nil {
		// Put can only fail with ErrSessionAlreadyExists (a race between the
		// Get check above and another goroutine's call). Close the PTY we just
		// opened so we don't leak the FD, then surface the existing id.
		_ = s.Close()
		existingID := serror.ExistingSessionID(err)
		return errorResult(codeSessionAlreadyExists,
			fmt.Sprintf("a session is already active (id=%s); call close_terminal first", existingID),
			map[string]any{"existing_id": existingID},
		), nil
	}

	// 5. Success.
	return successResult(CreateTerminalResult{
		ID:        s.ID,
		State:     string(session.StateRunning),
		StartedAt: s.StartedAt.UTC().Format(time.RFC3339Nano),
	}), nil
}

// defaultSpawner is the production Spawner: pty.StartWithSize with the fixed
// 100x30 window. It returns the PTY master and the started command (whose
// Process field is now populated by pty.Start).
func defaultSpawner(cmd *exec.Cmd) (*os.File, *exec.Cmd, error) {
	ws := &pty.Winsize{Rows: ptyRows, Cols: ptyCols}
	f, err := pty.StartWithSize(cmd, ws)
	if err != nil {
		return nil, nil, err
	}
	return f, cmd, nil
}

// successResult marshals out into a successful CallToolResult.
func successResult(out CreateTerminalResult) *mcp.CallToolResult {
	body, err := json.Marshal(out)
	if err != nil {
		// Should never happen for our struct; fall back to a generic error.
		return errorResult(codeSpawnFailed, fmt.Sprintf("marshal result: %v", err), nil)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: string(body)}},
	}
}

// errorResult builds an error CallToolResult carrying the code/message/data
// triple in its text content.
func errorResult(code int, msg string, data map[string]any) *mcp.CallToolResult {
	env := errorEnvelope{Code: code, Message: msg, Data: data}
	body, _ := json.Marshal(env)
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: string(body)}},
		IsError: true,
	}
}

// _ = description.CreateTerminalName // referenced to keep the import live;
// the handler does not re-declare the tool name, the server package uses it
// when registering the tool. Kept here as a compile-time link.
var _ = description.CreateTerminalName
