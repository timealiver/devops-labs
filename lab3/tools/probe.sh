#!/usr/bin/env bash
dur=${1:-30}
url=${URL:-http://localhost:30080/version}
tmp=$(mktemp)
end=$((SECONDS+dur))
while [ $SECONDS -lt $end ]; do
  out=$(curl -s -m 2 -w ' %{http_code}' "$url")
  echo "${out##* } ${out% *}" >> "$tmp"
  sleep 0.1
done
awk '{ if ($1 == 200) { ok++; v[$2]++ } else bad++ } END { printf "ok=%d failed=%d\n", ok, bad; for (k in v) printf "version %s: %d\n", k, v[k] }' "$tmp" | sort
rm -f "$tmp"
