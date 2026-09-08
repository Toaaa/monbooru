#!/bin/sh
# packaging/prefer-ipv4.sh <probe-url>

set -eu

probe=$1
host=$(echo "$probe" | sed -e 's|^[a-z]*://||' -e 's|/.*||')

getent hosts "$host" || true
v4=0
v6=0
curl -4 -fsS --max-time 30 -o /dev/null \
  -w 'ipv4: %{http_code} in %{time_total}s from %{remote_ip}\n' "$probe" && v4=1 || echo "ipv4: no answer"
curl -6 -fsS --max-time 30 -o /dev/null \
  -w 'ipv6: %{http_code} in %{time_total}s from %{remote_ip}\n' "$probe" && v6=1 || echo "ipv6: no answer"

if [ "$v4" = 1 ] && [ "$v6" = 0 ]; then
  echo "ipv6 is broken here; preferring ipv4"
  line='precedence ::ffff:0:0/96  100'
  if [ "$(id -u)" = 0 ]; then
    echo "$line" | tee -a /etc/gai.conf
  else
    echo "$line" | sudo tee -a /etc/gai.conf
  fi
fi
