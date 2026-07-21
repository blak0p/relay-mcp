# relay-mcp — Conventions

This document captures the conventions for working on relay-mcp. It is the single source of truth for "how we do things here". When in doubt, follow this file.

---

## 1. House Rules

How we get along when working together.

- **Be honest.** If you don't know, say so. Don't make things up. Verify before claiming. If you're wrong, own it with evidence.
- **Be brief.** Default to short answers. One thing at a time. Ask once and wait.
- **Match the user.** Mirror their language and tone. Adjust depth to their level.
- **Don't add what wasn't asked.** No scope creep. When a decision is needed, ask — don't decide for the user.
- **Respect the project.** Don't put private setup or agent config in the public repo. Keep the working tree tidy.
- **Push back with care.** Disagreement is fine. Use evidence and offer an alternative. Once the user decides, execute.

---

## 2. Module Layout

Each Go package owns one responsibility. Related packages group under a **namespace parent** that itself contains only documentation (`doc.go` and/or `README.md`, see Section 3).

### Established layout

```
internal/
├── session/                       # namespace parent for session lifecycle (docs only)
│   ├── session/                   # core types: Session, SessionState, New(), Close()
│   ├── registry/                  # Registry, NewRegistry, Put, Get
│   ├── liveness/                  # IsAlive(pid); reconcileState() helpers
│   └── error/                     # typed sentinel errors + helpers (ExistingSessionID, etc.)
│                                   # folder is singular; package name is `serror`
├── idgen/                         # flat (single concern, no sub-packages)
└── (future namespaces as needed)
```

### Rules

- One responsibility per package. If a new concern arises, create a new sub-package under the appropriate namespace parent, or a new namespace if none fits.
- The **namespace parent** (e.g., `internal/session/`) must not contain executable Go code. It may contain only `doc.go`, `README.md`, and other docs. Sub-packages are the real units of code.
- Package names use **singular, lowercase, no underscores** where possible: `session`, `registry`, `liveness`, `idgen`. Exceptions exist when the singular noun collides with a Go predeclared identifier (`error` → `serror`).
- Folder name equals package name, except for the documented exceptions.

### Why this layout

- **Discoverability**: the import path reflects the domain.
- **Testability**: each sub-package is independently importable; tests focus on one concern.
- **Refactor safety**: a change to one concern does not require touching unrelated code.
- **Clarity over brevity**: more folder depth, but each folder is small and obvious.

---

## 3. Package Documentation (Docs Pair)

Every Go package has **both** a `README.md` and a `doc.go`. They are NOT redundant; they serve different channels.

- **`README.md`** is for humans browsing GitHub. Rich markdown, links, usage examples, architecture diagrams.
- **`doc.go`** is for developers running `go doc` in their terminal. Idiomatic Go package doc comment — the first paragraph Go shows.

### Content split

- `README.md`: what is this package, when to use it, usage examples, links to related packages.
- `doc.go`: short package-level doc comment (one or two sentences). Pointers to the README for details are fine.

### Rules

- One without the other is incomplete.
- Do not duplicate content verbatim across the two.
- An empty `doc.go` (just `package foo` with no comment) is treated by Go as "no doc" and defeats the purpose.

### Anti-patterns

- README only (loses `go doc` integration).
- `doc.go` only (loses GitHub readability).
- Same content copy-pasted in both.
- No docs at all (the code does not self-document domain context).

### Documented exception: `package main` under `cmd/`

Binary entry points under `cmd/<name>/` typically contain a single `main.go`
holding both `package main` and `func main()`. The package doc comment lives
at the top of that `main.go` (idiomatic Go), and the README is the GitHub
landing page for that binary. A separate `doc.go` would be either empty
(`go doc` ignores files without a package comment above `package X`) or
duplicative. For this reason, `cmd/<name>/` is exempt from the docs pair
rule: the doc comment in `main.go` + the `README.md` next to it is the pair.

---

## 4. Commit Style

Conventional Commits. No scope parentheticals. No AI attribution.

### Format

```
<type>: <subject>

[optional body]
```

- **`<type>`** is one of: `feat`, `fix`, `chore`, `test`, `docs`, `refactor`, `style`, `perf`. No scope.
- **`<subject>`** is a short imperative description. Lowercase, no trailing period.
- **Body** is optional. Use it when the subject cannot carry the full message on its own.

### Examples

Good:

```
feat: generate unique session IDs with crypto/rand hex
```

```
refactor: split internal/session into focused sub-packages
```

```
fix: flip session state on Registry.Get when bash is gone
```

```
chore: replace package READMEs with idiomatic doc.go files
```

Bad:

```
feat(session): generate IDs with crypto/rand
```

→ Scope parenthetical adds nothing. Drop it.

```
chore(session,idgen): update READMEs
```

→ Multi-scope is a smell. If the change touches two packages, split into two commits.

```
feat: generate IDs

TDD: RED wrote idgen_test, GREEN passed, REFACTOR none.
Closes REQ-002.
```

→ Internal metadata and ticket references belong in the spec, not the commit body.

```
Co-Authored-By: Some AI <ai@example.com>
```

→ No AI attribution. No `Co-Authored-By` lines.

### Anti-patterns

- Scope parentheticals (`feat(scope):`)
- Multi-scope commits (`chore(a,b):`)
- `Co-Authored-By:` or AI attribution lines
- Long bodies that explain HOW the code works (the diff does that)
- Empty commit messages

---

## 5. Code Reuse (DRY Pragmatic)

Reuse when it is natural. Don't force coupling between genuinely different concerns.

- **Reuse** when two things do the same thing. Pull into a helper, a shared package, or a method.
- **Don't reuse** when two things only look similar but are conceptually distinct. Forcing them together creates coupling that hurts later.
- **Don't duplicate** (the WET anti-pattern): if you find yourself writing the same lines in two places, extract.
- **Don't over-abstract** (the DRY-strict anti-pattern): a 2-line "shared helper" with 3 callers is often worse than 3 callers with their own 2 lines.

### The test

> "If I change one, would the other break for the right reasons?"

If yes, they should share. If no, they are coincidentally similar — leave them separate.

### Heuristics

| Situation | Action |
|-----------|--------|
| Same package, same file, two methods doing the same | Merge immediately |
| Same package, different files, two functions doing the same | Extract to one file; both call |
| Different packages, same concept | Consider a shared package, but only if it is a real third thing |
| Different packages, similar but not identical | Leave as is. Comment the divergence if non-obvious |

---

## 6. When this document is wrong

This file is living. If a future situation genuinely contradicts a rule here, **the situation wins**. Update this file with a PR that explains what changed and why.

Rules that should not change without a lot of evidence:

- The namespace-parent + sub-packages layout (Section 2) — structural
- The docs pair convention (Section 3) — discoverability
- Conventional commits, no scope, no AI attribution (Section 4) — culture

Rules that are easier to revisit if needed:

- DRY vs. WET heuristics (Section 5) — judgment call
