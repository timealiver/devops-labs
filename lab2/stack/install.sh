#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo add grafana https://grafana.github.io/helm-charts
helm repo add wiremind https://wiremind.github.io/wiremind-helm-charts
helm repo update
kubectl create namespace monitoring --dry-run=client -o yaml | kubectl apply -f -
helm upgrade --install kps prometheus-community/kube-prometheus-stack -n monitoring -f kps-values.yaml --wait --timeout 8m
helm upgrade --install loki grafana/loki -n monitoring -f loki-values.yaml --wait --timeout 8m
helm upgrade --install alloy grafana/alloy -n monitoring -f alloy-values.yaml --wait --timeout 5m
helm upgrade --install jaeger ../charts/jaeger -n monitoring --wait
helm upgrade --install webhook-sink ../charts/webhook-sink -n monitoring --wait
helm upgrade --install karma wiremind/karma -n monitoring -f karma-values.yaml --wait --timeout 5m
