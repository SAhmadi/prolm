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

lint:
	@which staticcheck > /dev/null 2>&1 || (echo "staticcheck not found — run: go install honnef.co/go/tools/cmd/staticcheck@latest" && exit 1)
	staticcheck ./...

# ci mirrors the GitHub Actions workflow exactly — run this before opening a PR
ci: vet lint test-race

run:
	go run . $(ARGS)

clean:
	rm -f $(BINARY)
