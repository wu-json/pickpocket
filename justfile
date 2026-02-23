version := `cat VERSION`
dry_run := "true"

build:
    go build -ldflags "-X github.com/wu-json/pickpocket/cmd.Version={{version}}" -o pick .

test:
    go test ./...

vet:
    go vet ./...

bump-and-commit-version bump_type:
    #!/usr/bin/env bash
    new_version=$(svu {{bump_type}})
    echo "$new_version" > VERSION
    git add -A
    git commit -m "chore(release): v$new_version"
    git tag -a v$new_version -m "Release v$new_version"
    git push --follow-tags

release:
    GORELEASER_CURRENT_TAG=v{{version}} goreleaser release --clean {{ if dry_run == "true" { "--snapshot" } else { "" } }}
