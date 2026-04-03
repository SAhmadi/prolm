BINARY := prolm
MODULE := github.com/prolm/prolm

.PHONY: build test vet lint run clean

build:
	go build -o $(BINARY) .

test:
	go test ./...

vet:
	go vet ./...

lint:
	@which staticcheck > /dev/null 2>&1 || (echo "staticcheck not found — run: go install honnef.co/go/tools/cmd/staticcheck@latest" && exit 1)
	staticcheck ./...

run:
	go run . $(ARGS)

clean:
	rm -f $(BINARY)
