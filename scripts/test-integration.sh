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

export DATABASE_URL="$(python3 - "$TABUNGAN_SISWA_DATABASE_DSN" <<'PY'
import sys
from urllib.parse import urlsplit, urlunsplit

source = urlsplit(sys.argv[1])
if source.scheme not in {"postgres", "postgresql"}:
    raise SystemExit("ERROR: TABUNGAN_SISWA_DATABASE_DSN harus berupa URL PostgreSQL.")
if not source.hostname or not source.path.strip("/"):
    raise SystemExit("ERROR: DSN harus memuat host dan nama database.")

# Preserve credentials, host, port, SSL/query options; change only the database.
print(urlunsplit((source.scheme, source.netloc, "/tabungan_siswa_fase168_clean_test", source.query, source.fragment)))
PY
)"

actual_db="$(psql "$DATABASE_URL" -X -A -t -v ON_ERROR_STOP=1 -c 'SELECT current_database();' | tr -d '[:space:]')"
if [[ "$actual_db" != "tabungan_siswa_fase168_clean_test" ]]; then
  echo "ERROR: database aktual tidak sesuai target test yang diizinkan: $actual_db" >&2
  exit 1
fi

echo "Integration tests target verified: $actual_db"
RUN_INTEGRATION_TESTS=1 go test ./... -count=1
