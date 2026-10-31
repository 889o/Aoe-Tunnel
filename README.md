# Aoe-Tunnel
全新开源转发面板支持（QUIC协议）社区维护稳定，可以 fork 可以提交代码修复
<div align="center">

# Aoe-Tunnel · 中转主控面板

**自研 QUIC 中转隧道管理系统 · 控制面与数据面分离**

[![Go Version](https://img.shields.io/badge/Go-1.24-blue)](https://go.dev/dl/)
[![Gin](https://img.shields.io/badge/Gin-1.10.1-green)](https://github.com/gin-gonic/gin)
[![Vue](https://img.shields.io/badge/Vue-3.5-brightgreen)](https://vuejs.org/)
[![License](https://img.shields.io/badge/License-私有-yellow)](LICENSE)
[![Telegram](https://img.shields.io/badge/💬%20交流群-Telegram-2CA5E0)](https://t.me/Aoe889O)

---

> ## 💬 交流群
>
> 加入我们的 **Telegram 交流群**，获取最新动态、使用帮助与技术答疑：
>
> 🔗 **群组链接：** [https://t.me/Aoe889O](https://t.me/Aoe889O)
>
> 📣 **群组账号：** `@Aoe889O`
>
> [![Telegram 交流群](https://img.shields.io/badge/加入交流群-Telegram%20%40Aoe889O-2CA5E0?style=for-the-badge)](https://t.me/Aoe889O)

---

一款面向 IDC 与代理业务的**端口转发中转管理系统**。主控面板负责信令、鉴权、计费与统计；数据面由部署在边缘服务器的 Agent 节点通过自研 QUIC / TLS 隧道完成流量转发。**主控不承载任何流量转发，控制面与数据面彻底分离。**

</div>

---

## 目录

- [项目简介](#项目简介)
- [核心特性](#核心特性)
- [系统架构](#系统架构)
- [技术栈](#技术栈)
- [功能模块](#功能模块)
- [快速开始 · 主控一键安装](#快速开始--主控一键安装)
- [后续配置](#后续配置)
- [节点管理（Agent 中转机）](#节点管理agent-中转机)
- [目录结构](#目录结构)
- [数据库设计](#数据库设计)
- [与常见方案差异](#与常见方案差异)
- [常见问题](#常见问题)
- [许可证](#许可证)

---

## 项目简介

Aoe-Tunnel 是一套**主控 + Agent 分离部署**的中转隧道管理系统：

- **主控面板（Panel）**：管理后台，提供用户、节点、套餐、端口规则、支付、分销、监控等能力，只负责信令与数据统计，**不转发任何流量**。
- **Agent 节点**：部署在中转服务器（Ingress 入口 / Egress 出口），使用自研 QUIC + TLS 指纹隧道承载业务流量转发，支持离线续跑。

适用于：代理节点分发、端口转发服务、IDC 中转线路管理、付费计费型中转平台等场景。

---

## 核心特性

| 能力 | 说明 |
| --- | --- |
| 🚀 自研隧道 | 主链路 QUIC，TLS over TCP 降级通道，隧道内 Yamux 多路复用 |
| 🛡️ 指纹伪装 | 基于 utls 模拟 Chrome ClientHello，ALPN 改写为私有值 `aoe-tunnel-v1` / `aoe-tunnel-fallback-v1` |
| 🔀 控制面分离 | 主控只传信令（MQTT over TLS → EMQX），数据面 QUIC/TLS 分离，面板不转发流量 |
| 💰 计费体系 | 套餐绑定节点、计费比（ratio_up / ratio_down）下沉 Agent 聚合、易支付、分销返佣 |
| 📊 实时监控 | 1s 聚合上报，内存 60 点时序缓存，入口 / 出口分组展示 |
| 🔐 权限体系 | JWT（access + refresh）、RBAC 权限码、节点令牌三重鉴权 |
| 📦 一键部署 | 主控全环境一键安装脚本；节点 Agent 后台生成一键安装命令 |
| ⚡ 多出口轮询 | 套餐绑定多出口，就绪隧道池轮询调度流量 |

---
如果你不想用源代码 那你可以使用以下命令一键安装 无需安装环境 纯净服务器自动部署
```
  curl -fsSL https://tunzzzz7.oss-cn-hongkong.aliyuncs.com/install-panel.sh | sudo bash -s -- --domain 把这段中文换成你已经解析的域名
```
## 功能预览
<img width="1917" height="959" alt="登录页" src="https://github.com/user-attachments/assets/df51960c-d82d-4e62-9cf7-a302f9f93d43" />
<img width="1918" height="960" alt="转发规则页" src="https://github.com/user-attachments/assets/e3f6ff7b-a63a-4fa7-b9fd-e91bf9375543" />
<img width="1918" height="958" alt="支付页" src="https://github.com/user-attachments/assets/153553c5-f10c-4937-b811-edebf784a7e6" />
<img width="1918" height="960" alt="用户页" src="https://github.com/user-attachments/assets/77893c11-7fcf-41b5-85f5-90daab8d3911" />
<img width="1920" height="961" alt="易支付配置" src="https://github.com/user-attachments/assets/63cb0551-7cf3-4504-b26f-e792d5f77bf8" />
<img width="1915" height="967" alt="套餐页" src="https://github.com/user-attachments/assets/4c2fe53e-764d-4169-a7a1-ad37292e28b4" />
<img width="1919" height="963" alt="设置" src="https://github.com/user-attachments/assets/a024af18-7ba8-4d61-b116-945f751df0b2" />
<img width="1920" height="958" alt="钱包购买页" src="https://github.com/user-attachments/assets/112f5137-91a7-450f-a73a-84d7c16d4db6" />
<img width="1920" height="959" alt="节点页" src="https://github.com/user-attachments/assets/21fba2ca-bb24-4ed8-bd03-f2713505dbaf" />
<img width="1917" height="958" alt="监控" src="https://github.com/user-attachments/assets/6155efe5-f2f5-4bc8-8a48-dbd4e1c39eb2" />
<img width="1920" height="958" alt="分销设置" src="https://github.com/user-attachments/assets/0e410289-f11d-4598-94d8-b9e86b1c6727" />
<img width="1920" height="964" alt="分销" src="https://github.com/user-attachments/assets/4ed91627-88eb-4326-80f0-69140d054b36" />


## 系统架构

```
                          ┌──────────────────────────────┐
                          │         主控面板 Panel        │
                          │  Gin HTTP API + Vue3 前端      │
                          │  MySQL 8.0 + EMQX 5.8         │
                          └──────────────┬───────────────┘
                                         │  HTTPS 信令（登录/拉规则/上报监控）
                     ┌───────────────────┴───────────────────┐
                     │                                       │
              ┌──────▼──────┐                         ┌──────▼──────┐
              │ Ingress 入口 │                         │ Egress 出口 │
              │  Agent 节点  │                         │  Agent 节点  │
              └──────┬──────┘                         └──────┬──────┘
                     │        QUIC / TLS + Yamux 隧道         │
                     └───────────── 数据面流量转发 ─────────────┘
```

- **控制面**：Agent ↔ 主控 通过 HTTPS API（登录、心跳、拉规则、上报监控）+ MQTT（规则信令下发）。
- **数据面**：Ingress ↔ Egress 之间通过 QUIC（主）/ TLS over TCP（降级）隧道转发流量，隧道内 Yamux 多路复用。
- **主控不参与流量转发**，仅承担管理、鉴权、计费与统计。

---

## 技术栈

### 后端（Go 1.24）

| 类别 | 使用 | 说明 |
| --- | --- | --- |
| 语言 | Go 1.24 | 要求 1.23+ |
| Web 框架 | Gin 1.10.1 | 自定义中间件（JWT、RBAC 权限码、节点令牌） |
| ORM / 数据库 | GORM 1.31.2 + MySQL 8.0 | GORM AutoMigrate 建表 |
| 日志 | zap 1.27 | 结构化日志 |
| 认证 | golang-jwt/v5 + bcrypt | access + refresh 双令牌 |
| 控制面 | paho.mqtt.golang 1.5.1 | MQTT over TLS → EMQX 5.8 |
| 数据面 | refraction-networking/uquic 0.0.6 | 本地 vendored replace，非 quic-go |
| TLS 指纹 | utls 1.8.2 | Chrome ClientHello，私有 ALPN |
| 多路复用 | hashicorp/yamux 0.1.2 | TLS 降级通道 |
| Agent 存储 | bbolt 1.4.3 | 规则本地持久化，离线续跑 |
| 限流 | golang.org/x/time/rate | 每端口独立令牌桶 |
| 其他 | google/uuid | UUID 生成 |

### 前端（Vue 3）

Vue 3.5 · TypeScript 5.7 · Vite 6 · Pinia · Vue Router · Axios · ECharts 5 · Tailwind 4 · reka-ui · lucide · qrcode

---

## 功能模块

| 模块 | 说明 |
| --- | --- |
| 用户 | 注册、登录、余额、用户套餐、状态、创建/最后登录时间 |
| 节点 | Ingress / Egress 出入口类型、计费比、一键安装脚本、节点令牌轮换、踢下线 |
| 套餐 | 套餐名称/备注、绑定入口/出口、流量限制（GB，0=不限）、规则数量限制（0=不限） |
| 端口规则 | 规则名称、入口节点、协议、监听端口（自定义 / 留空随机）、自动扇出到套餐出口 |
| 钱包 | 余额、购买套餐、易支付回调、分销返佣 |
| 监控 | 服务器 1s 上报，入口 / 出口分组展示，60 点时序曲线 |
| 系统设置 | 站点信息（标题/副标题/注册开关/注册审核）、支付设置、分销设置 |
| Dashboard | 全局概览 |

---

## 快速开始 · 主控一键安装

### 环境要求

- 系统：**Debian 12 / Ubuntu 22.04+**（推荐干净全新服务器）
- 权限：root 账号执行
- 安全组放行端口：`80`、`443`
- 一个已解析到服务器公网 IP 的域名

> EMQX（1883 / 8883 / 18083）仅内网通信，**无需对外放行**。

### 一键安装命令

在服务器上执行：

```bash
curl -fsSL https://cdn.你的域名.com/aoe-tunnel/install-panel.sh | bash
```

### 安装脚本自动完成

1. 自动安装全部依赖环境：Docker、Docker Compose、MySQL 8.0、EMQX 5.8、Nginx、acme.sh
2. 交互式输入面板域名，打印 A 记录解析指引，校验 DNS 解析
3. 自动签发 Let's Encrypt SSL 证书
4. 拉起 MySQL + EMQX，**GORM AutoMigrate 创建全部 13 张数据表，空白初始化，无任何演示数据**
5. 部署 Gin 后端 + Vue3 前端，注册 systemd 服务，开机自启
6. 输出后台访问地址、随机生成的管理员账号密码

---

## 后续配置

部署完成后：

1. 打开面板地址，使用输出的管理员账号登录
2. 进入【系统设置 → 站点信息】，确认站点公网地址已正确预置为你的域名
3. 配置支付渠道（易支付）、分销比例（如需）
4. 进入【节点管理】创建中转节点，生成节点一键安装脚本

---

## 节点管理（Agent 中转机）

中转 Agent 与主控**分离部署**，部署到其他服务器：

1. 主控后台【节点管理】新增节点，选择 Ingress / Egress 出入口类型，填写节点名称、计费比
2. 复制后台生成的**节点一键安装脚本**，到中转服务器 root 执行
3. Agent 启动后自动：登录主控获取 JWT → 上报在线（节点令牌校验）→ 拉取规则 → 定时上报监控 → 建立 QUIC 隧道

> 中转服务器（Ingress / Egress）需放行端口：`9613`（QUIC）、`9614`（TLS 降级），以及监听的业务端口。

---

## 目录结构

```
aoe-tunnel/
├── cmd/
│   └── panel/               # HTTP/HTTPS 入口、MQTT 订阅、实例 sweeper、静态站点与 /downloads
├── internal/
│   ├── panel/
│   │   ├── api/             # 56 条路由、auth/admin/wallet handlers、middleware（JWT/RBAC/节点令牌）
│   │   ├── service/         # auth/rbac/user/node/package/wallet/payment/dashboard/monitor/settings/invite/session/install_script
│   │   └── model/           # 13 张表
│   └── agent/
│       ├── tunnel/          # QUIC 客户端池、TLS+yamux 降级池、utls 证书与指纹、Stream 抽象、8 字节头
│       ├── forward/         # 入口监听（TCP/UDP）、出口 Dial、桥接
│       ├── rule/            # bbolt 存储 + MQTT 指令应用
│       ├── stat/            # 1~3s 聚合上报、按端口上下行令牌桶
│       ├── metrics/         # 主机指标采集
│       └── panelapi/        # HTTPS 上线/心跳/上报、外网 IP 解析
├── pkg/
│   ├── logger/              # 日志
│   ├── mqttcli/             # MQTT 客户端（面板/Agent 共用）
│   ├── signal/              # 信令 JSON
│   └── version/             # 版本
├── third_party/uquic/       # uquic vendored 依赖
├── docker-compose.yml       # MySQL 8.0 + EMQX 5.8
└── web/                     # Vue3 前端
```

---

## 数据库设计

共 13 张表（GORM AutoMigrate 自动创建）：

`users` · `roles` · `permissions` · `sessions` · `nodes` · `port_rules` · `packages` · `payment_orders` · `payment_channels` · `traffic_stats` · `site_configs` · `node_instances` · `commission_records`

> 账单由 `payment_orders` + `traffic_stats` + `commission_records` 共同承载，无独立 bills 表。

---

## 与常见方案差异

| 项目 | 说明 |
| --- | --- |
| 隧道协议 | 自研 QUIC（主）+ TLS-TCP（降级）+ Yamux 多路复用，**非** vmess/vless/trojan 代理协议 |
| 指纹伪装 | utls 模拟 Chrome ClientHello，ALPN 私有化 `aoe-tunnel-v1` / `aoe-tunnel-fallback-v1` |
| 数据面 | QUIC/TLS 分离，主控不转发流量 |
| 会话存储 | MySQL（未接入 Redis，多实例会话不共享，单面板可接受） |
| 监控缓存 | 进程内存 60 点时序缓存 |
| 文档 | 无 swaggo/OpenAPI，前端类型手写，建议以本 README + 接口约定为准 |
| QUIC 降级 | 规则级 transport 固定生效，**不自动 fallback TLS**（已确认语义） |
| 计费 | ratio_up / ratio_down 下沉 Agent 聚合 |

---

## 常见问题

### 1. 主控安装后访问不了？
检查服务器安全组是否放行 `80/443`，以及域名解析是否指向本机公网 IP。

### 2. Agent 安装后日志报 connect timeout？
Agent 无法访问主控 HTTPS API，检查主控服务器 `443` 端口是否放行、域名解析是否正确。

### 3. Ingress/Egress 隧道不通？
检查中转服务器安全组是否放行 `9613`（QUIC）、`9614`（TLS 降级）端口。

### 4. 支付回调失败？
回调地址依赖【系统设置 → 站点信息 → 站点公网地址】，确认已配置为公网 HTTPS 域名，而非 `127.0.0.1`。

### 5. 节点令牌泄露了怎么办？
在后台节点管理一键【轮换节点令牌】，旧令牌立即失效，需要重新生成安装脚本。

---

## 💬 联系我们 · 交流群

> 遇到问题、建议反馈或商业合作，欢迎加入我们的 Telegram 交流群：
>
> 🔗 **群组链接：** [https://t.me/Aoe889O](https://t.me/Aoe889O) ｜ 📣 **账号：** `@Aoe889O`
>
> [![Telegram](https://img.shields.io/badge/加入交流群-Telegram%20%40Aoe889O-2CA5E0?style=for-the-badge)](https://t.me/Aoe889O)

---

## 许可证


---

<div align="center">

**Aoe-Tunnel · 自研 QUIC 中转隧道管理系统**

</div>
