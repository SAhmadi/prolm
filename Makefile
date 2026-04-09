BINARY := prolm
MODULE := github.com/prolm/prolm

.PHONY: build test test-race vet lint ci run clean

build:
	go build -o $(BINARY) .

test:
	go test ./...

test-race:
	go test -race -count=1 ./...

vet:
	go vet ./...

STATICCHECK := $(shell command -v staticcheck 2>/dev/null || echo $(shell go env GOPATH)/bin/staticcheck)

lint:
	@test -x "$(STATICCHECK)" || (echo "staticcheck not found — run: go install honnef.co/go/tools/cmd/staticcheck@latest" && exit 1)
	$(STATICCHECK) ./...

# ci mirrors the GitHub Actions workflow exactly — run this before opening a PR
ci: vet lint test-race

run:
	go run . $(ARGS)

clean:
	rm -f $(BINARY)
