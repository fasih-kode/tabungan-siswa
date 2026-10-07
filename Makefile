.PHONY: dev

dev:
	@set -a; . ./.env; set +a; go run ./cmd/web
