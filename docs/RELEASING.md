# 发布维护

项目：<https://github.com/ChlorophyTeio/scrcpy-web-docker>。Docker Hub 命名空间：`chlorophyimo`。首版使用 `v0.1.0` Git 标签及 `0.1.0` 镜像标签。

| 组件 | 镜像 | Unraid 模板 |
| --- | --- | --- |
| 主容器 | `chlorophyimo/scrcpycat` | `unraid/my-scrcpycat.xml` |
| 可选 MAA | `chlorophyimo/scrcpycat-maa` | `unraid/my-maa-plugin.xml` |

## GitHub Actions

1. 运行 `python3 tools/check-release.py`，完成前端、Go 和目标架构验证后提交源码。
2. Repository Secrets 中配置专用 Docker Hub access token：`DOCKERHUB_TOKEN`。`DOCKERHUB_USERNAME` 可选，默认 `chlorophyimo`。不要把 token 写进源码或命令参数。
3. Repository Variables 可覆盖 `DOCKERHUB_NAMESPACE`（默认 `chlorophyimo`）与 `IMAGE_NAME`（默认 `scrcpycat`）。
4. 创建并推送 `v版本号` Git 标签。手动运行 **Build or publish images**，填写不带 `v` 的版本号；`publish=false` 只构建，`publish=true` 上传两个镜像。
5. 工作流从指定标签构建 Android agent、scrcpy server 和 linux/amd64、linux/arm64 镜像。源码提交号写入镜像标签信息。稳定版标记 `latest`，预发布版不覆盖 `latest`。
6. 验证工作流与镜像清单后创建 GitHub Release，附上两份 Unraid XML、源码包及 SHA-256 校验文件。Unraid Community Applications 收录需要另外申请。

普通提交只运行检查，不上传镜像。仓库访问凭据通过 GitHub 登录授权管理，不放进发布目录。更改仓库或镜像名后运行 `tools/generate-unraid.py` 更新项目、镜像、支持和图标链接。

## 本地构建

```bash
make build-agent-docker build-scrcpy-server
docker build --build-arg VERSION=0.1.0 --build-arg REVISION="$(git rev-parse HEAD)" -t chlorophyimo/scrcpycat:0.1.0 .
docker build -f plugins/maa/Dockerfile -t chlorophyimo/scrcpycat-maa:0.1.0 .
```

在 amd64 主机普通 `docker build` 得到的镜像仅覆盖 amd64。多架构版本用发布工作流构建，并以实际推送的 manifest 为准；不要将 Go 交叉编译或 QEMU 构建称为 arm64 手机主机的实测。

首次启动 MAA 插件会下载官方 MaaCore 和资源。发布包不包含运行数据、数据库备份、ADB/TLS 私钥、账号密码或会话快照。Android 组件从当前仓库内源码构建，避免复用与源码版本不一致的历史二进制。
