# pickpocket

A CLI tool for managing vendored git clones as LLM context.

pickpocket lets you declare git repositories in a `pickpocket.json` file, clone them into a global cache, and give your LLM coding agents fast, local access to external codebases as context.

<img src="docs/assets/cover.png" alt="pickpocket cover" width="100%" /><p align="center"><sub><i>"If you don't watch out, someone might end up cloning you! I don't want that happening to me! I'm an original!" — Lupin the Third</i></sub></p>

## Installation

```bash
brew install wu-json/asahi/pickpocket
```

## Usage For Humans

1. **Declare** repos in a `pickpocket.json` file (the "Pickfile") at your project root
2. **Install** them with `pick install` — clones go into a shared global cache (`~/.pickpocket/`)
3. **Query** paths with `pick path --tag <tag>` so coding agents can read the code

Same repo + branch is only cloned once, even across multiple projects. Exact commit SHAs are pinned directly in the `pickpocket.json` file for reproducible setups across your team.

## Usage For Coding Agents

**`pick slash`** installs a slash command that teaches a coding agent how to use pickpocket:

```bash
pick slash claude   # Install /pick for Claude Code
pick slash codex    # Install /pick for Codex CLI
```

**`pick open <id>`** creates an ephemeral writable worktree in `/tmp/pickpocket/` — a playground copy of vendored code that agents can freely modify, build, and experiment in without touching the cached clone.

```bash
pick open my-lib          # Opens a worktree, prints the path
pick open --clean         # Remove all worktrees immediately
```

Worktrees auto-prune after 24 hours.

## Is Using OSS For Agent Context Stealing?

I equate the ethics behind this question to eating faster. Whether I choose to scarf down my food like [Kobayashi](https://en.wikipedia.org/wiki/Takeru_Kobayashi) in his prime or savor every last bite down to the bone, it doesn't say anything about how much I respect the food.

Perhaps eating faster makes me absorb less nutrients, and makes me feel sick afterwards, but at the end of the 
day the meal that the chef so graciously prepared is now in my stomach. Sometimes I eat fast when there's a lot of 
shit to do. Other times, I like taking my time.

Either way, I'm hungry and won't let it go to waste.

## Local Development

**Prerequisites:** Go 1.21+, git, [just](https://github.com/casey/just)

```bash
git clone https://github.com/wu-json/pickpocket.git && cd pickpocket
just build        # Build binary → ./pick
just test         # Run tests
just vet          # Run vet
```
