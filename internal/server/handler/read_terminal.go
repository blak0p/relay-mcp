package handler

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	serror "github.com/blak0p/relay-mcp/internal/session/error"
	"github.com/blak0p/relay-mcp/internal/session/output"
	"github.com/blak0p/relay-mcp/internal/session/registry"
)

const (
	readModeStream   = "stream"
	readModeSnapshot = "snapshot"
	readModeDrain    = "drain"
	streamThrottle   = 50 * time.Millisecond
)

// ReadTerminalResult is the payload returned by snapshot, drain, and completed
// stream calls. Cursors are absolute positions in the retained output stream.
type ReadTerminalResult struct {
	Output       string `json:"output"`
	Cursor       int64  `json:"cursor"`
	NextCursor   int64  `json:"next_cursor"`
	DroppedBytes int64  `json:"dropped_bytes"`
	Status       string `json:"status"`
}

// ProgressSender delivers one MCP progress notification. It is injected in
// tests so slow or failed recipients never need a live MCP server.
type ProgressSender func(context.Context, string, map[string]any) error

// ReadTerminalOption configures a read_terminal handler.
type ReadTerminalOption func(*readTerminalConfig)

type readTerminalConfig struct {
	send ProgressSender
}

// WithProgressSender replaces notification delivery. It is primarily useful
// for focused handler tests.
func WithProgressSender(sender ProgressSender) ReadTerminalOption {
	return func(config *readTerminalConfig) { config.send = sender }
}

type readTerminalArgs struct {
	Mode     *string `json:"mode"`
	Cursor   *int64  `json:"cursor"`
	MaxBytes *int    `json:"max_bytes"`
	WaitMS   *int    `json:"wait_ms"`
}

// NewReadTerminalHandler returns the read_terminal handler for the shared
// single-session registry.
func NewReadTerminalHandler(reg *registry.Registry, options ...ReadTerminalOption) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if reg == nil {
		panic("handler: NewReadTerminalHandler requires a non-nil registry")
	}
	config := readTerminalConfig{send: sendProgress}
	for _, option := range options {
		option(&config)
	}
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleReadTerminal(ctx, reg, config.send, req)
	}
}

func handleReadTerminal(ctx context.Context, reg *registry.Registry, send ProgressSender, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, result := parseReadArgs(req)
	if result != nil {
		return result, nil
	}
	if args.mode == readModeStream && (req.Params.Meta == nil || req.Params.Meta.ProgressToken == nil) {
		return readError(codeInvalidArgument, "stream mode requires _meta.progressToken", "stream_requires_progress_token"), nil
	}

	s, err := reg.Get()
	if err != nil {
		if errors.Is(err, serror.ErrSessionNotFound) {
			return readError(codeSessionNotFound, "no active session; call create_terminal first", "session_not_found"), nil
		}
		return errorResult(codeSpawnFailed, fmt.Sprintf("registry unavailable: %v", err), nil), nil
	}
	state, stateErr := s.Output.Snapshot(ctx, args.cursor, 1, 0)
	if stateErr != nil {
		return nil, stateErr
	}
	if state.Status == output.StatusClosed {
		return readError(codeSessionClosed, "session is closed", "session_closed"), nil
	}

	if args.mode == readModeStream {
		return streamTerminal(ctx, s.Output, args.cursor, req.Params.Meta.ProgressToken, send), nil
	}
	snapshot, err := s.Output.Snapshot(ctx, args.cursor, args.maxBytes, args.wait)
	if err != nil {
		return nil, err
	}
	return successResult(readResult(snapshot)), nil
}

type parsedReadArgs struct {
	mode     string
	cursor   int64
	maxBytes int
	wait     time.Duration
}

func parseReadArgs(req mcp.CallToolRequest) (parsedReadArgs, *mcp.CallToolResult) {
	var request readTerminalArgs
	if err := req.BindArguments(&request); err != nil {
		return parsedReadArgs{}, readError(codeInvalidArgument, "invalid read request", "invalid_read_request")
	}
	args := parsedReadArgs{mode: readModeStream, maxBytes: output.DefaultReadBytes, wait: output.DefaultWait}
	if request.Mode != nil {
		args.mode = *request.Mode
	}
	if request.Cursor != nil {
		args.cursor = *request.Cursor
	}
	if request.MaxBytes != nil {
		args.maxBytes = *request.MaxBytes
	}
	if request.WaitMS != nil {
		args.wait = time.Duration(*request.WaitMS) * time.Millisecond
	}
	if args.mode != readModeStream && args.mode != readModeSnapshot && args.mode != readModeDrain {
		return parsedReadArgs{}, readError(codeInvalidArgument, "invalid read mode", "invalid_read_request")
	}
	if args.cursor < 0 || args.maxBytes < 1 || args.maxBytes > output.MaxReadBytes || args.wait < 0 || args.wait > output.MaxWait {
		return parsedReadArgs{}, readError(codeInvalidArgument, "invalid read request", "invalid_read_request")
	}
	return args, nil
}

func streamTerminal(ctx context.Context, broker *output.Broker, cursor int64, token mcp.ProgressToken, send ProgressSender) *mcp.CallToolResult {
	var lastSent time.Time
	for {
		snapshot, err := broker.Snapshot(ctx, cursor, output.MaxReadBytes, output.MaxWait)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return successResult(ReadTerminalResult{Cursor: cursor, NextCursor: cursor, Status: string(output.StatusRunning)})
			}
			return errorResult(codeSpawnFailed, fmt.Sprintf("stream read failed: %v", err), nil)
		}
		if len(snapshot.Output) > 0 {
			if delay := streamThrottle - time.Since(lastSent); !lastSent.IsZero() && delay > 0 {
				if !waitForStream(ctx, delay) {
					return successResult(readResult(snapshot))
				}
			}
			if ctx.Err() != nil {
				return successResult(readResult(snapshot))
			}
			if err := send(ctx, "notifications/progress", progressParams(token, snapshot)); err != nil {
				return readError(codeSpawnFailed, "progress notification delivery failed", "stream_delivery_failed")
			}
			lastSent = time.Now()
			cursor = snapshot.NextCursor
		}
		if snapshot.Status != output.StatusRunning {
			return successResult(readResult(snapshot))
		}
	}
}

func progressParams(token mcp.ProgressToken, snapshot output.Snapshot) map[string]any {
	result := readResult(snapshot)
	return map[string]any{
		"progressToken": token,
		"progress":      result.NextCursor,
		"output":        result.Output,
		"cursor":        result.Cursor,
		"next_cursor":   result.NextCursor,
		"dropped_bytes": result.DroppedBytes,
		"status":        result.Status,
	}
}

func readResult(snapshot output.Snapshot) ReadTerminalResult {
	return ReadTerminalResult{Output: string(snapshot.Output), Cursor: snapshot.Cursor, NextCursor: snapshot.NextCursor, DroppedBytes: snapshot.DroppedBytes, Status: string(snapshot.Status)}
}

func readError(code int, message, reason string) *mcp.CallToolResult {
	return errorResult(code, message, map[string]any{"reason": reason})
}

func sendProgress(ctx context.Context, method string, params map[string]any) error {
	s := server.ServerFromContext(ctx)
	if s == nil {
		return server.ErrNotificationNotInitialized
	}
	return s.SendNotificationToClient(ctx, method, params)
}

func waitForStream(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
