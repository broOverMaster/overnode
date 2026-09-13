#!/usr/bin/env bash
set -euo pipefail

# Проверка обмена HTTP через все три overlay-узла.
proxy_address=${ACCEPTANCE_PROXY:-127.0.0.1:${ACCEPTANCE_PORT:-2080}}
declare -A expected=(
  [broOverMaster]=broOverMaster
  [alice]=alice
  [bob]=bob
)

for node in "${!expected[@]}"; do
  for suffix in ygg overspace; do
    target="${node}.${suffix}"
    response=$(curl --fail --silent --show-error --max-time 15 --noproxy '' \
      -x "http://${proxy_address}" "http://${target}/marker?q=1")
    grep -Fxq overlay-marker <<< "$response"
    grep -Fxq "served-by=${expected[$node]}" <<< "$response"
    printf 'PASS %s\n' "$target"
  done
done

curl --fail --silent --show-error --max-time 10 --noproxy '' \
  -x "http://${proxy_address}" http://local.overspace/ >/dev/null
printf 'PASS local route on broOverMaster\n'
