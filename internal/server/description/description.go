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
