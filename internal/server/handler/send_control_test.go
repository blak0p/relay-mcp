package handler

import (
	"context"
	"errors"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	serror "github.com/blak0p/relay-mcp/internal/session/error"
	"github.com/blak0p/relay-mcp/internal/session/registry"
	"github.com/blak0p/relay-mcp/internal/session/session"
)

func newSendControlRequest(key any) mcp.CallToolRequest {
	args := map[string]any{}
	if key != nil {
		args["key"] = key
	}
	return mcp.CallToolRequest{
		Params: mcp.CallToolParams{Name: "send_control", Arguments: args},
	}
}

func TestSendControl_RejectsInvalidKeyBeforeSessionLookup(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name string
		key  any
	}{
		{name: "missing", key: nil},
		{name: "wrong type", key: 3},
		{name: "unsupported", key: "alt+x"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			res, err := NewSendControlHandler(registry.NewRegistry())(context.Background(), newSendControlRequest(tt.key))
			if err != nil {
				t.Fatalf("handler returned Go error: %v", err)
			}
			if got := extractError(t, res).Code; got != codeInvalidArgument {
				t.Fatalf("error code = %d, want %d", got, codeInvalidArgument)
			}
		})
	}
}

func TestSendControl_ActiveSessionWritesCanonicalBytesAndReturnsExactResult(t *testing.T) {
	t.Parallel()

	reg := registry.NewRegistry()
	wantSession := seedLiveSession(t, reg)
	var gotSession *session.Session
	var gotBytes []byte

	res, err := handleSendControlWithWriter(context.Background(), reg, newSendControlRequest(" UP "), func(s *session.Session, data []byte) (int, error) {
		gotSession = s
		gotBytes = append([]byte(nil), data...)
		return len(data), nil
	})
	if err != nil {
		t.Fatalf("handler returned Go error: %v", err)
	}
	if gotSession != wantSession {
		t.Fatalf("writer session = %p, want active session %p", gotSession, wantSession)
	}
	if want := []byte{0x1b, 0x5b, 0x41}; string(gotBytes) != string(want) {
		t.Fatalf("writer bytes = %x, want %x", gotBytes, want)
	}
	if res.IsError || len(res.Content) != 1 {
		t.Fatalf("result = %#v, want one successful text result", res)
	}
	text, ok := res.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("result content = %T, want mcp.TextContent", res.Content[0])
	}
	if text.Text != `{"key":"up","bytes_sent":3}` {
		t.Fatalf("result text = %s, want exactly {\"key\":\"up\",\"bytes_sent\":3}", text.Text)
	}
}

func TestSendControl_MapsWriteFailures(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name string
		err  error
		want int
	}{
		{name: "typed closed", err: serror.ErrSessionClosed, want: codeSessionClosed},
		{name: "generic", err: errors.New("writer failed"), want: codeSpawnFailed},
	} {
		t.Run(tt.name, func(t *testing.T) {
			reg := registry.NewRegistry()
			seedLiveSession(t, reg)
			res, err := handleSendControlWithWriter(context.Background(), reg, newSendControlRequest("enter"), func(_ *session.Session, _ []byte) (int, error) {
				return 0, tt.err
			})
			if err != nil {
				t.Fatalf("handler returned Go error: %v", err)
			}
			if got := extractError(t, res).Code; got != tt.want {
				t.Fatalf("error code = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestSendControl_ShortWriteFailsWithoutRetry(t *testing.T) {
	t.Parallel()

	reg := registry.NewRegistry()
	seedLiveSession(t, reg)
	calls := 0
	res, err := handleSendControlWithWriter(context.Background(), reg, newSendControlRequest("up"), func(_ *session.Session, data []byte) (int, error) {
		calls++
		return len(data) - 1, nil
	})
	if err != nil {
		t.Fatalf("handler returned Go error: %v", err)
	}
	if got := extractError(t, res).Code; got != codeSpawnFailed {
		t.Fatalf("error code = %d, want %d", got, codeSpawnFailed)
	}
	if calls != 1 {
		t.Fatalf("writer calls = %d, want exactly 1 (no retry)", calls)
	}
}
