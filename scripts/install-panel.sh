#!/usr/bin/env bash
# Aoe-tunnel 主控面板一键安装脚本（Debian 12 / Ubuntu 22.04，root 执行）
# 仅部署主控面板；中转 Agent 由面板节点页面生成命令并独立部署到其他服务器。
set -Eeuo pipefail

ROOT="/opt/aoe-panel"
ENV_DIR="/etc/aoe-panel"
ENV_FILE="$ENV_DIR/panel.env"
CRED_FILE="$ENV_DIR/credentials.txt"
SERVICE_FILE="/etc/systemd/system/aoe-panel.service"
NGINX_SITE="/etc/nginx/conf.d/aoe-panel.conf"
SSL_DIR="/etc/nginx/ssl"
ACME_HOME="/root/.acme.sh"
RELEASE_BASE="${PANEL_RELEASE_BASE:-https://tunzzzz7.oss-cn-hongkong.aliyuncs.com}"

DOMAIN=""
ADMIN_USER="admin"
RELEASE_DIR=""
SKIP_DNS_CHECK="false"
DRY_RUN="false"

log()  { printf '\033[1;36m[aoe-panel]\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m[aoe-panel][warn]\033[0m %s\n' "$*"; }
die()  { printf '\033[1;31m[aoe-panel][error]\033[0m %s\n' "$*" >&2; exit 1; }

usage() {
  cat <<'EOF'
用法：install-panel.sh [选项]
  --domain <域名>         面板域名（交互式安装时可省略，脚本会提示输入）
  --admin-user <账号>     管理员账号，默认 admin，密码自动生成
  --release-base <URL>    预编译发布包地址前缀，默认 PANEL_RELEASE_BASE 或内置云存储
  --release-dir <目录>    从本地目录读取发布包（离线安装 / 测试）
  --skip-dns-check        跳过 DNS 解析校验
  --root <目录>           安装目录，默认 /opt/aoe-panel
  --dry-run               只生成配置文件，不安装依赖、不启动容器/服务（用于自检）
  -h, --help              显示帮助
EOF
}

while [ $# -gt 0 ]; do
  case "$1" in
    --domain) DOMAIN="${2:-}"; shift 2 ;;
    --admin-user) ADMIN_USER="${2:-}"; shift 2 ;;
    --release-base) RELEASE_BASE="${2:-}"; shift 2 ;;
    --release-dir) RELEASE_DIR="${2:-}"; shift 2 ;;
    --skip-dns-check) SKIP_DNS_CHECK="true"; shift ;;
    --root) ROOT="${2:-}"; shift 2 ;;
    --dry-run) DRY_RUN="true"; shift ;;
    -h|--help) usage; exit 0 ;;
    *) die "未知参数：$1（使用 --help 查看用法）" ;;
  esac
done

ENV_FILE="$ROOT/panel.env"
CRED_FILE="$ROOT/credentials.txt"
SERVICE_FILE="$ROOT/aoe-panel.service"
NGINX_SITE="$ROOT/aoe-panel.nginx.conf"
[ "$DRY_RUN" = "true" ] || { ENV_FILE="$ENV_DIR/panel.env"; CRED_FILE="$ENV_DIR/credentials.txt"; SERVICE_FILE="/etc/systemd/system/aoe-panel.service"; NGINX_SITE="/etc/nginx/conf.d/aoe-panel.conf"; }

run() {
  if [ "$DRY_RUN" = "true" ]; then
    printf '[dry-run] %s\n' "$*"
  else
    "$@"
  fi
}

need_root() {
  [ "$DRY_RUN" = "true" ] && return 0
  [ "$(id -u)" = "0" ] || die "必须以 root 执行"
}

check_os() {
  [ -r /etc/os-release ] || die "无法识别操作系统"
  # shellcheck disable=SC1091
  . /etc/os-release
  case "${ID:-}:${VERSION_ID:-}" in
    debian:12|ubuntu:22.04) log "系统检查通过：${PRETTY_NAME}" ;;
    *) die "仅支持 Debian 12 / Ubuntu 22.04，当前：${PRETTY_NAME:-unknown}" ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) die "不支持的 CPU 架构：$(uname -m)，仅提供 amd64 / arm64" ;;
  esac
  log "CPU 架构：$(uname -m) -> $ARCH"
}

rand_hex() { openssl rand -hex "${1:-12}"; }

install_deps() {
  export DEBIAN_FRONTEND=noninteractive
  log "安装基础依赖：nginx / socat / openssl / cron"
  run apt-get update -qq
  run apt-get install -y -qq ca-certificates curl openssl tar gzip socat cron nginx

  if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
    log "docker 与 compose 插件已存在，跳过安装"
  else
    log "安装 Docker Engine 与 compose 插件"
    if [ "$DRY_RUN" = "true" ]; then
      printf '[dry-run] curl -fsSL https://get.docker.com | sh\n'
    else
      curl -fsSL https://get.docker.com | sh
    fi
  fi
  if [ "$DRY_RUN" != "true" ]; then
    systemctl enable --now docker
    if docker compose version >/dev/null 2>&1; then
      COMPOSE=(docker compose)
    elif command -v docker-compose >/dev/null 2>&1; then
      COMPOSE=(docker-compose)
    else
      die "docker compose 安装失败"
    fi
  else
    COMPOSE=(docker compose)
  fi
}

prompt_domain() {
  if [ -n "$DOMAIN" ]; then
    return
  fi
  [ -t 0 ] || die "非交互环境需通过 --domain 指定面板域名"
  echo
  read -r -p "请输入面板域名（例如 panel.example.com）：" DOMAIN
  [ -n "$DOMAIN" ] || die "域名不能为空"
}

validate_domain() {
  [[ "$DOMAIN" =~ ^[a-zA-Z0-9.-]+$ ]] || die "域名格式不正确：$DOMAIN"
  case "$DOMAIN" in
    *.com|*.cn|*.net|*.org|*.vip|*.io|*.xyz|*.top|*.cloud|*.site|*.me) ;;
    *.) die "域名不能以点结尾" ;;
  esac
  log "面板域名：$DOMAIN"
}

dns_check() {
  SERVER_IP="$(curl -fsS --max-time 6 https://api.ipify.org 2>/dev/null || hostname -I 2>/dev/null | awk '{print $1}')"
  SERVER_IP="${SERVER_IP:-未知}"
  echo
  log "请在 DNS 服务商添加如下解析记录："
  printf '  记录类型 : A\n  主机记录 : %s\n  记录值   : %s\n  TTL      : 600\n\n' "$DOMAIN" "$SERVER_IP"

  if [ "$SKIP_DNS_CHECK" = "true" ]; then
    warn "已跳过 DNS 解析校验"
    return
  fi

  local resolved=""
  for i in 1 2 3; do
    resolved="$(getent ahostsv4 "$DOMAIN" 2>/dev/null | awk '{print $1}' | sort -u | paste -sd ',' -)"
    [ -n "$resolved" ] && break
    warn "第 $i 次解析未生效，5 秒后重试"
    sleep 5
  done
  [ -n "$resolved" ] || die "域名 $DOMAIN 无法解析，请确认 A 记录已生效后重试（或加 --skip-dns-check）"
  log "当前解析结果：$resolved"

  if [ "$SERVER_IP" != "未知" ] && ! printf '%s\n' "$resolved" | tr ',' '\n' | grep -qx "$SERVER_IP"; then
    warn "解析 IP 与本机公网 IP（$SERVER_IP）不一致，证书签发可能失败"
    if [ -t 0 ]; then
      read -r -p "是否继续？[y/N] " ans
      [ "$ans" = "y" ] || [ "$ans" = "Y" ] || die "已取消"
    else
      warn "非交互环境，继续执行"
    fi
  fi
}

install_acme() {
  if [ -x "$ACME_HOME/acme.sh" ]; then
    log "acme.sh 已安装，跳过"
    return
  fi
  log "安装 acme.sh"
  if [ "$DRY_RUN" = "true" ]; then
    printf '[dry-run] curl -fsSL https://get.acme.sh | sh -s email=admin@%s\n' "$DOMAIN"
  else
    curl -fsSL https://get.acme.sh | sh -s "email=admin@$DOMAIN"
  fi
}

issue_cert() {
  log "使用 acme.sh standalone 签发证书（临时停止 nginx 释放 80 端口）"
  run systemctl stop nginx || true
  if [ "$DRY_RUN" = "true" ]; then
    printf '[dry-run] %s --issue -d %s --standalone --httpport 80\n' "$ACME_HOME/acme.sh" "$DOMAIN"
  else
    "$ACME_HOME/acme.sh" --register-account -m "admin@$DOMAIN" >/dev/null 2>&1 || true
    "$ACME_HOME/acme.sh" --issue -d "$DOMAIN" --standalone --httpport 80
    run mkdir -p "$SSL_DIR"
    "$ACME_HOME/acme.sh" --install-cert -d "$DOMAIN" \
      --key-file "$SSL_DIR/$DOMAIN.key" \
      --fullchain-file "$SSL_DIR/$DOMAIN.pem" \
      --reloadcmd "systemctl reload nginx || true"
  fi
}

gen_mqtt_certs() {
  local dir="$ROOT/certs"
  mkdir -p "$dir"
  if [ "$DRY_RUN" = "true" ]; then
    printf '[dry-run] openssl 生成 MQTT CA 与服务器证书到 %s\n' "$dir"
    return
  fi
  openssl genrsa -out "$dir/mqtt-ca.key" 4096 >/dev/null 2>&1
  openssl req -x509 -new -nodes -key "$dir/mqtt-ca.key" -sha256 -days 3650 \
    -subj "/CN=Aoe-Panel MQTT CA" -out "$dir/mqtt-ca.crt" >/dev/null 2>&1
  openssl genrsa -out "$dir/mqtt-server.key" 2048 >/dev/null 2>&1
  openssl req -new -key "$dir/mqtt-server.key" -subj "/CN=$DOMAIN" -out "$dir/mqtt-server.csr" >/dev/null 2>&1
  cat > "$dir/san.cnf" <<EOF
subjectAltName=DNS:$DOMAIN,IP:127.0.0.1
EOF
  openssl x509 -req -in "$dir/mqtt-server.csr" -CA "$dir/mqtt-ca.crt" -CAkey "$dir/mqtt-ca.key" \
    -CAcreateserial -days 3650 -sha256 -extfile "$dir/san.cnf" -out "$dir/mqtt-server.crt" >/dev/null 2>&1
  rm -f "$dir/mqtt-server.csr" "$dir/san.cnf" "$dir/mqtt-ca.srl"
  chmod 600 "$dir"/*.key
}

write_compose() {
  log "生成 docker-compose.yml（MySQL 8.0 + EMQX 5.8，MQTT TLS 对公网开放、其余仅内部）"
  cat > "$ROOT/docker-compose.yml" <<EOF
name: aoe-panel
services:
  mysql:
    image: mysql:8.0
    restart: unless-stopped
    command: ["--character-set-server=utf8mb4", "--collation-server=utf8mb4_unicode_ci"]
    environment:
      MYSQL_ROOT_PASSWORD: "$MYSQL_ROOT_PASSWORD"
      MYSQL_DATABASE: "aoe_tunnel"
      MYSQL_USER: "aoe"
      MYSQL_PASSWORD: "$MYSQL_PASSWORD"
    ports:
      - "127.0.0.1:3306:3306"
    volumes:
      - mysql-data:/var/lib/mysql
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "127.0.0.1", "-uroot", "-p$MYSQL_ROOT_PASSWORD"]
      interval: 5s
      timeout: 5s
      retries: 30

  emqx:
    image: emqx/emqx:5.8
    restart: unless-stopped
    environment:
      EMQX_NODE__COOKIE: "$EMQX_COOKIE"
      EMQX_DASHBOARD__DEFAULT_PASSWORD: "$EMQX_DASH_PASS"
      EMQX_LISTENERS__SSL__DEFAULT__BIND: "0.0.0.0:8883"
      EMQX_LISTENERS__SSL__DEFAULT__SSL_OPTIONS__CERTFILE: "/certs/mqtt-server.crt"
      EMQX_LISTENERS__SSL__DEFAULT__SSL_OPTIONS__KEYFILE: "/certs/mqtt-server.key"
      EMQX_LISTENERS__SSL__DEFAULT__SSL_OPTIONS__CACERTFILE: "/certs/mqtt-ca.crt"
      EMQX_LISTENERS__SSL__DEFAULT__SSL_OPTIONS__VERIFY: "verify_none"
    volumes:
      - ./certs:/certs:ro
      - emqx-data:/opt/emqx/data
    ports:
      - "127.0.0.1:1883:1883"
      # 远端 Agent 需通过 MQTT over TLS 接收实时规则，故 8883 对外监听
      - "0.0.0.0:8883:8883"
      - "127.0.0.1:18083:18083"
    healthcheck:
      test: ["CMD", "emqx", "ctl", "status"]
      interval: 10s
      timeout: 5s
      retries: 30

volumes:
  mysql-data:
  emqx-data:
EOF
}

start_infra() {
  log "启动 MySQL 与 EMQX 容器"
  run "${COMPOSE[@]}" -f "$ROOT/docker-compose.yml" pull
  run "${COMPOSE[@]}" -f "$ROOT/docker-compose.yml" up -d
  if [ "$DRY_RUN" = "true" ]; then
    return
  fi
  log "等待 MySQL 就绪"
  local i
  for i in $(seq 1 60); do
    if "${COMPOSE[@]}" -f "$ROOT/docker-compose.yml" exec -T mysql \
        mysqladmin ping -h127.0.0.1 -uroot -p"$MYSQL_ROOT_PASSWORD" --silent >/dev/null 2>&1; then
      log "MySQL 已就绪"
      return
    fi
    sleep 2
  done
  die "MySQL 120 秒内未就绪，查看：docker compose -f $ROOT/docker-compose.yml logs mysql"
}

fetch() {
  local name="$1" dest="$2"
  if [ -n "$RELEASE_DIR" ]; then
    [ -f "$RELEASE_DIR/$name" ] || die "发布包缺失：$RELEASE_DIR/$name"
    cp "$RELEASE_DIR/$name" "$dest"
    return
  fi
  curl -fsSL --retry 3 --connect-timeout 10 "$RELEASE_BASE/$name" -o "$dest" \
    || die "下载失败：$RELEASE_BASE/$name，可改用 --release-dir 指定本地发布包"
}

# 可选物料：缺失时返回 1，由调用方告警，不中止主控部署。
fetch_optional() {
  local name="$1" dest="$2"
  if [ -n "$RELEASE_DIR" ]; then
    [ -f "$RELEASE_DIR/$name" ] || return 1
    cp "$RELEASE_DIR/$name" "$dest"
    return 0
  fi
  curl -fsSL --retry 2 --connect-timeout 10 "$RELEASE_BASE/$name" -o "$dest" 2>/dev/null
}

install_release() {
  log "拉取预编译发布包：aoe-panel + 前端 dist"
  mkdir -p "$ROOT/bin" "$ROOT/web" "$ROOT/agent-dist"
  local tmp="$ROOT/.download"
  mkdir -p "$tmp"

  fetch "aoe-panel-linux-$ARCH" "$ROOT/bin/aoe-panel"
  run chmod +x "$ROOT/bin/aoe-panel"

  fetch "aoe-panel-web.tar.gz" "$tmp/web.tar.gz"
  if [ "$DRY_RUN" = "true" ]; then
    printf '[dry-run] tar -xzf %s -C %s 并移动 dist -> %s\n' "$tmp/web.tar.gz" "$ROOT" "$ROOT/web"
  else
    tar -xzf "$tmp/web.tar.gz" -C "$tmp"
    rm -rf "$ROOT/web"
    mv "$tmp/dist" "$ROOT/web"
  fi

  # 节点页一键安装所需的 Agent 物料，缺失时仅告警不阻断主控部署。
  local agent_files=("aoe-agent-linux-amd64" "aoe-agent-linux-arm64" "aoe-agent-certs.tar.gz" "install.sh")
  local f
  for f in "${agent_files[@]}"; do
    if fetch_optional "$f" "$ROOT/agent-dist/$f"; then
      log "已下载 Agent 物料：$f"
    else
      warn "Agent 物料缺失（节点一键安装将不可用）：$f"
    fi
  done
  run chmod +x "$ROOT/agent-dist"/*.sh 2>/dev/null || true
  run rm -rf "$tmp"
}

write_env() {
  log "生成面板配置文件：$ENV_FILE"
  mkdir -p "$(dirname "$ENV_FILE")"
  local dsn_pass="$MYSQL_PASSWORD"
  cat > "$ENV_FILE" <<EOF
# Aoe-tunnel 主控面板环境配置（由 install-panel.sh 生成，权限 600）
PANEL_HTTP_ADDR=127.0.0.1:18080
PANEL_WEB_DIR=$ROOT/web
PANEL_AGENT_DIST_DIR=$ROOT/agent-dist

MYSQL_DSN=aoe:${dsn_pass}@tcp(127.0.0.1:3306)/aoe_tunnel?charset=utf8mb4&parseTime=True&loc=Local

MQTT_BROKER=ssl://127.0.0.1:8883
MQTT_CA=$ROOT/certs/mqtt-ca.crt

# 站点公网地址：远端被控节点访问主控 API 的唯一入口
PANEL_SITE_BASE_URL=https://$DOMAIN
# Agent 侧 MQTT 接入地址：EMQX 8883 已对外监听，远端 Agent 直连该地址收实时规则
PANEL_AGENT_BROKER=ssl://$DOMAIN:8883

PANEL_JWT_SECRET=$JWT_SECRET
PANEL_NODE_SECRET=$NODE_SECRET
PANEL_ADMIN_USER=$ADMIN_USER
PANEL_ADMIN_PASS=$ADMIN_PASS
PANEL_AGENT_USER=agent
PANEL_AGENT_PASS=$AGENT_PASS
LOG_LEVEL=info
LOG_FORMAT=console
EOF
  run chmod 600 "$ENV_FILE"
}

write_service() {
  log "注册 systemd 服务：$SERVICE_FILE"
  cat > "$SERVICE_FILE" <<EOF
[Unit]
Description=Aoe-tunnel Panel
After=network-online.target docker.service
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=$ROOT
EnvironmentFile=$ENV_FILE
ExecStart=$ROOT/bin/aoe-panel
Restart=on-failure
RestartSec=5
LimitNOFILE=1048576

[Install]
WantedBy=multi-user.target
EOF
  run systemctl daemon-reload
  run systemctl enable --now aoe-panel
}

write_nginx() {
  log "配置 Nginx 反向代理：443 -> 127.0.0.1:18080"
  mkdir -p "$(dirname "$NGINX_SITE")"
  cat > "$NGINX_SITE" <<EOF
# Aoe-tunnel 面板反向代理（由 install-panel.sh 生成）
server {
    listen 80;
    server_name $DOMAIN;
    return 308 https://$DOMAIN\$request_uri;
}

server {
    listen 443 ssl http2;
    server_name $DOMAIN;

    ssl_certificate     $SSL_DIR/$DOMAIN.pem;
    ssl_certificate_key $SSL_DIR/$DOMAIN.key;
    ssl_protocols       TLSv1.2 TLSv1.3;
    ssl_session_cache   shared:SSL:10m;
    ssl_session_timeout 10m;

    client_max_body_size 20m;

    location / {
        proxy_pass http://127.0.0.1:18080;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto https;
        proxy_read_timeout 300s;
    }
}
EOF
  run nginx -t
  run systemctl enable --now nginx
  run systemctl reload nginx
}

health_check() {
  [ "$DRY_RUN" = "true" ] && return
  log "健康检查：https://$DOMAIN/healthz"
  local i
  for i in $(seq 1 45); do
    if curl -fsS --max-time 3 "https://$DOMAIN/healthz" >/dev/null 2>&1; then
      log "面板已可访问"
      return
    fi
    sleep 2
  done
  warn "健康检查未通过，查看日志：journalctl -u aoe-panel -n 80 --no-pager"
}

print_summary() {
  echo
  echo "=================================================================="
  echo " Aoe-tunnel 主控面板部署完成"
  echo "------------------------------------------------------------------"
  echo " 访问地址 : https://$DOMAIN"
  echo " 管理员   : $ADMIN_USER"
  echo " 初始密码 : $ADMIN_PASS"
  echo " 配置文件 : $ENV_FILE"
  echo " 凭据文件 : $CRED_FILE"
  echo " 数据目录 : $ROOT"
  echo "------------------------------------------------------------------"
  echo " 数据表   : GORM AutoMigrate 首次启动创建 13 张表（无演示数据）"
  echo " 控制面   : HTTPS API（上线校验 / 规则同步 / 监控上报）"
  echo " 数据面   : Agent 间 QUIC 9613 / TLS+yamux 9614，主控不转发流量"
  echo " MQTT     : EMQX 0.0.0.0:8883（TLS，供远端 Agent）；1883/18083 仅 127.0.0.1"
  echo "=================================================================="
  echo " 提示：请确认云厂商安全组 / 防火墙已放行 8883，否则异地 Agent 无法接入"
}

main() {
  need_root
  if [ "$DRY_RUN" != "true" ]; then
    check_os
    detect_arch
    install_deps
  else
    detect_arch
    COMPOSE=(docker compose)
  fi
  prompt_domain
  validate_domain
  dns_check
  install_acme
  issue_cert
  gen_mqtt_certs

  MYSQL_ROOT_PASSWORD="$(rand_hex 16)"
  MYSQL_PASSWORD="$(rand_hex 16)"
  EMQX_COOKIE="aoe_$(rand_hex 11)"
  EMQX_DASH_PASS="$(rand_hex 12)"
  JWT_SECRET="$(rand_hex 32)"
  NODE_SECRET="$(rand_hex 32)"
  ADMIN_PASS="$(rand_hex 12)"
  AGENT_PASS="$(rand_hex 12)"

  mkdir -p "$ROOT"
  write_compose
  start_infra
  install_release
  write_env
  write_service
  write_nginx
  health_check

  umask 077
  cat > "$CRED_FILE" <<EOF
panel: https://$DOMAIN
admin_user: $ADMIN_USER
admin_pass: $ADMIN_PASS
agent_user: agent
agent_pass: $AGENT_PASS
mysql_user: aoe
mysql_pass: $MYSQL_PASSWORD
emqx_dashboard: admin / $EMQX_DASH_PASS
EOF
  chmod 600 "$CRED_FILE"
  print_summary
}

main "$@"
