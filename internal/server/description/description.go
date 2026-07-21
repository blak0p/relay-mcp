package description

// create_terminal constants are the single source of truth for the tool's
// client-facing metadata.
const (
	// CreateTerminalName is the MCP tool name clients invoke.
	CreateTerminalName = "create_terminal"

	// CreateTerminalSummary is the one-line summary shown in tool listings.
	CreateTerminalSummary = "Spawn a persistent interactive bash session in a PTY."

	// CreateTerminalDescription is the full description sent in the tool
	// manifest. It tells the client what the tool does and how it behaves.
	CreateTerminalDescription = "Spawns a real bash process inside a PTY and returns a session handle. The session persists until close_terminal is called. If a session is already active, this call returns an error."
)

// write_terminal constants. These are the single source of truth for the
// tool name, summary, and full description registered with the MCP server
// (REQ-WT-007). The description states the two non-obvious contracts: the
// 1 MiB cap and the raw-byte (no auto-Enter) rule.
const (
	// WriteTerminalName is the MCP tool name clients invoke.
	WriteTerminalName = "write_terminal"

	// WriteTerminalSummary is the one-line summary shown in tool listings.
	WriteTerminalSummary = "Inject raw bytes into a running terminal session."

	// WriteTerminalDescription is the full description sent in the tool
	// manifest. It tells the client what the tool does and how it behaves.
	WriteTerminalDescription = "Writes raw bytes to the PTY of an active terminal session. The data is injected verbatim — no auto-Enter, no transformation. Maximum 1 MiB per call. Returns bytes_written and the current session state. Partial writes are not retried; the agent must resend the remainder."
)

// read_terminal constants are the single source of truth for the tool's
// client-facing metadata. The description states the default streaming
// contract and the explicit polling alternatives.
const (
	// ReadTerminalName is the MCP tool name clients invoke.
	ReadTerminalName = "read_terminal"

	// ReadTerminalSummary is the one-line summary shown in tool listings.
	ReadTerminalSummary = "Read retained output from the active terminal session."

	// ReadTerminalDescription is the full description sent in the tool
	// manifest. It tells clients how to select streaming or bounded reads.
	ReadTerminalDescription = "Streams terminal output through MCP progress notifications by default. Stream mode requires _meta.progressToken; use snapshot or drain for bounded polling reads. Cursor values are absolute byte positions in the retained 1 MiB output tail."
)
