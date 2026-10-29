#!/usr/bin/env bash
base=${BASE:-http://localhost:18080}
dur=${1:-120}
mode=${2:-mix}
end=$((SECONDS+dur))
if [ "$mode" = alert ]; then
  for i in 1 2 3 4 5 6; do
    (while [ $SECONDS -lt $end ]; do curl -s -o /dev/null "$base/slow"; done) &
  done
  while [ $SECONDS -lt $end ]; do
    curl -s -o /dev/null "$base/fail"
    curl -s -o /dev/null "$base/health"
    sleep 0.2
  done
  wait
  exit 0
fi
while [ $SECONDS -lt $end ]; do
  curl -s -o /dev/null "$base/load?n=40"
  curl -s -o /dev/null "$base/fail"
  curl -s -o /dev/null "$base/fail"
  curl -s -o /dev/null "$base/slow" &
  sleep 1
done
wait
