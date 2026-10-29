#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
helm repo add kyverno https://kyverno.github.io/kyverno/
helm repo add cnpg https://cloudnative-pg.github.io/charts
helm repo update
helm upgrade --install kyverno kyverno/kyverno -n kyverno --create-namespace -f kyverno-values.yaml --wait --timeout 8m
helm upgrade --install cnpg cnpg/cloudnative-pg -n cnpg-system --create-namespace --wait --timeout 5m
