#!/usr/bin/env bash
# 生成自用 CA + 服务端/客户端证书（数据面 mTLS 与控制面 MQTT TLS 共用）。
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_DIR="${ROOT_DIR}/deploy/certs"

mkdir -p "${OUT_DIR}"
cd "${OUT_DIR}"

if [[ -f ca.crt && -f server.crt && -f client.crt ]]; then
  echo "certificates already exist in ${OUT_DIR}, skip (remove the directory to regenerate)"
  exit 0
fi

openssl genrsa -out ca.key 2048
openssl req -x509 -new -nodes -key ca.key -sha256 -days 3650 \
  -subj "/C=CN/O=Aoe-Tunnel/CN=Aoe-Tunnel-CA" -out ca.crt

cat > server.ext <<'EOF'
basicConstraints=CA:FALSE
keyUsage=digitalSignature,keyEncipherment
extendedKeyUsage=serverAuth
subjectAltName=DNS:aoe-node,DNS:localhost,DNS:ask.30ss.vip,IP:127.0.0.1,IP:::1
EOF
openssl genrsa -out server.key 2048
openssl req -new -key server.key -subj "/C=CN/O=Aoe-Tunnel/CN=aoe-node" -out server.csr
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial \
  -days 3650 -sha256 -extfile server.ext -out server.crt

cat > client.ext <<'EOF'
basicConstraints=CA:FALSE
keyUsage=digitalSignature
extendedKeyUsage=clientAuth
EOF
openssl genrsa -out client.key 2048
openssl req -new -key client.key -subj "/C=CN/O=Aoe-Tunnel/CN=aoe-agent" -out client.csr
openssl x509 -req -in client.csr -CA ca.crt -CAkey ca.key -CAcreateserial \
  -days 3650 -sha256 -extfile client.ext -out client.crt

rm -f server.csr client.csr server.ext client.ext ca.srl

# EMQX 容器内以非 root 用户运行，key 必须可读；自用环境接受 644，生产请收敛权限。
chmod 644 ./*.crt ./*.key

openssl verify -CAfile ca.crt server.crt client.crt
echo "certificates written to ${OUT_DIR}"
