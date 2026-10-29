#!/usr/bin/env bash
ns=${NS:-shop}
interval=${INTERVAL:-3}
dur=${1:-120}
end=$((SECONDS+dur))
while [ $SECONDS -lt $end ]; do
  pod=$(kubectl get pods -n "$ns" -l 'app.kubernetes.io/name in (api,worker),test!=true' -o name | sort -R | head -1)
  echo "$(date +%s) $pod"
  kubectl delete -n "$ns" "$pod" --wait=false >/dev/null
  sleep "$interval"
done
