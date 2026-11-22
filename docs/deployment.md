# 部署 LAN Clipboard

## 1. 首次启动

需要 Docker Engine 或 Docker Desktop，以及 Docker Compose v2。稳定版本的镜像包含 Linux x86-64 和 ARM64 构建。在本项目的 GitHub **Releases** 中选定版本，下载附件中的 **compose.yaml**，然后在文件所在目录运行：

```sh
docker compose config --quiet
docker compose up -d
docker compose ps
```

等待状态显示 `healthy`，再打开 [http://127.0.0.1:18080](http://127.0.0.1:18080)。访问 [http://127.0.0.1:18080/api/stats](http://127.0.0.1:18080/api/stats) 应返回带有 `totalClips` 和 `totalSize` 的 JSON。空实例的两项值均为 0。可以打开两个浏览器窗口，确认一边发送文本或上传文件，另一边即时收到。

Release 附件通过镜像摘要锁定经过验收的版本，可以从公开 GHCR 匿名拉取。若要从源码自行构建，在仓库根目录执行：

```sh
docker build -t lan-clipboard:local .
docker compose up -d --pull never
```

## 2. 数据存储与配置

默认将 SQLite 数据库、WAL/SHM、上传文件和预览缓存写入命名卷 `lan-clipboard_app-data`，容器内的挂载点是 `/app/data`。在同一个 Compose 项目目录里执行更新，Docker 会复用该卷。`docker compose down -v` 会删除命名卷和其中的数据。

有额外配置时，可将 Release 附件 `env.example` 复制为同目录的 `.env`。从源码使用仓库的 `.env.example`。其中支持以下设置。

| 变量 | 用途 |
| --- | --- |
| `CLIP_BIND_ADDRESS` | 宿主机端口绑定地址，默认为 `127.0.0.1` |
| `CLIP_HOST_PORT` | 宿主机访问端口，默认为 `18080` |
| `CLIP_IMAGE` | 覆盖镜像版本，填写完整镜像名称和固定标签 |
| `CLIP_PUBLIC_ORIGIN` | 使用自定义域名或 HTTPS 时，填浏览器访问的来源，如 `https://clip.example.test`，包含非默认端口时写上端口 |
| `CLIP_HOST_DATA_DIR` | 配合 `compose.bind.yaml` 使用，填已存在的本地数据目录 |

默认在应用容器中以 UID/GID `10001:10001` 运行。本地目录挂载时目录必须允许该用户写入。使用本地磁盘保存 SQLite 数据，单个数据目录对应一个应用实例。

## 3. 局域网和 HTTPS

服务为共享剪贴板。访问端口的客户端都可以读取、添加和删除内容。只给可信设备开放网络访问。较旧的 Docker Engine 有本机端口映射的局域网可达例外，使用旧版 NAS 时也应在宿主机与网络层限制访问范围。[Docker 端口绑定说明](https://docs.docker.com/engine/network/port-publishing/)

- **同一台电脑上使用**时，保持默认本机地址。
- **其他局域网设备直连**时，将 `CLIP_BIND_ADDRESS` 设为 Docker 主机的局域网 IP，通过 `http://<局域网IP>:18080` 访问。用防火墙限制可达设备或网段。
- **通过反向代理使用域名**时，在 `.env` 设置 `CLIP_PUBLIC_ORIGIN`。使代理保留浏览器传入的 Host、将整个站点转发到应用端口，并支持 `/ws` 的 WebSocket Upgrade。代理与应用同机可连 `127.0.0.1:18080`。代理在另一台主机上时，绑定服务器的局域网 IP 并限制该端口的访问范围。
- **跨不可信网络访问**时，使用 VPN 或在反向代理上配置身份验证，覆盖页面、API、上传下载和 WebSocket。HTTPS 为连接加密，也可让浏览器使用完整剪贴板 API。

DSM 登录门户可用「高级 → 反向代理服务器」配置目标 `http://127.0.0.1:18080`，并在自定义标题里新增 WebSocket 规则。如果代理由 OpenWrt 上的服务提供，本地 DNS 记录指向 OpenWrt 的局域网地址，上游填 NAS 的局域网 IP 与应用端口。证书安装在终止 HTTPS 的代理上。

## 4. 群晖及宿主机目录挂载

群晖需要 Container Manager 或提供 Docker Compose v2 的等效环境。若希望用 DSM 套件管理应用数据，下载同一 Release 的 `compose.yaml` 和 `compose.bind.yaml`，放在同一项目目录。以下命令仅用于 **首次创建空数据目录**：

```sh
sudo mkdir -p /volume1/docker/lan-clipboard/data
sudo chown 10001:10001 /volume1/docker/lan-clipboard/data
sudo chmod 700 /volume1/docker/lan-clipboard/data
```

在项目目录的 `.env` 中设置：

```dotenv
CLIP_HOST_DATA_DIR=/volume1/docker/lan-clipboard/data
```

根据反向代理所在位置设置 `CLIP_BIND_ADDRESS` 和 `CLIP_PUBLIC_ORIGIN`，然后运行：

```sh
sudo docker compose -f compose.yaml -f compose.bind.yaml config --quiet
sudo docker compose -f compose.yaml -f compose.bind.yaml up -d
sudo docker compose -f compose.yaml -f compose.bind.yaml ps
```

从旧版 Bind Mount 部署升级时，继续使用原来的数据目录和 Compose 项目名。每次操作都同时指定两个文件。首次切换前用 `docker compose -f compose.yaml -f compose.bind.yaml config` 查看 `/app/data` 对应的宿主机目录。旧版通过自定义域名访问的实例，升级前还要配置 `CLIP_PUBLIC_ORIGIN`。

## 5. 更新、回滚与备份

更新前记录正在使用的镜像标签。将新 Release 的安装文件放入原项目目录，并在已有 `.env` 时检查 `CLIP_IMAGE` 覆盖值。也可以直接在 `.env` 中指定要部署的版本，再执行：

```sh
docker compose pull clipboard
docker compose up -d --no-build --pull never clipboard
docker compose ps clipboard
```

确认镜像版本、状态和浏览器访问。使用宿主机目录的安装在这些命令中继续指定两个 Compose 文件。回滚时将 `CLIP_IMAGE` 设回旧镜像，再执行同样的命令。若该版本发布说明有数据库迁移要求，先按发布说明核对回滚兼容性或恢复备份。

SQLite 使用 WAL 模式。完整备份需要保留数据库与上传文件的同一时点状态。选择空闲时段短暂停止应用，在私有目录做归档，然后启动。默认命名卷可在 Compose 项目目录中执行：

```sh
umask 077
docker compose stop clipboard
docker compose run --rm --no-deps -T --entrypoint tar clipboard -C /app/data -czf - . > clipboard-backup.tar.gz
docker compose start clipboard
tar -tzf clipboard-backup.tar.gz > /dev/null
```

将同名归档移到独立的备份位置再做下一次备份。绑定挂载可以停服务后在宿主机归档整个数据目录。在应用运行时，单独复制数据库文件可能漏掉 WAL 中尚未检查点写入的更改，数据库在线快照也无法同时冻结上传目录。[SQLite WAL](https://www.sqlite.org/wal.html) · [SQLite Backup API](https://www.sqlite.org/backup.html)

恢复时将归档还原到新的数据卷或空目录，先用与备份匹配的镜像启动，确认历史记录、搜索和文件下载后再切换。宿主机目录恢复时应保持 UID/GID `10001:10001` 可写。

## 6. 可选 Office 文件预览

在 `.env` 中启用配置后运行 `docker compose up -d`：

```dotenv
COMPOSE_PROFILES=office
CLIP_OFFICE_CONVERTER_URL=http://office:3000
```

应用将需要预览的 Word 或 PowerPoint 文件发给同一 Compose 项目内的 Gotenberg，转换服务使用内部网络且不发布端口。预览失败时可下载原文件。其他文件格式的预览无需此服务。

## 7. 排查

| 现象 | 检查项 |
| --- | --- |
| 拉取失败 | 使用 Release 附件里的镜像路径。确认主机能访问 GHCR；公开版无需登录 |
| 容器持续未就绪 | 查看 `docker compose logs --tail=100 clipboard`，检查端口、数据权限及剩余磁盘空间 |
| 返回 403 | IP 访问用私有 IP 或回环地址。自定义域名经代理时核对来源设置、Host 头和端口一致 |
| 页面可以打开但内容不同步 | 检查代理转发的 `/ws` 是否返回 HTTP 101，观察浏览器是否能收到新片段 |
| 更新后看到空剪贴板 | 查看实际挂载的数据卷或路径，确认使用原 Compose 项目名和原数据源，再恢复写入 |
