# control

`control` is the single source of truth for the terminal-control allowlist.
It resolves a case-insensitive, surrounding-whitespace-tolerant key name to a
canonical key and exact bytes suitable for writing to the active PTY.

## Allowlist

| Key | Bytes (hex) |
|---|---|
| `ctrl+c` | `03` |
| `ctrl+d` | `04` |
| `ctrl+z` | `1a` |
| `ctrl+\` | `1c` |
| `ctrl+l` | `0c` |
| `ctrl+a` | `01` |
| `ctrl+e` | `05` |
| `ctrl+k` | `0b` |
| `ctrl+u` | `15` |
| `ctrl+w` | `17` |
| `ctrl+r` | `12` |
| `tab` | `09` |
| `enter` | `0d` |
| `escape` | `1b` |
| `backspace` | `7f` |
| `up` | `1b 5b 41` |
| `down` | `1b 5b 42` |
| `right` | `1b 5b 43` |
| `left` | `1b 5b 44` |
| `home` | `1b 5b 48` |
| `end` | `1b 5b 46` |
| `delete` | `1b 5b 33 7e` |
| `page_up` | `1b 5b 35 7e` |
| `page_down` | `1b 5b 36 7e` |

`Resolve` rejects every key outside this list. Returned `Sequence.Bytes` and
the `SupportedKeys` slice are copies, so callers cannot mutate package state.
