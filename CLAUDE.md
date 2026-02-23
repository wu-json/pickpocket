# CLAUDE.md

## Project overview

Pickpocket is a Go CLI tool (`pick`) for managing vendored git clones as LLM context. Users declare repos in a `pickpocket.json` file, install them into a global cache (`~/.pickpocket/`), and query paths for coding agents.

## Tech stack

- **Language:** Go 1.25.5
- **CLI framework:** cobra
- **Terminal styling:** lipgloss
- **Task runner:** just (see `justfile`)
- **Tooling:** aqua (tool version manager), goreleaser (releases), svu (semantic versioning)

## Common commands

```bash
just build    # Build binary with version injection → ./pick
just test     # go test ./...
just vet      # go vet ./...
```

## Project structure

```
main.go              # Entry point
cmd/                 # Cobra command handlers (add, install, path, list, tag, update, cache, info, open, remove, init)
internal/
  pickfile/          # pickpocket.json read/write/discovery
  git/               # Git CLI wrapper (clone, branch detection)
  giturl/            # URL parsing and normalization
  cache/             # Global cache index (~/.pickpocket/cache.json)
```

## Code conventions

- Standard Go formatting (gofmt)
- Table-driven tests with `t.TempDir()` for isolation
- Error wrapping with `fmt.Errorf("context: %w", err)`
- JSON field tags use snake_case
- Atomic file writes via temp file + rename pattern
- Version injected at build time via ldflags (`cmd.Version`)
