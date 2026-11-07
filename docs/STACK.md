# Compose stack

此 stack 用于新部署或由维护者自行规划的迁移。下载发布文件不会更改已有 Unraid 容器、网络、挂载或数据库。需要 Linux Docker、Docker Compose **2.20+** 和 Python 3；当前发布镜像为 **linux/amd64**。

## 一键部署

在 [GitHub Release](https://github.com/ChlorophyTeio/scrcpy-web-docker/releases/latest) 下载 `scrcpycat-stack-版本号.tar.gz`，解压并进入目录：

```bash
python3 init-env.py --output .env --host-ip YOUR_SERVER_LAN_IP --public-url https://YOUR_SERVER_LAN_IP:9015
# 用本地编辑器打开 .env，查看生成的管理员账号密码；不要公开该文件。
docker compose pull
docker compose up -d
```

默认包括 `scrcpy-web`、`maa-web`、`scrcpycat-cache` 和新的 PostgreSQL 17。服务使用命名卷，数据库和缓存均无宿主机端口；MAA 通过主网页访问。首次启动 MAA 需要从 GitHub 下载核心和资源，可能晚于主服务就绪。

从源码仓库操作时，对应命令为：

```bash
python3 tools/init-env.py --output deploy/.env --host-ip YOUR_SERVER_LAN_IP --public-url https://YOUR_SERVER_LAN_IP:9015
docker compose --env-file deploy/.env -f deploy/docker-compose.yml pull
docker compose --env-file deploy/.env -f deploy/docker-compose.yml up -d
```

访问 `SCRCPYCAT_PUBLIC_URL`。首次自动生成自签名证书，确认自己的服务器后信任证书，或由反向代理提供可信 HTTPS。USB 调试需在手机上授权。完整映射 `/dev/bus/usb`，支持后续插拔；不另起本机 agent。

`SCRCPYCAT_WEBRTC_HOST_IPS` 填客户端可达的服务器 IP，UDP 40000–40100 需通过防火墙；公网环境按需配置 TURN。`SCRCPYCAT_PUBLIC_URL` 必须与浏览器入口一致。MAA 可在仅有一台已授权设备时自动选择，或在 `.env` 填 `MAA_DEVICE`。

## 使用现有 PostgreSQL 17

```bash
python3 init-env.py --output .env --database existing --host-ip YOUR_SERVER_LAN_IP --public-url https://YOUR_SERVER_LAN_IP:9015
```

编辑生成的 `.env`：

```dotenv
COMPOSE_PROFILES=
SCRCPYCAT_NETWORK=YOUR_EXISTING_DOCKER_NETWORK
SCRCPYCAT_EXTERNAL_NETWORK=true
SCRCPYCAT_POSTGRES_DSN=postgres://YOUR_APP_USER:URL_ENCODED_PASSWORD@postgresql17:5432/YOUR_APP_DATABASE?sslmode=disable
```

使用已经存在的自定义 bridge 网络，数据库容器与 stack 在同一网络内，以数据库容器名连接内部 5432 端口。先为应用创建专用账号和数据库，账号需有该数据库内建表/迁移权限。密码含特殊字符时进行 URL 编码。外部服务器数据库可直接填写可达地址，网络设置保留默认值。`--database existing` 留空 DSN 并要求填好后启动，避免误创建另一套数据库。

`SCRCPYCAT_POSTGRES_PASSWORD` 仅供可选的新数据库初始化；接现有数据库时不会修改其密码。所有模式保留该变量，以便 Compose 解析配置。

## 管理与更新

```bash
docker compose ps
docker compose logs --tail 100 scrcpy-web maa-web scrcpycat-cache
# 停用可选 MAA；入口会自动隐藏。
docker compose stop maa-web
# 恢复插件。
docker compose up -d maa-web
```

更新前备份 `.env`、应用和 MAA 命名卷及 PostgreSQL 数据。将 `.env` 中三个镜像标签改为同一发布版本，然后 `docker compose pull && docker compose up -d`。MAA 核心和资源也可在网页内独立更新。`docker compose down` 保留数据；`down -v` 会删除 stack 的命名卷，请勿当作常规更新命令。

沿用此前公开 Compose 的 `redis-data` 卷名，缓存服务改名为 `scrcpycat-cache`，保留内部 `redis` DNS 别名。从早期公开 Compose 升级时需自行确认项目名和卷名，并在备份后移除已停用的旧 `redis` 服务；此发布不自动迁移任何现有容器。

缓存基于官方 Redis 7.4.12，启用 AOF 和健康检查，限定内部网络使用；不要公开 6379。来源与完整许可见 [cache/SOURCE.md](../cache/SOURCE.md)。
