# Relay-mcp

Servidor MCP de terminal interactiva para agentes de IA.

**Relay** porque el agente delega comandos, vos los ejecutás en un PTY real y devolvés el resultado. Como un relevo.

## Filosofía

No queremos un monstruo de 27 tools como Forge. Queremos **5 tools, bien hechas**, que cubran el 90% de la interacción real con un agente:

1. `create_terminal` — spawnear PTY
2. `write_terminal` — mandar input
3. `read_terminal` — leer output incremental
4. `send_control` — Ctrl+C, flechas, Tab, etc.
5. `close_terminal` — matar sesión

Después, si hace falta, agregamos `read_screen` (viewport renderizado sin ANSI) y `run_command` (one-shot con auto-cleanup).

## Por qué Go

- Un binario estático, cero runtime
- `github.com/creack/pty` — PTY nativo, maduro
- `github.com/mark3labs/mcp-go` — SDK MCP para Go
- Ring buffer, control chars, manejo de procesos: todo stdlib
- Lo que sea complejo (ANSI parser) se puede embeber via WASM o hacerlo después

## Arquitectura

```
┌─────────────────────────────────────────────┐
│           relay-mcp (stdin/stdout)           │
│                                             │
│  ┌─────────────────────────────────────┐    │
│  │         MCP Server (JSON-RPC)        │    │
│  │                                       │    │
│  │  create_terminal  write_terminal     │    │
│  │  read_terminal    send_control       │    │
│  │  close_terminal                      │    │
│  └──────────┬──────────────────────────┘    │
│             │                                │
│  ┌──────────▼──────────────────────────┐    │
│  │         Session Manager              │    │
│  │  Map[string]*TerminalSession        │    │
│  │  maxSessions, idleTimeout           │    │
│  └──────────┬──────────────────────────┘    │
│             │                                │
│  ┌──────────▼──────────────────────────┐    │
│  │      TerminalSession                 │    │
│  │  ┌──────────┬──────────┬─────────┐  │    │
│  │  │ creack/pty│ RingBuf  │ Control │  │    │
│  │  │ (PTY real)│(circular)│ Chars   │  │    │
│  │  └──────────┴──────────┴─────────┘  │    │
│  └─────────────────────────────────────┘    │
└─────────────────────────────────────────────┘
```

### Flujo de datos

```
Agente IA
   │ create_terminal({ command: "bash" })
   ▼
relay-mcp ──► creack/pty ──► proceso (bash / python / lazygit)
   │                              │
   │  write_terminal              │ stdout
   │  "echo hola\n" ────────────► │
   │                              │
   │  read_terminal ◄──────────── │ "hola\n"
   │                              │
   │  send_control("ctrl+c") ───► │ SIGINT
   │                              │
   │  close_terminal ───────────► │ kill
```

## Componentes

### 1. PTY Adapter (interfaz)

```go
type PtyProcess interface {
    Pid() int
    Write(data []byte) (int, error)
    Read(buf []byte) (int, error)
    Resize(cols, rows uint16) error
    Kill() error
    Wait() error
}
```

Implementación concreta vía `creack/pty` + `os/exec`.

### 2. Ring Buffer (circular, multi-consumer)

Misma idea que Forge pero en Go:

- Buffer circular de []byte con capacidad fija (1MB default)
- `writeOffset` monotónico que nunca se reinicia
- Multiples consumidores con cursor individual
- `Read(consumerID)` → solo datos nuevos desde último read
- `droppedBytes` si el consumidor se atrasó por wrap-around

### 3. Terminal Session

Wrappeo del PTY + RingBuffer:

```go
type TerminalSession struct {
    ID        string
    Command   string
    Status    SessionStatus
    pid       int
    pty       PtyProcess
    ringBuf   *RingBuffer
    cmd       *exec.Cmd
    mu        sync.Mutex
}
```

El loop de lectura corre en una goroutine:

```go
go func() {
    buf := make([]byte, 4096)
    for {
        n, err := session.pty.Read(buf)
        if err != nil {
            break
        }
        session.ringBuf.Write(buf[:n])
    }
}()
```

### 4. Session Manager

```go
type SessionManager struct {
    mu       sync.RWMutex
    sessions map[string]*TerminalSession
    max      int
}
```

CRUD básico: Create, Get, Close, List.

### 5. MCP Server (JSON-RPC sobre stdio)

Usando `mark3labs/mcp-go` o implementando el protocolo a mano:

```go
// Cada tool se registra con:
// - Name: "create_terminal"
// - Description: "..."
// - InputSchema: JSON Schema de parámetros
// - Handler: func(args json.RawMessage) (json.RawMessage, error)
```

### 6. Control Chars

```go
var ControlMap = map[string][]byte{
    "ctrl+c":    {0x03},
    "ctrl+d":    {0x04},
    "up":        {0x1B, '[', 'A'},
    "down":      {0x1B, '[', 'B'},
    "tab":       {0x09},
    "enter":     {0x0D},
    "escape":    {0x1B},
    "backspace": {0x7F},
    // ...
}
```

## Código de referencia

El directorio `original-repo/` contiene el código de Forge v0.9.0 (Node.js/TypeScript) con los archivos clave que inspiran esta arquitectura:

| Archivo original | Equivalente Go | Notas |
|---|---|---|
| `server.ts` | `cmd/relay-mcp/main.go` + `server/handlers.go` | Tools MCP + handlers |
| `terminal-session.ts` | `session/session.go` | PTY + ring buffer loop |
| `ring-buffer.ts` | `ringbuf/ringbuf.go` | Circular incremental |
| `session-manager.ts` | `manager/manager.go` | CRUD + lifecycle |
| `pty-adapter.ts` | `pty/pty.go` | Interfaz abstracta |
| `control-chars.ts` | `control/control.go` | Key mapping |
| `types.ts` | `types/types.go` | Structs compartidos |

## Plan de implementación

### Fase 1: Core (esta sesión)
1. Proyecto Go base + módulo
2. Ring buffer con tests
3. PTY adapter con creack/pty
4. Terminal session (goroutine de lectura)
5. Session manager
6. MCP server mínimo (stdin/stdout JSON-RPC)
7. 5 tools funcionando

### Fase 2: Robustez
- Manejo de errores consistente
- Idle timeout
- Logging estructurado
- Max sessions

### Fase 3: Extras (si hacen falta)
- `read_screen` con parseo ANSI básico
- `run_command` (one-shot)
- `wait_for` pattern
- `grep_terminal`

## Vs Forge

| Aspecto | Forge (Node) | Relay-mcp (Go) |
|---|---|---|
| Runtime | Node.js ≥ 18 | Binario estático |
| Dependencias | 98 packages npm | 0-3 deps Go |
| Tools | 27 | 5 (core) + extras opcionales |
| PTY | node-pty | creack/pty |
| Ring buffer | Buffer nativo | []byte + sync |
| VT Emulator | @xterm/headless (completo) | Sin emulador (fase 3) |
| Tamaño | ~50MB con node_modules | ~10MB binario |
| Instalación | npm install -g | go install / descargar bin |
