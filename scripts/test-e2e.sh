#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

if [[ ! -f .env ]]; then
  echo "ERROR: .env tidak ditemukan di root proyek." >&2
  exit 1
fi

set -a
# shellcheck disable=SC1091
source ./.env
set +a

if [[ -z "${TABUNGAN_SISWA_DATABASE_DSN:-}" ]]; then
  echo "ERROR: TABUNGAN_SISWA_DATABASE_DSN belum diatur di .env." >&2
  exit 1
fi

export TABUNGAN_SISWA_DATABASE_DSN="$(python3 - "$TABUNGAN_SISWA_DATABASE_DSN" <<'PY'
import sys
from urllib.parse import urlsplit, urlunsplit

source = urlsplit(sys.argv[1])
if source.scheme not in {"postgres", "postgresql"}:
    raise SystemExit("ERROR: TABUNGAN_SISWA_DATABASE_DSN harus berupa URL PostgreSQL.")
if not source.hostname or not source.path.strip("/"):
    raise SystemExit("ERROR: DSN harus memuat host dan nama database.")

print(urlunsplit((source.scheme, source.netloc, "/tabungan_siswa_fase168_clean_test", source.query, source.fragment)))
PY
)"

actual_db="$(psql "$TABUNGAN_SISWA_DATABASE_DSN" -X -A -t -v ON_ERROR_STOP=1 -c 'SELECT current_database();' | tr -d '[:space:]')"
if [[ "$actual_db" != "tabungan_siswa_fase168_clean_test" ]]; then
  echo "ERROR: database aktual tidak sesuai target E2E yang diizinkan." >&2
  exit 1
fi

export GOCACHE="${GOCACHE:-${TMPDIR:-/tmp}/tabungan-siswa-gocache}"

fixture_ready=0
cleanup() {
  exit_code=$?
  trap - EXIT
  if [[ "$fixture_ready" == "1" ]]; then
    if ! go run ./cmd/e2e-fixture cleanup; then
      echo "ERROR: fixture E2E belum terhapus; jalankan ulang helper cleanup pada database E2E." >&2
      exit_code=1
    fi
  fi
  exit "$exit_code"
}
trap cleanup EXIT

go run ./cmd/e2e-fixture setup
fixture_ready=1
./node_modules/.bin/playwright test --config=playwright.config.cjs
