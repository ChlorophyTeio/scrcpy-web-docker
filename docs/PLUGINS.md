# 插件 API v1

服务清单位于镜像 `/opt/scrcpycat/plugins/*.json`；管理员可通过持久化 `/data/plugins/*.json` 覆盖同 ID 的清单。复制整个清单再设 `enabled:false` 可以禁用插件。最多加载 32 个服务。

```json
{
  "id":"sample",
  "name":"示例工具",
  "description":"功能说明",
  "frontend":"sample",
  "api_version":1,
  "target":"http://sample-worker:8080",
  "health_path":"/health",
  "health_match":{"id":"sample","api_version":1}
}
```

健康接口返回 HTTP 200 JSON，且 `health_match` 字段匹配时才显示入口。单次探测超时 1.5 秒，结果缓存 3 秒；浏览器每 10 秒检查，切回标签页也会检查。清单更改会使缓存失效。服务启动、停止无需重建主镜像。

- `GET /api/plugins`：管理员读取插件及可用状态，不返回内部服务地址。
- `POST /api/plugins/:id/session`：服务就绪后建立专属 HttpOnly 会话 cookie，返回 UI 和 API 前缀。
- `/api/plugins/:id/proxy/*`：HTTP / WebSocket 代理。管理员权限及令牌撤销继续生效；主站 Authorization 和 Cookie 不转发给服务。
- `/plugins/:pluginId`：统一 UI 容器。Vue 界面映射在 `frontend/web-app/src/plugins/registry.js` 中注册；新增界面属于编译过程。

服务清单是服务器管理员维护的可信配置，浏览器不能提交任意代理目标。API v1 仅供管理员使用。未知/停止/禁用的服务不能创建插件会话。

MAA 的实现目录：`plugins/maa/`（worker、镜像、清单、资源）以及 `frontend/web-app/src/plugins/maa/`（Vue 组件、设置迁移、样式与测试）。旧 `/maa/*` 认证代理暂保留以兼容历史客户端，新界面全部使用通用插件 API。
