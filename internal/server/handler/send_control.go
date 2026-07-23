package handler

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/blak0p/relay-mcp/internal/control"
	serror "github.com/blak0p/relay-mcp/internal/session/error"
	"github.com/blak0p/relay-mcp/internal/session/registry"
	"github.com/blak0p/relay-mcp/internal/session/session"
)

// SendControlResult is the success payload returned by send_control.
type SendControlResult struct {
	Key       string `json:"key"`
	BytesSent int    `json:"bytes_sent"`
}

type controlWriter func(*session.Session, []byte) (int, error)

// NewSendControlHandler returns the send_control handler for the shared
// single-session registry.
func NewSendControlHandler(reg *registry.Registry) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if reg == nil {
		panic("handler: NewSendControlHandler requires a non-nil registry")
	}
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleSendControlWithWriter(ctx, reg, req, func(s *session.Session, data []byte) (int, error) {
			return s.Write(data)
		})
	}
}

func handleSendControlWithWriter(_ context.Context, reg *registry.Registry, req mcp.CallToolRequest, write controlWriter) (*mcp.CallToolResult, error) {
	key, err := req.RequireString("key")
	if err != nil {
		return errorResult(codeInvalidArgument, fmt.Sprintf("missing or invalid 'key' argument: %v", err), nil), nil
	}

	sequence, err := control.Resolve(key)
	if err != nil {
		return errorResult(codeInvalidArgument, fmt.Sprintf("unsupported 'key' argument: %q", key), nil), nil
	}

	s, err := reg.Get()
	if err != nil {
		if errors.Is(err, serror.ErrSessionNotFound) {
			return errorResult(codeSessionNotFound, "no active session; call create_terminal first", nil), nil
		}
		return errorResult(codeSpawnFailed, fmt.Sprintf("registry unavailable: %v", err), nil), nil
	}

	n, err := write(s, sequence.Bytes)
	if err == nil && n != len(sequence.Bytes) {
		err = fmt.Errorf("send control %q: %w", sequence.Key, io.ErrShortWrite)
	}
	if err != nil {
		return mapWriteError(err, s.ID), nil
	}

	return successResult(SendControlResult{Key: sequence.Key, BytesSent: n}), nil
}
