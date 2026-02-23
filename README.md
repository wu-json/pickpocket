# pickpocket

A CLI tool for managing vendored git clones as LLM context.

pickpocket lets you declare git repositories in a `pickpocket.json` file, clone them into a global cache, and give your LLM coding agents fast, local access to external codebases as context.

<img src="docs/assets/cover.png" alt="pickpocket cover" width="100%" /><p align="center"><sub><i>Lupin the Third</i></sub></p>

## How it works

1. **Declare** repos in a `pickpocket.json` file (the "Pickfile") at your project root
2. **Install** them with `pick install` — clones go into a shared global cache (`~/.pickpocket/`)
3. **Query** paths with `pick path --tag <tag>` so coding agents can read the code

Same repo + branch is only cloned once, even across multiple projects. Exact commit SHAs are pinned directly in the `pickpocket.json` file for reproducible setups across your team.

## Using with coding agents

**`pick prompt`** outputs a system prompt that teaches a coding agent how to use pickpocket. Pipe it into your agent's context:

```bash
pick prompt >> .claude/CLAUDE.md
```

**`pick open <id>`** creates an ephemeral writable worktree in `/tmp/pickpocket/` — a playground copy of vendored code that agents can freely modify, build, and experiment in without touching the cached clone.

```bash
pick open my-lib          # Opens a worktree, prints the path
pick open --clean         # Remove all worktrees immediately
```

Worktrees auto-prune after 24 hours.

## Local development

**Prerequisites:** Go 1.21+, git, [just](https://github.com/casey/just)

```bash
git clone https://github.com/wu-json/pickpocket.git && cd pickpocket
just build        # Build binary → ./pick
just test         # Run tests
just vet          # Run vet
```

## Project structure

```
main.go                     # Entry point
cmd/                        # Cobra command handlers
  root.go                   #   Root command + pick <url> shortcut
  add.go                    #   pick add
  install.go                #   pick install
  path.go                   #   pick path
  list.go                   #   pick list
  tag.go                    #   pick tag (add/remove/list)
  update.go                 #   pick update
  remove.go                 #   pick remove
  open.go                   #   pick open (ephemeral worktrees)
  info.go                   #   pick info
  cache.go                  #   pick cache (list/remove/clean)
  init.go                   #   pick init
  system_prompt.go          #   pick prompt
internal/
  pickfile/                 # pickpocket.json read/write/discovery
  git/                      # Git CLI wrapper (clone, worktrees, branch detection)
  giturl/                   # URL parsing and normalization
  cache/                    # Global cache index (~/.pickpocket/cache.json)
```
