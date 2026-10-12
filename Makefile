.PHONY: dev test-integration

dev:
	@set -a; . ./.env; set +a; go run ./cmd/web

test-integration:
	@./scripts/test-integration.sh
