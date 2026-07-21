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

// TestNewServer_RegistersWriteTerminalTool proves the write_terminal tool is
// registered alongside create_terminal with the correct name, description,
// and a non-nil handler. REQ-WT-008.
func TestNewServer_RegistersWriteTerminalTool(t *testing.T) {
	t.Parallel()
	reg := registry.NewRegistry()
	s, err := NewServer(reg)
	if err != nil {
		t.Fatalf("NewServer = %v, want nil", err)
	}

	tools := s.ListTools()

	// create_terminal must still be present.
	if _, ok := tools[description.CreateTerminalName]; !ok {
		t.Fatalf("create_terminal not registered; registered tools = %v", toolNames(tools))
	}

	// write_terminal must be present with the right name, description, and handler.
	wt, ok := tools[description.WriteTerminalName]
	if !ok {
		t.Fatalf("write_terminal not registered; registered tools = %v", toolNames(tools))
	}
	if wt.Tool.Name != description.WriteTerminalName {
		t.Fatalf("write_terminal tool name = %q, want %q", wt.Tool.Name, description.WriteTerminalName)
	}
	if wt.Tool.Description != description.WriteTerminalDescription {
		t.Fatalf("write_terminal description = %q, want %q", wt.Tool.Description, description.WriteTerminalDescription)
	}
	if wt.Handler == nil {
		t.Fatal("write_terminal handler is nil")
	}

	// The input schema must declare exactly one required string property "data".
	props := wt.Tool.InputSchema.Properties
	if len(props) != 1 {
		t.Fatalf("write_terminal input schema has %d properties, want 1 (data)", len(props))
	}
	rawProp, ok := props["data"]
	if !ok {
		t.Fatalf("write_terminal input schema has no 'data' property; got %v", props)
	}
	propMap, ok := rawProp.(map[string]any)
	if !ok {
		t.Fatalf("write_terminal 'data' property is %T, want map[string]any", rawProp)
	}
	propType, _ := propMap["type"].(string)
	if propType != "string" {
		t.Fatalf("write_terminal 'data' property type = %q, want \"string\"", propType)
	}
	if !containsString(wt.Tool.InputSchema.Required, "data") {
		t.Fatalf("write_terminal 'data' is not required; required = %v", wt.Tool.InputSchema.Required)
	}
}

// containsString reports whether s is in the slice. Kept local to avoid pulling
// in slices for one-off test use.
func containsString(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

func toolNames(m map[string]*mcpserver.ServerTool) []string {
	out := make([]string, 0, len(m))
	for n := range m {
		out = append(out, n)
	}
	return out
}
