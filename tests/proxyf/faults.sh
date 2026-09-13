#!/usr/bin/env bash
set -euo pipefail
# Проверка отказов останавливает только тестовые overlay-сервисы и восстанавливает их.
fixture_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
compose=(docker compose -f "$fixture_dir/compose.yaml")
proxy_address=${ACCEPTANCE_PROXY:-127.0.0.1:${ACCEPTANCE_PORT:-2080}}
target=3d4017c3e843895a92b70aa74d1b7ebc9c982ccf2ec4968cc0cd55f12af4660c.ygg
restore() {
  "${compose[@]}" up -d --no-deps overlay-site overlay-node overlay-gate yggstack
}
trap restore EXIT
check_failure() {
  status=$(curl --max-time 40 --silent --show-error --noproxy '' -x "http://${proxy_address}" \
    -o /dev/null -w '%{http_code}' "http://${target}/marker")
  case "$status" in 502|504) printf 'PASS bounded overlay failure: %s\n' "$status";; *) exit 1;; esac
  curl -fsS --max-time 10 --noproxy '' -x "http://${proxy_address}" http://local.overspace/ >/dev/null
  curl -fsS --max-time 10 --noproxy '' -x "http://${proxy_address}" http://oversite:8000/ >/dev/null
}
"${compose[@]}" stop yggstack
check_failure
restore
for attempt in {1..40}; do
  if curl -fsS --max-time 3 --noproxy '' -x "http://${proxy_address}" "http://${target}/marker" >/dev/null 2>&1; then break; fi
  sleep 1
done
curl -fsS --max-time 15 --noproxy '' -x "http://${proxy_address}" "http://${target}/marker" >/dev/null
"${compose[@]}" stop overlay-gate overlay-node
check_failure
restore
for attempt in {1..40}; do
  if curl -fsS --max-time 3 --noproxy '' -x "http://${proxy_address}" "http://${target}/marker" >/dev/null 2>&1; then break; fi
  sleep 1
done
bash "$fixture_dir/check.sh"
for service in overlay-gate overlay-site; do
  "${compose[@]}" stop "$service"
  check_failure
  restore
  for attempt in {1..40}; do
    if curl -fsS --max-time 3 --noproxy '' -x "http://${proxy_address}" "http://${target}/marker" >/dev/null 2>&1; then break; fi
    sleep 1
  done
  bash "$fixture_dir/check.sh"
done
trap - EXIT
