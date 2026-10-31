#!/usr/bin/env bash
# Aoe Tunnel Agent 一键安装脚本。
# 两种用法：
#   1) 面板节点详情页渲染的脚本：顶部已注入 AOE_* 变量，直接 sudo bash 执行；
#   2) 云存储托管：curl -fsSL <脚本地址> | sudo bash -s -- --code <节点编码> --token <令牌> --panel <面板地址> --base <下载地址>
set -euo pipefail

INSTALL_DIR="${AOE_INSTALL_DIR:-/opt/aoe-agent}"
SERVICE_NAME="aoe-agent"
ROLE="${AOE_ROLE:-agent}"
NODE_CODE="${AOE_NODE_CODE:-}"
NODE_TOKEN="${AOE_NODE_TOKEN:-}"
PANEL_API_URL="${AOE_PANEL_URL:-}"
MQTT_BROKER="${AOE_MQTT_BROKER:-}"
MQTT_SNI="${AOE_MQTT_SNI:-}"
MQTT_USER="${AOE_MQTT_USER:-}"
MQTT_PASS="${AOE_MQTT_PASS:-}"
PANEL_USER="${AOE_PANEL_USER:-agent}"
PANEL_PASS="${AOE_PANEL_PASS:-agent123}"
DOWNLOAD_BASE="${AOE_DOWNLOAD_BASE:-}"
QUIC_ADDR="${AOE_QUIC_ADDR:-:9613}"
TLS_ADDR="${AOE_TLS_ADDR:-:9614}"
TLS_SNI="${AOE_SNI:-aoe-node}"
DRY_RUN="${AOE_DRY_RUN:-0}"

usage() {
  cat <<'EOF'
用法:
  sudo bash install.sh [选项]

选项:
  --code <编码>       节点编码（必填）
  --token <令牌>      节点令牌 aoe_nt_xxx（必填，仅用于上线校验）
  --panel <地址>      面板地址，如 https://panel.example.com（必填）
  --broker <地址>     MQTT 接入地址（必填），如 ssl://panel.example.com:8883
  --base <地址>       二进制与证书下载目录，默认主控面板 ${PANEL_API_URL}/downloads
  --mqtt-user <账号>  MQTT 账号（EMQX 启用认证时必填）
  --mqtt-pass <密码>  MQTT 密码
  --panel-user <账号> 面板 Agent 登录账号，默认 agent
  --panel-pass <密码> 面板 Agent 登录密码
  --quic-addr <地址>  QUIC 数据面监听地址，默认 :9613
  --tls-addr <地址>   TLS 数据面监听地址，默认 :9614
  --dir <目录>        安装目录，默认 /opt/aoe-agent
  --sni <域名>        数据面 TLS SNI，默认 aoe-node
  --dry-run           只打印将要执行的动作，不落盘
  -h, --help          显示本帮助
EOF
}

die() {
  echo "[aoe-agent] 错误: $*" >&2
  exit 1
}

log() {
  echo "[aoe-agent] $*"
}

run() {
  if [[ "${DRY_RUN}" == "1" ]]; then
    log "[dry-run] $*"
  else
    "$@"
  fi
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --code) NODE_CODE="${2:-}"; shift 2 ;;
    --token) NODE_TOKEN="${2:-}"; shift 2 ;;
    --panel) PANEL_API_URL="${2:-}"; shift 2 ;;
    --broker) MQTT_BROKER="${2:-}"; shift 2 ;;
    --base) DOWNLOAD_BASE="${2:-}"; shift 2 ;;
    --mqtt-user) MQTT_USER="${2:-}"; shift 2 ;;
    --mqtt-pass) MQTT_PASS="${2:-}"; shift 2 ;;
    --panel-user) PANEL_USER="${2:-}"; shift 2 ;;
    --panel-pass) PANEL_PASS="${2:-}"; shift 2 ;;
    --quic-addr) QUIC_ADDR="${2:-}"; shift 2 ;;
    --tls-addr) TLS_ADDR="${2:-}"; shift 2 ;;
    --dir) INSTALL_DIR="${2:-}"; shift 2 ;;
    --sni) TLS_SNI="${2:-}"; shift 2 ;;
    --mqtt-sni) MQTT_SNI="${2:-}"; shift 2 ;;
    --dry-run) DRY_RUN=1; shift ;;
    -h|--help) usage; exit 0 ;;
    *) die "未知参数: $1（用 --help 查看用法）" ;;
  esac
done

[[ -n "${NODE_CODE}" ]] || die "缺少节点编码（--code）"
[[ -n "${NODE_TOKEN}" ]] || die "缺少节点令牌（--token）"
[[ -n "${PANEL_API_URL}" ]] || die "缺少面板地址（--panel）"
[[ -n "${MQTT_BROKER}" ]] || die "缺少 MQTT 地址（--broker）"
# 参数解析后再补默认值：二进制与证书统一从主控面板 /downloads 拉取。
if [[ -z "${DOWNLOAD_BASE}" ]]; then
  DOWNLOAD_BASE="${PANEL_API_URL%/}/downloads"
fi

if [[ "${DRY_RUN}" != "1" && "${EUID}" -ne 0 ]]; then
  die "需要 root 权限，请使用 sudo 执行"
fi

# 按内核架构选择二进制；与发布包命名保持一致 aoe-agent-linux-<arch>。
case "$(uname -m)" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  armv7l|armv7) ARCH="armv7" ;;
  i386|i686) ARCH="386" ;;
  *) die "不支持的系统架构: $(uname -m)" ;;
esac
log "系统架构: $(uname -m) -> ${ARCH}，节点: ${NODE_CODE}（${ROLE}）"

BIN_DIR="${INSTALL_DIR}/bin"
DATA_DIR="${INSTALL_DIR}/data"
CERT_DIR="${INSTALL_DIR}/certs"
ENV_FILE="${INSTALL_DIR}/agent.env"
SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}.service"
TRIM_BASE="${DOWNLOAD_BASE%/}"
BIN_URL="${TRIM_BASE}/aoe-agent-linux-${ARCH}"
CERT_URL="${TRIM_BASE}/aoe-agent-certs.tar.gz"

run mkdir -p "${BIN_DIR}" "${DATA_DIR}" "${CERT_DIR}"

if command -v systemctl >/dev/null 2>&1; then
  if systemctl is-active --quiet "${SERVICE_NAME}" 2>/dev/null; then
    log "停止已有服务，准备替换二进制"
    run systemctl stop "${SERVICE_NAME}"
  fi
fi

BIN_PATH="${BIN_DIR}/aoe-agent"
if [[ -n "${TRIM_BASE}" ]]; then
  log "下载 Agent 二进制: ${BIN_URL}"
  run curl -fsSL --retry 3 --connect-timeout 10 "${BIN_URL}" -o "${BIN_PATH}.tmp"
  run mv -f "${BIN_PATH}.tmp" "${BIN_PATH}"
  run chmod 755 "${BIN_PATH}"
elif [[ ! -x "${BIN_PATH}" ]]; then
  die "缺少下载地址（--base 或 --panel），且 ${BIN_PATH} 不存在"
fi

if [[ -n "${TRIM_BASE}" && "${DRY_RUN}" != "1" ]]; then
  TMP_CERT="$(mktemp -d)"
  if curl -fsSL --retry 3 --connect-timeout 10 "${CERT_URL}" -o "${TMP_CERT}/certs.tar.gz" 2>/dev/null; then
    log "解压数据面与控制面证书"
    run tar -xzf "${TMP_CERT}/certs.tar.gz" -C "${CERT_DIR}"
    run chmod 600 "${CERT_DIR}"/*.key 2>/dev/null || true
  else
    log "警告: 证书包下载失败（${CERT_URL}），将使用已存在的证书文件"
  fi
  run rm -rf "${TMP_CERT}"
fi

for f in ca.crt server.crt server.key client.crt client.key; do
  if [[ "${DRY_RUN}" != "1" && ! -f "${CERT_DIR}/${f}" ]]; then
    die "缺少证书 ${CERT_DIR}/${f}，请把证书包放到 ${TRIM_BASE:-下载目录}"
  fi
done

log "写入配置 ${ENV_FILE}"
if [[ "${DRY_RUN}" == "1" ]]; then
  log "[dry-run] 生成 ${ENV_FILE}（node=${NODE_CODE}, panel=${PANEL_API_URL}）"
else
  umask 077
  cat > "${ENV_FILE}" <<EOF
AGENT_NODE_CODE=${NODE_CODE}
AGENT_NODE_TOKEN=${NODE_TOKEN}
PANEL_API_URL=${PANEL_API_URL}
PANEL_AGENT_USER=${PANEL_USER}
PANEL_AGENT_PASS=${PANEL_PASS}
MQTT_BROKER=${MQTT_BROKER}
MQTT_USER=${MQTT_USER}
MQTT_PASS=${MQTT_PASS}
MQTT_SNI=${MQTT_SNI}
MQTT_CA=${CERT_DIR}/emqx-ca.pem
AGENT_QUIC_ADDR=${QUIC_ADDR}
AGENT_TLS_ADDR=${TLS_ADDR}
AGENT_SNI=${TLS_SNI}
AGENT_DATA_DIR=${DATA_DIR}
AGENT_CERT_FILE=${CERT_DIR}/server.crt
AGENT_KEY_FILE=${CERT_DIR}/server.key
AGENT_CLIENT_CERT_FILE=${CERT_DIR}/client.crt
AGENT_CLIENT_KEY_FILE=${CERT_DIR}/client.key
AGENT_CA_FILE=${CERT_DIR}/ca.crt
LOG_LEVEL=info
LOG_FORMAT=json
EOF
fi

if ! command -v systemctl >/dev/null 2>&1; then
  die "未检测到 systemd；可手动运行: set -a && . ${ENV_FILE} && set +a && ${BIN_PATH}"
fi

log "注册 systemd 服务 ${SERVICE_NAME}"
if [[ "${DRY_RUN}" == "1" ]]; then
  log "[dry-run] 写入 ${SERVICE_FILE} 并 systemctl enable --now ${SERVICE_NAME}"
else
  cat > "${SERVICE_FILE}" <<EOF
[Unit]
Description=Aoe Tunnel Agent (${NODE_CODE}/${ROLE})
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=${INSTALL_DIR}
EnvironmentFile=${ENV_FILE}
ExecStart=${BIN_PATH}
Restart=always
RestartSec=5
LimitNOFILE=1048576

[Install]
WantedBy=multi-user.target
EOF
  systemctl daemon-reload
  systemctl enable "${SERVICE_NAME}" >/dev/null
  systemctl restart "${SERVICE_NAME}"
  sleep 2
  if ! systemctl is-active --quiet "${SERVICE_NAME}"; then
    echo "[aoe-agent] 服务启动失败，最近日志：" >&2
    journalctl -u "${SERVICE_NAME}" -n 30 --no-pager >&2 || true
    exit 1
  fi
fi

log "安装完成: ${NODE_CODE}（${ROLE}）"
log "服务状态: systemctl status ${SERVICE_NAME}"
log "实时日志: journalctl -u ${SERVICE_NAME} -f"
