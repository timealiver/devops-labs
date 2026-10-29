#!/usr/bin/env bash
PG=$(kubectl get pod shop-db-1 -n shop -o jsonpath='{.status.podIP}')
API=shop-api.shop.svc.cluster.local
WORKER=$(kubectl get svc shop-worker -n shop -o jsonpath='{.spec.clusterIP}')
check() {
  local ns=$1 pod=$2 host=$3 port=$4
  kubectl exec -n "$ns" "$pod" -- true >/dev/null 2>&1 || { echo no-pod; return; }
  kubectl exec -n "$ns" "$pod" -- nc -z -w 3 "$host" "$port" >/dev/null 2>&1 && echo open || echo blocked
}
resolve() {
  kubectl exec -n shop as-api -- true >/dev/null 2>&1 || { echo no-pod; return; }
  kubectl exec -n shop as-api -- nslookup -timeout=2 -retry=1 shop-api.shop.svc.cluster.local >/dev/null 2>&1 && echo open || echo blocked
}
row() {
  local name=$1 expected=$2 actual=$3 mark=OK
  [ "$expected" = "$actual" ] || mark=FAIL
  printf '%s\t%s\t%s\t%s\n' "$name" "$expected" "$actual" "$mark"
}
{
printf '%s\t%s\t%s\t%s\n' "источник -> цель" "ожидали" "факт" "итог"
row "клиент (снаружи, NodePort) -> api" open "$([ "$(docker run --rm --network kind registry.local/netcheck:1.0 curl -s -m 3 http://lab-worker:30080/health)" = ok ] && echo open || echo blocked)"
row "client/loadgen -> api:8080" open "$(check client "$(kubectl get pod -n client -o name | head -1 | cut -d/ -f2)" $API 8080)"
row "api -> postgres:5432" open "$(check shop as-api "$PG" 5432)"
row "worker -> postgres:5432" open "$(check shop as-worker "$PG" 5432)"
row "intruder (shop) -> postgres:5432" blocked "$(check shop intruder "$PG" 5432)"
row "outsider (другой ns) -> postgres" blocked "$(check outsider outsider "$PG" 5432)"
row "intruder (shop) -> api:8080" blocked "$(check shop intruder $API 8080)"
row "outsider (другой ns) -> api:8080" blocked "$(check outsider outsider $API 8080)"
row "api -> worker:8080" blocked "$(check shop as-api "$WORKER" 8080)"
row "api -> интернет 1.1.1.1:443" blocked "$(check shop as-api 1.1.1.1 443)"
row "api -> DNS (резолв имени)" open "$(resolve)"
} | column -t -s "$(printf '\t')"
