#!/usr/bin/env bash
set -euo pipefail

fixture_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
compose=(docker compose -f "$fixture_dir/compose.yaml")
proxy_address=${ACCEPTANCE_PROXY:-127.0.0.1:${ACCEPTANCE_PORT:-2080}}
target=alice.overspace

restore() {
  "${compose[@]}" up -d --no-deps ygg-bro ygg-alice ygg-bob gate-bro gate-alice gate-bob site-bro site-alice site-bob
}
trap restore EXIT

check_failure() {
  status=$(curl --max-time 40 --silent --show-error --noproxy '' -x "http://${proxy_address}" \
    -o /dev/null -w '%{http_code}' "http://${target}/marker")
  case "$status" in
    502|504) printf 'PASS bounded overlay failure: %s\n' "$status";;
    *) printf 'FAIL unexpected overlay failure status: %s\n' "$status" >&2; exit 1;;
  esac
}

"${compose[@]}" stop ygg-alice
check_failure
restore
for attempt in {1..40}; do
  if curl -fsS --max-time 3 --noproxy '' -x "http://${proxy_address}" "http://${target}/marker" >/dev/null 2>&1; then break; fi
  sleep 1
done
bash "$fixture_dir/check.sh"
