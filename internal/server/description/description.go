package description

// create_terminal constants. These are the only tool registered in v1; the
// remaining four tools (write_terminal, read_terminal, send_control,
// close_terminal) will add their own blocks here in follow-up SDDs.
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
