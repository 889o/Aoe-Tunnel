# Aoe-tunnel Web 控制台

Vue3 + TypeScript + Vite + Tailwind4 + shadcn-vue（reka-ui 原语）+ ECharts，纸感实体 UI。

## 运行

```bash
cd web
npm install
VITE_ADMIN_KEY=s5key npm run dev   # http://127.0.0.1:5173
```

`/api/v1` 由 Vite 代理到面板 `http://127.0.0.1:18080`（见 `vite.config.ts`），需要面板与 Agent 已启动。

构建：`npm run build`（先 `vue-tsc --noEmit` 类型检查，产物在 `dist/`）。

## 登录与角色

- 登录页输入面板 `X-API-Key`，通过 `GET /api/v1/nodes` 校验，成功后存 `localStorage` 的 `aoe-api-key`。
- `VITE_ADMIN_KEY` 与输入的 Key 相同 → 管理员（全部 CRUD）；其他合法 Key → 只读（隐藏写操作）。
- 未配置 `VITE_ADMIN_KEY` 时全部为只读视图。首期不接 JWT，后端无角色接口。

## 主题

- 石墨暗色为默认主题，纸白为亮色；切换写入 `localStorage` 的 `aoe-theme`，`index.html` 内联脚本在首屏预置，避免闪白。
- 设计 token 在 `src/styles/main.css`：`--paper-0/1/2/3`、`--ink-*`、`--accent`、`--raise-*`/`--sink-*`，统一左上光源的凸起（`.paper-card`）与凹槽（`.paper-sink`、`.paper-input`），按压回弹见 `.paper-btn`。

## 目录

```
src/api          axios 封装（注入 X-API-Key、解包 {code,message,data}）与接口函数
src/types        Node / PortRule / TrafficStat，字段对齐面板 JSON
src/stores       auth（Key + 管理员判定）、theme（本地持久化）
src/components/paper  PaperCard/Button/Input/Select/Switch/Dialog/Badge/Progress/EChart/ThemeToggle
src/layouts      AppLayout（侧栏导航 + 主题/退出）
src/pages        LoginPage / NodesPage / PortsPage / TrafficPage
```

## 页面

1. **Key 登录**：校验 Key，401 时不落库。
2. **节点**：状态、QUIC/TLS 地址、最后心跳；管理员可新建/编辑，改地址提示「自动重推端口规则」。
3. **端口管理**：规则列表（限速、配额进度、状态），管理员可新建/编辑/删除；入口/出口节点在编辑时只读。
4. **流量看板**：按规则与时间窗口查询 `GET /port-rules/:id/stats`，汇总卡片 + 上下行曲线（每个点为一次聚合上报）。

## 约束

只读面板 API，不新增后端接口；不修改 `internal/panel` 与 `internal/agent`。
