.PHONY: live-smoke test build

live-smoke:
	@bash scripts/live-smoke.sh

test:
	go test ./...

build:
	go build ./cmd/...
