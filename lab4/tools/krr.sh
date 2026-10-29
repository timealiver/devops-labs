#!/usr/bin/env bash
ns=${1:-shop}
kind get kubeconfig --name lab --internal > /tmp/kc-internal
docker run --rm --network kind -v /tmp/kc-internal:/kc:ro -e KUBECONFIG=/kc robustadev/krr:v1.30.0 \
  python krr.py simple -n "$ns" -p http://lab-control-plane:30090 \
  --history_duration 1 --timeframe_duration 1 --points_required 3 --width 112 2>&1 \
  | sed 's/\x1b\[[0-9;]*m//g' \
  | awk '/Simple Strategy/{p=1} /^All parameters/{p=0} /┏/{p=1} /points - D/{p=0} p' | grep -v '^ *$'
