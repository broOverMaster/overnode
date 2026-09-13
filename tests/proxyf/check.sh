#!/usr/bin/env bash
set -euo pipefail

# Проверка работающего стенда; identities взяты из тестовых векторов RFC 8032.
proxy_address=${ACCEPTANCE_PROXY:-127.0.0.1:${ACCEPTANCE_PORT:-2080}}
node_key=3d4017c3e843895a92b70aa74d1b7ebc9c982ccf2ec4968cc0cd55f12af4660c
for suffix in ygg overspace; do
  for port in '' ':8081'; do
    target="${node_key}.${suffix}${port}"
    response=$(curl --fail --silent --show-error --max-time 15 --noproxy '' \
      -x "http://${proxy_address}" "http://${target}/marker?q=1")
    for expected in overlay-marker 'served-by=oversite-B'; do
      if ! grep -Fxq "$expected" <<< "$response"; then
        printf 'FAIL %s: missing %s\n' "$target" "$expected" >&2
        exit 1
      fi
    done
    status=$(curl --silent --show-error --max-time 15 --noproxy '' -x "http://${proxy_address}" \
      -o /dev/null -w '%{http_code}' --data 'acceptance-body' "http://${target}/marker")
    test "$status" = 200
    printf 'PASS %s\n' "$target"
  done
done
status=$(curl --silent --show-error --max-time 15 --noproxy '' -x "http://${proxy_address}" \
  -o /dev/null -w '%{http_code}' "http://invalid.ygg/")
test "$status" = 400
printf 'PASS invalid key invalid.ygg: 400\n'
status=$(curl --silent --show-error --max-time 15 --noproxy '' -x "http://${proxy_address}" \
  -o /dev/null -w '%{http_code}' "http://human.overspace/")
test "$status" = 404
printf 'PASS unknown name human.overspace: 404\n'
for target in local.overspace "${node_key}.ygg" "${node_key}.overspace"; do
  status=$(curl --silent --max-time 15 --noproxy '' -x "http://${proxy_address}" \
    -o /dev/null -w '%{http_connect}' "https://${target}/" || true)
  test "$status" = 405
  printf 'PASS CONNECT %s: 405\n' "$target"
done
curl --fail --silent --show-error --max-time 10 --noproxy '' -x "http://${proxy_address}" \
  http://local.overspace/ > /dev/null
curl --fail --silent --show-error --max-time 10 --noproxy '' -x "http://${proxy_address}" \
  http://oversite:8000/ > /dev/null
printf 'PASS local and direct internet regression\n'
