all: pbmon

.PHONY: pbmon
pbmon:
	go build -o bin/pbmon ./cmd/pbmon/...
