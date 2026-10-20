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
export GOCACHE="${GOCACHE:-${TMPDIR:-/tmp}/tabungan-siswa-gocache}"

go run ./cmd/e2e-fixture cleanup
