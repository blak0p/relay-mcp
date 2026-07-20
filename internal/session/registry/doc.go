// Package registry owns the thread-safe single-session store used by the MCP
// server. MCP over stdio serves exactly one client, so a single slot is
// sufficient; the store is a thin wrapper that guarantees mutual exclusion and
// performs lazy liveness reconciliation on Get.
//
// Liveness: Registry.Get reconciles the stored session state before
// returning, so callers always see up-to-date state. See the liveness
// sub-package.
package registry
