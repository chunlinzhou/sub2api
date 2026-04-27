#!/usr/bin/env bash

set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP_DIR="$(mktemp -d)"

REMOTE_HOST="root@47.85.189.37"
REMOTE_APP_DIR="/root/sub2api-test-8081"
REMOTE_DEPLOY_DIR="${REMOTE_APP_DIR}/deploy"
REMOTE_ENV_FILE="${REMOTE_DEPLOY_DIR}/.env"
REMOTE_TMP_DIR="/tmp/sub2api-test-8081-deploy"

COMPOSE_BASE="docker-compose.yml"
COMPOSE_OVERRIDE="docker-compose.test-8081.yml"
SERVICE_NAME="sub2api"
CONTAINER_NAME="sub2api-test-8081"
IMAGE_TAG="sub2api:test-8081"
BACKUP_TAG="sub2api:test-8081-backup"

HEALTH_TIMEOUT=180
HEALTH_INTERVAL=3

SSH_OPTS=(
  -o BatchMode=yes
  -o StrictHostKeyChecking=accept-new
  -o ConnectTimeout=10
)

ARCHIVE_LOCAL="${TMP_DIR}/sub2api-test-8081-src.tar.gz"
ARCHIVE_REMOTE="${REMOTE_TMP_DIR}/sub2api-test-8081-src.tar.gz"

cleanup() {
  rm -rf "${TMP_DIR}"
}
trap cleanup EXIT

log() {
  printf '[deploy] %s\n' "$*"
}

die() {
  printf '[deploy] ERROR: %s\n' "$*" >&2
  exit 1
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "缺少命令: $1"
}

remote() {
  ssh "${SSH_OPTS[@]}" "${REMOTE_HOST}" "$@"
}

upload_source() {
  log "本地打包源码"
  tar \
    --exclude=".git" \
    --exclude="frontend/node_modules" \
    --exclude="frontend/dist" \
    --exclude="backend/.cache" \
    --exclude="backend/tmp" \
    --exclude=".DS_Store" \
    --exclude="deploy/.env" \
    --exclude="deploy/${COMPOSE_OVERRIDE}" \
    -czf "${ARCHIVE_LOCAL}" \
    -C "${ROOT_DIR}" .

  log "上传源码包到远端"
  remote "mkdir -p '${REMOTE_TMP_DIR}'"
  scp "${SSH_OPTS[@]}" "${ARCHIVE_LOCAL}" "${REMOTE_HOST}:${ARCHIVE_REMOTE}" >/dev/null

  log "远端替换代码目录并保留环境文件"
  remote "
    set -e
    mkdir -p '${REMOTE_TMP_DIR}'
    cp '${REMOTE_ENV_FILE}' '${REMOTE_TMP_DIR}/.env'
    cp '${REMOTE_DEPLOY_DIR}/${COMPOSE_OVERRIDE}' '${REMOTE_TMP_DIR}/${COMPOSE_OVERRIDE}'
    rm -rf '${REMOTE_APP_DIR}'
    mkdir -p '${REMOTE_APP_DIR}'
    tar -xzf '${ARCHIVE_REMOTE}' -C '${REMOTE_APP_DIR}'
    mkdir -p '${REMOTE_DEPLOY_DIR}'
    mv '${REMOTE_TMP_DIR}/.env' '${REMOTE_ENV_FILE}'
    mv '${REMOTE_TMP_DIR}/${COMPOSE_OVERRIDE}' '${REMOTE_DEPLOY_DIR}/${COMPOSE_OVERRIDE}'
  "
}

compose_cmd() {
  local project_name="$1"
  cat <<EOF
cd '${REMOTE_DEPLOY_DIR}' && docker compose -p '${project_name}' --env-file '${REMOTE_ENV_FILE}' -f '${COMPOSE_BASE}' -f '${COMPOSE_OVERRIDE}'
EOF
}

wait_for_healthy() {
  local elapsed=0
  local status=""

  while (( elapsed < HEALTH_TIMEOUT )); do
    status="$(remote "docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' '${CONTAINER_NAME}'" 2>/dev/null || true)"
    case "${status}" in
      healthy|running)
        printf '%s' "${status}"
        return 0
        ;;
      unhealthy|exited|dead)
        printf '%s' "${status}"
        return 1
        ;;
    esac
    sleep "${HEALTH_INTERVAL}"
    elapsed=$((elapsed + HEALTH_INTERVAL))
  done

  printf '%s' "${status}"
  return 1
}

rollback() {
  log "新版本健康检查失败，开始回滚"
  remote "docker image inspect '${BACKUP_TAG}' >/dev/null 2>&1" || die "缺少回滚镜像 ${BACKUP_TAG}"
  remote "docker tag '${BACKUP_TAG}' '${IMAGE_TAG}'"
  remote "$(compose_cmd "${PROJECT_NAME}") up -d --no-deps '${SERVICE_NAME}' >/dev/null"

  local rollback_status
  rollback_status="$(wait_for_healthy || true)"
  [[ "${rollback_status}" == "healthy" || "${rollback_status}" == "running" ]] || die "回滚失败，当前状态: ${rollback_status}"

  die "已回滚到旧版本"
}

require_cmd ssh
require_cmd scp
require_cmd tar

log "检查远端部署环境"
remote "test -d '${REMOTE_APP_DIR}'" || die "远端缺少部署目录 ${REMOTE_APP_DIR}"
remote "test -f '${REMOTE_ENV_FILE}'" || die "远端缺少环境文件 ${REMOTE_ENV_FILE}"
remote "test -f '${REMOTE_DEPLOY_DIR}/${COMPOSE_OVERRIDE}'" || die "远端缺少覆盖文件 ${REMOTE_DEPLOY_DIR}/${COMPOSE_OVERRIDE}"
remote "docker compose version >/dev/null"

PROJECT_NAME="$(remote "docker inspect --format '{{ index .Config.Labels \"com.docker.compose.project\" }}' '${CONTAINER_NAME}' 2>/dev/null || true")"
[[ -n "${PROJECT_NAME}" ]] || die "无法从当前容器 ${CONTAINER_NAME} 识别 compose project 名"
log "使用现有 compose project: ${PROJECT_NAME}"

upload_source

log "校验远端 compose 配置"
remote "$(compose_cmd "${PROJECT_NAME}") config >/dev/null"

log "备份当前运行镜像"
remote "current_image_id=\$(docker inspect --format '{{.Image}}' '${CONTAINER_NAME}' 2>/dev/null || true); if [ -n \"\$current_image_id\" ]; then docker tag \"\$current_image_id\" '${BACKUP_TAG}'; fi"

log "在远端构建镜像"
remote "$(compose_cmd "${PROJECT_NAME}") build '${SERVICE_NAME}'"

log "在远端更新容器"
remote "$(compose_cmd "${PROJECT_NAME}") up -d --no-deps '${SERVICE_NAME}'"

log "等待健康检查"
status="$(wait_for_healthy || true)"
if [[ "${status}" != "healthy" && "${status}" != "running" ]]; then
  rollback
fi

log "输出当前容器状态"
remote "docker ps --filter 'name=^/${CONTAINER_NAME}\$' --format 'table {{.Names}}\t{{.Image}}\t{{.Status}}'"

log "清理远端临时文件"
remote "rm -rf '${REMOTE_TMP_DIR}'"

log "部署完成"
log "已同步本地源码到远端并完成构建部署"
