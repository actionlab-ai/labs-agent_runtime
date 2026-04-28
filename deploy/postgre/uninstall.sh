#!/usr/bin/env bash
set -euo pipefail

NAMESPACE="${NAMESPACE:-aict}"
APP_NAME="${APP_NAME:-postgres}"
DELETE_PVC="${DELETE_PVC:-false}"
DELETE_NAMESPACE="${DELETE_NAMESPACE:-false}"

if ! command -v kubectl >/dev/null 2>&1; then
  echo "ERROR: kubectl not found" >&2
  exit 1
fi

echo "Deleting PostgreSQL workload from namespace=${NAMESPACE}, app=${APP_NAME}"

kubectl delete svc "${APP_NAME}-nodeport" -n "$NAMESPACE" --ignore-not-found=true
kubectl delete svc "${APP_NAME}" -n "$NAMESPACE" --ignore-not-found=true
kubectl delete svc "${APP_NAME}-headless" -n "$NAMESPACE" --ignore-not-found=true
kubectl delete sts "${APP_NAME}" -n "$NAMESPACE" --ignore-not-found=true
kubectl delete secret "${APP_NAME}-secret" -n "$NAMESPACE" --ignore-not-found=true

if [[ "$DELETE_PVC" == "true" ]]; then
  echo "DELETE_PVC=true, deleting PVC data-${APP_NAME}-0"
  kubectl delete pvc "data-${APP_NAME}-0" -n "$NAMESPACE" --ignore-not-found=true
else
  echo "PVC kept. To delete data too: export DELETE_PVC=true && ./uninstall.sh"
fi

if [[ "$DELETE_NAMESPACE" == "true" ]]; then
  echo "DELETE_NAMESPACE=true, deleting namespace ${NAMESPACE}"
  kubectl delete ns "$NAMESPACE" --ignore-not-found=true
fi
