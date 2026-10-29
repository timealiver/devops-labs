#!/usr/bin/env bash
cd "$(dirname "$0")/.."
options=(dns worker-label api-port no-client-ingress)
pick=${options[RANDOM % ${#options[@]}]}
helm upgrade shop charts/shop -n shop --reuse-values --set networkPolicy.breakage="$pick" >/dev/null
kubectl delete pod -n shop -l 'app.kubernetes.io/name in (api,worker),test!=true' --wait=false >/dev/null
echo "breakage applied"
