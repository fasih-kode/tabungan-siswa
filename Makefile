.PHONY: dev test-integration bootstrap-users

dev:
	@set -a; . ./.env; set +a; go run ./cmd/web

test-integration:
	@./scripts/test-integration.sh

# Create the initial ADMIN and WALI_KELAS accounts in tabungan_siswa_dev.
bootstrap-users:
	@set -a; . ./.env; set +a; go run ./cmd/bootstrap-users
