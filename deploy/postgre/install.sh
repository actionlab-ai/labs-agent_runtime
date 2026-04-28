#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEMPLATE_DIR="${ROOT_DIR}/templates"
OUT_DIR="${ROOT_DIR}/rendered"

NAMESPACE="${NAMESPACE:-codefly}"
APP_NAME="${APP_NAME:-postgres}"
IMAGE="${IMAGE:-swr.cn-north-4.myhuaweicloud.com/ddn-k8s/docker.io/postgres:16.13-alpine3.23}"
STORAGE_CLASS="${STORAGE_CLASS:-nfs}"
STORAGE_SIZE="${STORAGE_SIZE:-20Gi}"
ACCESS_MODE="${ACCESS_MODE:-ReadWriteOnce}"
POSTGRES_DB="${POSTGRES_DB:-codefly}"
POSTGRES_USER="${POSTGRES_USER:-postgres}"
: "${POSTGRES_PASSWORD:?POSTGRES_PASSWORD is required. Set it before running install.sh}"
TZ="${TZ:-Asia/Shanghai}"
CPU_REQUEST="${CPU_REQUEST:-100m}"
MEMORY_REQUEST="${MEMORY_REQUEST:-256Mi}"
CPU_LIMIT="${CPU_LIMIT:-2}"
MEMORY_LIMIT="${MEMORY_LIMIT:-2Gi}"
ENABLE_NODEPORT="${ENABLE_NODEPORT:-false}"
NODE_PORT="${NODE_PORT:-30432}"

if ! command -v kubectl >/dev/null 2>&1; then
  echo "ERROR: kubectl not found" >&2
  exit 1
fi

b64() {
  printf '%s' "$1" | base64 | tr -d '\n'
}

sed_escape() {
  printf '%s' "$1" | sed -e 's/[\/&]/\\&/g'
}

render() {
  local src="$1"
  local dst="$2"
  sed \
    -e "s/__NAMESPACE__/$(sed_escape "$NAMESPACE")/g" \
    -e "s/__APP_NAME__/$(sed_escape "$APP_NAME")/g" \
    -e "s#__IMAGE__#$(sed_escape "$IMAGE")#g" \
    -e "s/__STORAGE_CLASS__/$(sed_escape "$STORAGE_CLASS")/g" \
    -e "s/__STORAGE_SIZE__/$(sed_escape "$STORAGE_SIZE")/g" \
    -e "s/__ACCESS_MODE__/$(sed_escape "$ACCESS_MODE")/g" \
    -e "s/__POSTGRES_DB_B64__/$(sed_escape "$(b64 "$POSTGRES_DB")")/g" \
    -e "s/__POSTGRES_USER_B64__/$(sed_escape "$(b64 "$POSTGRES_USER")")/g" \
    -e "s/__POSTGRES_PASSWORD_B64__/$(sed_escape "$(b64 "$POSTGRES_PASSWORD")")/g" \
    -e "s/__TZ__/$(sed_escape "$TZ")/g" \
    -e "s/__CPU_REQUEST__/$(sed_escape "$CPU_REQUEST")/g" \
    -e "s/__MEMORY_REQUEST__/$(sed_escape "$MEMORY_REQUEST")/g" \
    -e "s/__CPU_LIMIT__/$(sed_escape "$CPU_LIMIT")/g" \
    -e "s/__MEMORY_LIMIT__/$(sed_escape "$MEMORY_LIMIT")/g" \
    -e "s/__NODE_PORT__/$(sed_escape "$NODE_PORT")/g" \
    "$src" > "$dst"

  if [[ -z "$STORAGE_CLASS" ]]; then
    sed -i '/storageClassName:/d' "$dst"
  fi
}

rm -rf "$OUT_DIR"
mkdir -p "$OUT_DIR"

render "$TEMPLATE_DIR/00-namespace.yaml.tpl" "$OUT_DIR/00-namespace.yaml"
render "$TEMPLATE_DIR/01-secret.yaml.tpl" "$OUT_DIR/01-secret.yaml"
render "$TEMPLATE_DIR/02-service.yaml.tpl" "$OUT_DIR/02-service.yaml"
render "$TEMPLATE_DIR/03-statefulset.yaml.tpl" "$OUT_DIR/03-statefulset.yaml"

if [[ "$ENABLE_NODEPORT" == "true" ]]; then
  render "$TEMPLATE_DIR/04-service-nodeport.yaml.tpl" "$OUT_DIR/04-service-nodeport.yaml"
fi

cat <<INFO
============================================================
PostgreSQL Kubernetes Deploy
============================================================
Namespace       : ${NAMESPACE}
App Name        : ${APP_NAME}
Image           : ${IMAGE}
StorageClass    : ${STORAGE_CLASS:-<cluster default>}
StorageSize     : ${STORAGE_SIZE}
AccessMode      : ${ACCESS_MODE}
Database        : ${POSTGRES_DB}
User            : ${POSTGRES_USER}
Port            : 5432
Cluster DNS     : ${APP_NAME}.${NAMESPACE}.svc.cluster.local:5432
NodePort        : ${ENABLE_NODEPORT} ${ENABLE_NODEPORT:+${NODE_PORT}}
Rendered Dir    : ${OUT_DIR}
============================================================
INFO

kubectl apply -f "$OUT_DIR"

kubectl rollout status "sts/${APP_NAME}" -n "$NAMESPACE" --timeout=300s

cat <<INFO

部署完成。常用命令：

  kubectl get pod,svc,pvc -n ${NAMESPACE} -l app.kubernetes.io/name=${APP_NAME}
  kubectl logs -n ${NAMESPACE} sts/${APP_NAME} -f
  kubectl get secret -n ${NAMESPACE} ${APP_NAME}-secret -o jsonpath='{.data.POSTGRES_PASSWORD}' | base64 -d; echo

集群内连接：

  host: ${APP_NAME}.${NAMESPACE}.svc.cluster.local
  port: 5432
  db  : ${POSTGRES_DB}
  user: ${POSTGRES_USER}

测试连接：

  kubectl run psql-client \\
    -n ${NAMESPACE} \\
    --rm -it \\
    --restart=Never \\
    --image=${IMAGE} \\
    --env PGPASSWORD='${POSTGRES_PASSWORD}' \\
    -- psql -h ${APP_NAME} -p 5432 -U ${POSTGRES_USER} -d ${POSTGRES_DB}
INFO
