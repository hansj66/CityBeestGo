BINARY := citybeest
CMD_PATH := ./cmd/citybeest

.PHONY: all build vet lint check clean

all: check build

build:
	go build -o $(BINARY) $(CMD_PATH)

lint:
	@echo "*** $@"
	@revive ./...

staticcheck:
	@echo "*** $@"
	@staticcheck ./...

check: lint staticcheck

install-deps:
	@go install github.com/mgechev/revive@latest
	@go install honnef.co/go/tools/cmd/staticcheck@latest

clean:
	rm -f $(BINARY)
