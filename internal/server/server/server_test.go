package server

import (
	"testing"

	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/blak0p/relay-mcp/internal/server/description"
	"github.com/blak0p/relay-mcp/internal/session/registry"
)

func TestNewServer_RegistersCreateTerminalTool(t *testing.T) {
	t.Parallel()
	reg := registry.NewRegistry()
	s, err := NewServer(reg)
	if err != nil {
		t.Fatalf("NewServer = %v, want nil", err)
	}
	if s == nil {
		t.Fatal("NewServer returned nil server, want non-nil")
	}

	tools := s.ListTools()
	if len(tools) == 0 {
		t.Fatal("server has no tools registered, want create_terminal")
	}
	tool, ok := tools[description.CreateTerminalName]
	if !ok {
		t.Fatalf("create_terminal not registered; registered tools = %v", toolNames(tools))
	}
	if tool.Tool.Name != description.CreateTerminalName {
		t.Fatalf("registered tool name = %q, want %q", tool.Tool.Name, description.CreateTerminalName)
	}
	if tool.Tool.Description != description.CreateTerminalDescription {
		t.Fatalf("registered tool description = %q, want %q", tool.Tool.Description, description.CreateTerminalDescription)
	}
	if tool.Handler == nil {
		t.Fatal("registered tool handler is nil")
	}
}

func TestNewServer_NilRegistryReturnsError(t *testing.T) {
	t.Parallel()
	_, err := NewServer(nil)
	if err == nil {
		t.Fatal("NewServer(nil) returned nil error, want non-nil")
	}
}

func toolNames(m map[string]*mcpserver.ServerTool) []string {
	out := make([]string, 0, len(m))
	for n := range m {
		out = append(out, n)
	}
	return out
}
