all: pbmon

.PHONY: pbmon
pbmon:
	go build -o bin/pbmon ./cmd/pbmon/...

.PHONY: run
run: tpl
	go run ./cmd/pbmon/*.go

.PHONY: tpl
tpl:
	@go-bindata \
	    -o tpl/bindata.go \
	    -pkg=tpl \
	    -prefix=tpl \
	    -tags build_bindata \
	    tpl/*.html
