# Security Policy

## Reporting a vulnerability

relay-mcp spawns real PTY sessions on the host. A vulnerability could allow
arbitrary command execution. If you find one, **do not open a public issue**.

Report it privately to the maintainers by email or via the security tab on
GitHub. We will acknowledge receipt within 48 hours and work on a fix before
disclosure.

## Scope

- The relay-mcp binary itself (Go code)
- The MCP protocol implementation
- PTY lifecycle and process isolation

Out of scope: the MCP client (Claude, etc.) that drives the server.
