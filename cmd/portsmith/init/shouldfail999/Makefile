.PHONY: run test gen mock check

run:
	go run ./cmd/server

test:
	go test ./...

gen:
	portsmith gen ./internal/...

mock:
	portsmith mock ./internal/...

check:
	portsmith check ./internal/...
