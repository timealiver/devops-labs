#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
kubectl create namespace monitoring --dry-run=client -o yaml | kubectl apply -f -
helm upgrade --install kps prometheus-community/kube-prometheus-stack -n monitoring -f kps-values.yaml --wait --timeout 8m
helm upgrade --install webhook-sink ../lab2/charts/webhook-sink -n monitoring --wait
helm upgrade --install kyverno kyverno/kyverno -n kyverno --create-namespace -f ../lab3/platform/kyverno-values.yaml --wait --timeout 8m
helm upgrade --install cnpg cnpg/cloudnative-pg -n cnpg-system --create-namespace --wait --timeout 5m
kubectl apply -f ../lab3/policies/
