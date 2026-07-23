package handler

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	serror "github.com/blak0p/relay-mcp/internal/session/error"
	"github.com/blak0p/relay-mcp/internal/session/registry"
	"github.com/blak0p/relay-mcp/internal/session/session"
)

const (
	codeSessionCleanup = -32008
	closeGracePeriod   = 5 * time.Second
)

// CloseTerminalResult is the success payload returned by close_terminal.
// Status and ExitCode are omitted for idempotent no-op responses.
type CloseTerminalResult struct {
	Closed   bool   `json:"closed"`
	Status   string `json:"status,omitempty"`
	ExitCode *int   `json:"exit_code,omitempty"`
}

// NewCloseTerminalHandler returns the close_terminal handler for the shared
// single-session registry.
func NewCloseTerminalHandler(reg *registry.Registry) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if reg == nil {
		panic("handler: NewCloseTerminalHandler requires a non-nil registry")
	}
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleCloseTerminal(ctx, reg, req)
	}
}

func handleCloseTerminal(_ context.Context, reg *registry.Registry, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionID, err := req.RequireString("session_id")
	if err != nil {
		return errorResult(codeInvalidArgument, fmt.Sprintf("missing or invalid 'session_id' argument: %v", err), nil), nil
	}

	result, matched, err := reg.Release(sessionID, func(s *session.Session) (session.CloseResult, error) {
		return s.Shutdown(closeGracePeriod)
	})
	if !matched {
		return successResult(CloseTerminalResult{Closed: false}), nil
	}
	if err != nil {
		return mapCloseTerminalError(err, sessionID), nil
	}
	return successResult(closeTerminalSuccess(result)), nil
}

func closeTerminalSuccess(result session.CloseResult) CloseTerminalResult {
	exitCode := result.ExitCode
	return CloseTerminalResult{
		Closed:   true,
		Status:   string(result.State),
		ExitCode: &exitCode,
	}
}

func mapCloseTerminalError(err error, sessionID string) *mcp.CallToolResult {
	reason := "cleanup_failed"
	if !errors.Is(err, serror.ErrSessionCleanup) {
		reason = "cleanup_failed"
	}
	return errorResult(codeSessionCleanup,
		fmt.Sprintf("failed to close session %s: %v", sessionID, err),
		map[string]any{"session_id": sessionID, "reason": reason},
	)
}
