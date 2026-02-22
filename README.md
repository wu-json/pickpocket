# pickpocket

A CLI tool for managing vendored git clones as LLM context.

pickpocket lets you declare git repositories in a `.pickpocket` file, clone them into a global cache, and give your LLM coding agents fast, local access to external codebases as context.

## How it works

1. **Declare** repos in a `.pickpocket` file (the "Pickfile") at your project root
2. **Install** them with `pick install` — clones go into a shared global cache (`~/.pickpocket/`)
3. **Query** paths with `pick path --tag <tag>` so coding agents can read the code

Same repo + branch is only cloned once, even across multiple projects. Exact commit SHAs are pinned directly in the `.pickpocket` file for reproducible setups across your team.

## Local development

**Prerequisites:** Go 1.21+, git, [just](https://github.com/casey/just)

```bash
# Clone the repo
git clone https://github.com/wu-json/pickpocket.git
cd pickpocket

# Install dependencies
go mod download

# Build (injects version from VERSION file via ldflags)
just build

# Run
./pick --help

# Check version
./pick version

# Run tests
just test

# Vet
just vet
```

You can also use plain `go build -o pick .` — the version will show as `dev`.

## Project structure

```
main.go                     # Entry point
cmd/
  root.go                   # Cobra root command
internal/
  giturl/                   # URL parsing and normalization
  git/                      # Git CLI wrapper (clone, branch detection)
  pickfile/                 # .pickpocket file read/write/discovery
cache/                    # Global cache index (~/.pickpocket/cache.json)
```

## License

MIT
