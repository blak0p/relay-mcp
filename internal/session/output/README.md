# output

`output` retains a bounded tail of bytes captured from one terminal. Readers use
absolute cursors to take immutable snapshots without consuming the shared buffer.

`Broker` keeps the newest bytes within its configured capacity. If a requested
cursor is older than the retained range, `Snapshot` reports the exact gap in
`DroppedBytes` and begins at the oldest available cursor. Waiting snapshots wake
when output arrives or the terminal lifecycle status changes.
