version := `cat VERSION`

build:
    go build -ldflags "-X github.com/wu-json/pickpocket/cmd.Version={{version}}" -o pick .

test:
    go test ./...

vet:
    go vet ./...
