# pickpocket - Vendored Clone Manager for LLM Context

**Date:** 2026-02-22
**Status:** Draft

## Overview

pickpocket is a CLI tool for managing vendored git clones intended to be used as LLM context for coding agents. It is project-centric: each project has a **Pickfile** (`.pickpocket`) that declares which repos it needs. Clones are stored in a global cache (`~/.pickpocket/`) for deduplication — if two projects need the same repo at the same branch, it's only cloned once.

The CLI provides a beautiful, well-animated command-line experience.

## Core Concepts

- **Pick**: A vendored git clone managed by pickpocket.
- **Pickfile**: The primary manifest — a `.pickpocket` file in a project's root, committed to version control. Declares which repos the project uses for context, along with tags and branch preferences. Teammates run `pick install` to sync.
- **Lockfile**: A `.pickpocket.lock` file alongside the Pickfile, also committed to version control. Records the exact commit SHA for each pick at the time of install/update. This ensures every contributor gets the same code at the same point in time. Think `package-lock.json` or `go.sum`.
- **Cache**: The global directory (`~/.pickpocket/`) where cloned repos live on disk. This is a shared, deduplicated cache. Each unique repo+branch combination gets its own clone. If a repo at the same branch is already cached from another project, it's reused instantly.
- **Tags**: User-defined labels attached to picks in the Pickfile, used for filtering and querying.

## Directory Layout

```
project/
  .pickpocket              # the Pickfile — checked into version control
  .pickpocket.lock         # the Lockfile — checked into version control

~/.pickpocket/             # global cache
  cache.json               # internal index of cached repos and their state
  repos/
    github.com/
      owner/
        repo/
          main/            # full clone, checked out at the "main" branch
          develop/          # separate clone for the "develop" branch
```

Each branch gets its own full clone so that multiple projects (or the same project) can reference different branches of the same repo without conflicts.

## Pickfile Schema

The `.pickpocket` file is the source of truth for a project. It lives in the project root and is checked into version control.

```jsonc
{
  "picks": [
    {
      "url": "https://github.com/anthropics/claude-code.git",
      "branch": "main",
      "tags": ["agent", "cli"]
    },
    {
      "url": "https://github.com/charmbracelet/bubbletea.git",
      "tags": ["tui", "go"]
    },
    {
      "url": "https://github.com/spf13/cobra.git"
    }
  ]
}
```

Fields per entry:
- `url` (required) — the git clone URL.
- `branch` (optional) — branch to track. Defaults to the repo's default branch.
- `tags` (optional) — labels for filtering and querying.

Design goals:
- **Minimal and diff-friendly** — easy to hand-edit, review in PRs, and merge.
- **Declarative** — describes *what* the project needs, not *where* it lives on disk or *when* it was fetched. All of that is the cache's problem.

## Lockfile Schema

The `.pickpocket.lock` file sits alongside the Pickfile and is also committed to version control. It pins the exact commit SHA for each pick, ensuring deterministic, reproducible setups across all contributors.

```jsonc
{
  "locked": [
    {
      "url": "https://github.com/anthropics/claude-code.git",
      "branch": "main",
      "commit": "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"
    },
    {
      "url": "https://github.com/charmbracelet/bubbletea.git",
      "branch": "main",
      "commit": "f6e5d4c3b2a1f6e5d4c3b2a1f6e5d4c3b2a1f6e5"
    },
    {
      "url": "https://github.com/spf13/cobra.git",
      "branch": "main",
      "commit": "1234567890abcdef1234567890abcdef12345678"
    }
  ]
}
```

Lifecycle:
- **Created/updated** by `pick add`, `pick install`, and `pick update`. Any operation that resolves a commit writes to the lockfile. `pick add` appends or updates only the entry for the added pick; it does not rewrite unrelated entries.
- **Consumed** by `pick install`. When a lockfile exists, `pick install` checks out the exact locked commits rather than fetching the branch tip. This means a fresh `pick install` is fully deterministic.
- **Explicitly updated** by `pick update`. Running `pick update` fetches latest and rewrites the lockfile with new SHAs. This is an intentional action — the diff shows up in version control for review.

This separation mirrors the Pickfile/Lockfile split in package managers:
- The **Pickfile** says "I want this repo on this branch" (intent).
- The **Lockfile** says "last time we resolved that, the commit was X" (fact).

## Cache Index Schema

The global cache index (`~/.pickpocket/cache.json`) is an internal implementation detail. Users don't edit it. It tracks the state of each cached clone.

```jsonc
{
  "version": 1,
  "repos": [
    {
      "id": "github.com/owner/repo@main",
      "url": "https://github.com/owner/repo.git",
      "path": "repos/github.com/owner/repo/main",
      "branch": "main",
      "commit": "abc123...",
      "cloned_at": "2026-02-22T10:00:00Z",
      "updated_at": "2026-02-22T10:00:00Z"
    }
  ]
}
```

Key fields:
- `id` — derived from the normalized URL plus the branch name (e.g., `github.com/owner/repo@main`). This is the unique key for cache deduplication. The same repo at different branches produces separate cache entries.
- `path` — relative path within `~/.pickpocket/` to the clone.
- `commit` — SHA of the latest fetched commit on the tracked branch.
- `cloned_at` / `updated_at` — timestamps for cache management.

Note: tags are **not** stored in the cache index. Tags are project-specific and live in the Pickfile. The same repo cached globally might have different tags in different projects.

## CLI Design

Written in Go. All commands output with animated spinners, progress bars, and color. Errors are clearly formatted. The CLI should feel polished — think `gh`, `charm` tooling, etc. Use libraries like [charmbracelet/lipgloss](https://github.com/charmbracelet/lipgloss), [charmbracelet/bubbles](https://github.com/charmbracelet/bubbles) (spinner/progress), and [cobra](https://github.com/spf13/cobra) for the CLI framework.

All commands that operate on picks use the **project's Pickfile** as the scope by default. The global cache is an implementation detail.

### Commands

#### `pick add <url> [flags]`

Add a repo to the project's Pickfile and ensure it's cached locally.

```
pick add https://github.com/anthropics/claude-code
pick add git@github.com:anthropics/claude-code.git
pick add https://github.com/anthropics/claude-code --tag agent --tag cli --branch main
```

Behavior:
1. Find the nearest `.pickpocket` file (or create one in the current directory if none exists).
2. Parse and normalize the URL (see [URL Normalization](#url-normalization)).
3. Check the Pickfile for duplicates (same normalized URL **and** same branch).
4. Add the entry to the Pickfile.
5. If the repo+branch isn't already in the global cache, clone it (with animated progress).
6. If it is cached, print that it was already available — no network needed.
7. Append or update the entry for this pick in the lockfile (`.pickpocket.lock`) with the resolved commit SHA.

Flags:
- `--tag, -t` (repeatable) — tags to attach to the pick.
- `--branch, -b` — branch to track (default: repo default).

#### `pick install`

Ensure all picks declared in the Pickfile are cached locally. This is the command teammates run after cloning a project.

```
pick install
```

Behavior:
1. Find the nearest `.pickpocket` file by walking up from the current directory.
2. If a `.pickpocket.lock` file exists, use the locked commit SHAs — this ensures deterministic installs.
3. For each entry:
   - If already cached at the correct commit, skip (print "cached").
   - If cached but at a different commit (and lockfile specifies one), check out the locked commit.
   - If not cached, clone into the global cache and check out the locked commit (with animated progress).
4. If no lockfile exists, clone/fetch branch tips and **create** the lockfile with the resolved commits.
5. Display a summary on completion.

#### `pick list [flags]`

List picks in the current project's Pickfile.

```
pick list
pick list --tag frontend
pick list --tag frontend --tag react   # AND filter
```

Output: a formatted table showing URL (or short id), branch, commit (short SHA from cache), tags, and last updated time (from cache).

Flags:
- `--tag, -t` (repeatable) — filter by tags (AND logic).
- `--json` — output as JSON for scripting.

#### `pick update [id...] [flags]`

Fetch the latest commits for picks in the current project.

```
pick update                              # update all project picks
pick update github.com/anthropics/claude-code@main
pick update --tag agent                  # update project picks with tag
```

Behavior:
1. Resolve which picks to update from the Pickfile (all, by id, or by tag filter).
2. `git fetch` + `git reset --hard` to latest on tracked branch, in parallel where possible.
3. Update cache index with new commit SHA and `updated_at`.
4. **Rewrite the lockfile** with the new commit SHAs. The lockfile diff is visible in version control for review.
5. Display per-repo animated progress and a summary of what changed (old SHA -> new SHA).

Flags:
- `--tag, -t` (repeatable) — only update picks matching these tags.

#### `pick remove <id> [flags]`

Remove a pick from the project's Pickfile.

```
pick remove github.com/anthropics/claude-code@main
```

Behavior:
1. Remove the entry from the Pickfile.
2. Remove the corresponding entry from the lockfile.
3. The cached clone is **not** deleted (other projects may use it). Use `pick cache remove` or `pick cache clean` to manage cache storage directly.

Flags:
- `--force, -f` — skip confirmation.

#### `pick tag <subcommand>`

Manage tags on picks in the current project's Pickfile.

##### `pick tag add <id> <tags...>`

```
pick tag add github.com/anthropics/claude-code@main agent cli
```

##### `pick tag remove <id> <tags...>`

```
pick tag remove github.com/anthropics/claude-code@main cli
```

##### `pick tag list`

List all tags in use in the current project's Pickfile, with counts.

```
pick tag list
```

Output:
```
TAG         PICKS
agent       3
frontend    5
react       2
```

#### `pick path [flags]`

Output the filesystem paths of cached clones for picks in the current project. This is the primary integration point for coding agents.

```
pick path --tag agent
pick path github.com/anthropics/claude-code@main
pick path                                 # all paths
```

Output: newline-separated absolute paths, suitable for piping and scripting.

Flags:
- `--tag, -t` (repeatable) — filter by tags.
- `--json` — output as JSON array.

#### `pick info <id>`

Show detailed info for a single pick in the current project.

```
pick info github.com/anthropics/claude-code@main
```

Output: formatted display of Pickfile entry (url, branch, tags) plus cache state (full commit SHA, disk path, timestamps, disk size).

#### `pick doctor`

Health check and stats overview, scoped to the current project's picks.

```
pick doctor
```

Output: a styled dashboard showing:

```
pickpocket doctor

  Project      /Users/you/myproject/.pickpocket
  Picks        12 entries, 8 tags

  Cache        ~/.pickpocket/
  Disk Usage
  Total        1.34 GB  (this project's picks)
  Largest      github.com/chromium/chromium@main   892 MB
  Smallest     github.com/charmbracelet/glow@main    4 MB
  Average      112 MB

  Top Tags
  agent        5 picks    620 MB
  frontend     3 picks    210 MB
  cli          4 picks    510 MB

  Health
  ✓ All picks are cached
  ✓ All cached repos have valid git state
  ✗ 2 picks have not been updated in over 30 days
    - github.com/some/old-repo@main (47 days)
    - github.com/another/stale-one@main (33 days)
```

Checks performed:
- **Cache completeness** — every Pickfile entry has a corresponding cached clone.
- **Git state** — each cached clone has a valid `.git` directory and the tracked branch exists.
- **Staleness** — flag picks that haven't been updated in over 30 days.
- **Disk usage** — compute per-repo, per-tag, and total size for this project's picks.

#### `pick cache <subcommand>`

Manage the global cache directly. These commands operate on `~/.pickpocket/`, independent of any project Pickfile.

##### `pick cache list`

List all repos in the global cache.

```
pick cache list
```

Output: a table showing id, branch, commit (short), disk size, and last updated time.

##### `pick cache remove <id> [flags]`

Remove a specific repo from the global cache.

```
pick cache remove github.com/anthropics/claude-code@main
```

Behavior:
1. Confirm removal (unless `--force`).
2. Delete the cloned directory from the cache.
3. Remove the entry from `cache.json`.
4. Warn if the repo is referenced by the current project's Pickfile (it will need to be re-fetched on next `pick install`).

Flags:
- `--force, -f` — skip confirmation.

##### `pick cache clean [flags]`

Nuclear option: wipe the entire global cache.

```
pick cache clean
```

Behavior:
1. Display total cache size and repo count.
2. Confirm (unless `--force`).
3. Delete all repos and reset `cache.json`.

Flags:
- `--force, -f` — skip confirmation.

#### `pick init`

Initialize a `.pickpocket` file in the current directory.

```
pick init
```

Behavior:
1. Create an empty `.pickpocket` file (`{"picks": []}`) in the current working directory.
2. If one already exists, print a message and exit (no overwrite).

This is optional — `pick add` will also create the file if needed.

## UX / Animation Guidelines

- **Spinners** on all network/git operations (clone, fetch). Use charmbracelet spinner styles.
- **Progress bars** for multi-repo operations (`pick install`, `pick update` with multiple targets).
- **Color-coded output**: green for success, yellow for warnings (e.g., already cached), red for errors.
- **Tables** for list output, using lipgloss for styling.
- **Confirmation prompts** for destructive actions (remove, cache clean), styled consistently.
- Fast operations (tag manipulation, path output) should not have spinners — just print results immediately.

## Claude Code Agent Skill

Provide a Claude Code slash command / agent skill that teaches the coding agent how to use pickpocket to discover relevant vendored context.

The skill should be installable (as a `.md` file in the user's Claude Code skills directory or similar mechanism) and should instruct the agent to:

1. Run `pick list --json` to discover available picks and their tags for the current project.
2. Use `pick path --tag <tag>` to get filesystem paths for relevant cached clones.
3. Explore those paths using file reading and grep tools to gather context.
4. Reference the vendored code when answering questions or generating code.

Example skill prompt (rough draft):

```markdown
# Skill: pickpocket-context

When the user asks you to reference external code, or when you need context from
a vendored repository:

1. Run `pick list --json` to see this project's vendored repositories and their tags.
2. Identify which repositories are relevant based on tags and names.
3. Run `pick path --tag <relevant-tag>` to get filesystem paths.
4. Use Read and Grep tools to explore the code at those paths.
5. Use what you find as context for your response.

If no picks are available, inform the user they can add repos with:
  pick add <github-url> --tag <tag>
```

The exact skill format and installation mechanism should follow whatever Claude Code supports at build time.

## Technical Decisions

- **Language**: Go
- **CLI framework**: Cobra
- **TUI/styling**: charmbracelet lipgloss, bubbles (spinner, progress)
- **Clone type**: Full clones (not bare) so agents can read working tree files directly.
- **Parallelism**: Multi-repo operations (install, update) should run concurrently with a sensible concurrency limit.
- **Cache locking**: Use a lockfile (`~/.pickpocket/cache.lock`) during writes to avoid races from concurrent `pick install` in different projects.
- **Pickfile discovery**: Walk up from cwd to find the nearest `.pickpocket` file (similar to how git finds `.git`).

### URL Normalization

All URLs are normalized before use in the Pickfile, lockfile, cache index, and dedup checks. The rules:

1. **Strip `.git` suffix** — `https://github.com/owner/repo.git` becomes `https://github.com/owner/repo`.
2. **Normalize SSH to HTTPS** — `git@host:owner/repo` becomes `https://host/owner/repo`.
3. **Derive the cache ID** — strip the scheme to produce `host/owner/repo`, then append `@branch` (e.g., `github.com/owner/repo@main`).

This means `git@github.com:owner/repo.git` and `https://github.com/owner/repo` resolve to the same cache entry (given the same branch).

The **stored URL** in the Pickfile and lockfile is always the normalized HTTPS form.

Note: v1 is tested against GitHub-hosted repos only. The normalization rules are designed to be host-agnostic so that other git hosts (GitLab, Bitbucket, self-hosted, etc.) work in the future without schema changes.

## Future Considerations (Out of Scope for v1)

- Sparse checkouts / partial clones for large repos.
- Pinning to a specific commit or git tag in the Pickfile (not just branches).
- `pick search` for searching across all picked repos.
- Auto-update on a schedule.
- Config file for defaults (concurrency, default tags, etc.).
- Support for non-git sources (tarballs, zip archives).
- First-class support for non-GitHub git hosts (GitLab, Bitbucket, self-hosted). URL normalization is host-agnostic by design; this is about testing and any host-specific UX (e.g., shorthand aliases).
