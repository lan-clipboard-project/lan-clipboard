# syntax=docker/dockerfile:1
#
# LAN Clipboard — 生产用多阶段镜像
#
# DS224+ 目标构建（在 amd64 构建机上执行；其他架构需要 QEMU）：
#     docker buildx build --platform linux/amd64 --load -t lan-clipboard:ci .
#
# 多架构构建（需要 buildx + QEMU；CGO 不允许交叉编译，只能在目标平台内模拟编译）：
#     docker buildx build --platform linux/amd64,linux/arm64 \
#         -t <registry>/lan-clipboard:1.0.0 --push .
#
# 部署、验证清单与故障排查见 docs/deployment.md。
#
# 三条硬约束：
#   1. mattn/go-sqlite3 是 CGO 扩展；CGO_ENABLED 必须为 1，且编译环境要有 gcc。
#   2. FTS5 只能在编译期通过 -tags sqlite_fts5 打开；运行期无法补救。
#   3. 绝不能为了 arm64 设置 GOARCH 交叉编译（CGO 会链接错误的 libc）；
#      正确做法是用 buildx 在 linux/arm64 平台内执行“原生”编译（QEMU 模拟或真实 arm64 机器）。
#
# 使用 Docker Official Images 的精确版本 tag。
# Go 与 CI 使用同一版本；升级时同步 workflow。可通过 --build-arg 覆盖镜像引用，
# 或使用 @sha256:<digest> 固定基础镜像内容。
ARG NODE_IMAGE=node:22.23.4-alpine
ARG GO_IMAGE=golang:1.27.4-bookworm
ARG RUNTIME_IMAGE=debian:bookworm-20261020-slim

# ─────────────────────────────────────────────────────────────────────────────
# 阶段 1/3：前端构建  web/  ──npm ci + vite build──▶  /src/web/dist
#
# 前端产物与 CPU 架构无关，因此固定在 BUILDPLATFORM（构建机原生架构）上只构建
# 一次：多架构构建时不会为每个目标架构在 QEMU 里重复跑 npm，能省下大量时间。
# ─────────────────────────────────────────────────────────────────────────────
FROM --platform=$BUILDPLATFORM ${NODE_IMAGE} AS web-build

# 可通过 --build-arg NPM_REGISTRY=... 调整 npm 源。
ARG NPM_REGISTRY=https://registry.npmjs.org/

WORKDIR /src/web

# 先只复制依赖清单，让依赖层缓存不被源码改动破坏。
COPY web/package.json web/package-lock.json ./

# npm ci 严格按 package-lock.json 安装；devDependencies 是必需的
# （package.json 的 build 脚本先跑 tsc 类型检查，再跑 vite build）。
RUN --mount=type=cache,target=/root/.npm \
    npm ci --no-audit --no-fund --registry=${NPM_REGISTRY}

# 再复制源码；依赖目录已在 .dockerignore 中排除，不会覆盖上面的 node_modules。
COPY web/ ./
COPY scripts/copy-license-files.sh /tmp/copy-license-files.sh

RUN npm run build \
 && npm ls --omit=dev --all --parseable > /tmp/npm-packages \
 && awk -v prefix="$PWD/node_modules/" 'index($0, prefix) == 1 {print substr($0, length(prefix) + 1) "\t" $0}' \
      /tmp/npm-packages > /tmp/npm-license-paths \
 && sh /tmp/copy-license-files.sh /out/licenses < /tmp/npm-license-paths

# ─────────────────────────────────────────────────────────────────────────────
# 阶段 2/3：Go 编译  server/ ──CGO + FTS5──▶  /out/lan-clipboard
#
# 该阶段跟随目标平台（buildx --platform），因此 Go 与其调用的 gcc 都是目标架构，
# 这是 CGO 唯一正确的跨架构方式。
# ─────────────────────────────────────────────────────────────────────────────
FROM ${GO_IMAGE} AS go-build

# 可通过 --build-arg GOPROXY=... 调整 Go 模块源。
ARG GOPROXY=https://proxy.golang.org,direct

# CGO_ENABLED=1：go-sqlite3 必需。
# GOTOOLCHAIN=local：禁止 go 命令偷偷下载其它工具链，构建必须使用本镜像内的 Go。
# GOFLAGS=-mod=readonly：模块依赖必须与 go.mod/go.sum 完全一致，不自动改写。
ENV CGO_ENABLED=1 \
    GOTOOLCHAIN=local \
    GOFLAGS=-mod=readonly \
    GOPROXY=${GOPROXY}

WORKDIR /src/server

# 依赖层单独缓存：只有 go.mod/go.sum 变化时才重新下载。
COPY server/go.mod server/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY server/ ./
COPY scripts/copy-license-files.sh /tmp/copy-license-files.sh

# -tags sqlite_fts5 ：编译 go-sqlite3 内置 SQLite 时开启 FTS5（全文搜索必需）。
# -trimpath          ：去掉构建机绝对路径，产物可复现。
# -buildvcs=false    ：构建上下文没有 .git（.dockerignore 已排除），避免 VCS 探测。
# -ldflags "-s -w"   ：剥离符号表/调试信息，通常可让二进制小 4–6MB。
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -tags sqlite_fts5 -trimpath -buildvcs=false -ldflags="-s -w" -o /out/lan-clipboard . \
 && go list -deps -tags sqlite_fts5 -f '{{with .Module}}{{if not .Main}}{{printf "%s\t%s" .Path .Dir}}{{end}}{{end}}' > /tmp/go-packages \
 && sort -u /tmp/go-packages > /tmp/go-license-paths \
 && sh /tmp/copy-license-files.sh /out/licenses/modules < /tmp/go-license-paths \
 && cp "$(go env GOROOT)/LICENSE" /out/licenses/Go-LICENSE

# ─────────────────────────────────────────────────────────────────────────────
# 阶段 3/3：运行时  debian:bookworm-slim
#   /app/lan-clipboard   Go 二进制（root 拥有、只读可执行）
#   /app/static/         前端构建产物（root 拥有、只读）
#   /app/data/           数据卷：clipboard.db + uploads/（UID/GID 10001 可写）
# ─────────────────────────────────────────────────────────────────────────────
FROM ${RUNTIME_IMAGE} AS runtime

ARG APP_VERSION=dev
ARG APP_REVISION=unknown
ARG APP_SOURCE
LABEL org.opencontainers.image.title="LAN Clipboard" \
      org.opencontainers.image.description="LAN clipboard: Go REST/WebSocket + SPA + SQLite FTS5" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.version="${APP_VERSION}" \
      org.opencontainers.image.revision="${APP_REVISION}" \
      org.opencontainers.image.source="${APP_SOURCE}"

# 运行时依赖（最小集合）：
#   ca-certificates  常规 CA 信任
#   curl             HEALTHCHECK 与排障
#   tzdata           让 TZ 生效（03:00 保留期清理跟随进程本地时区）
#   sqlite3          在线 WAL 安全备份（.backup）与 PRAGMA integrity_check，见 docs/deployment.md
# 说明：基础镜像已按日期 tag 固定；apt 包故意不锁版本，以便随 Debian 安全更新一起升级。
# hadolint ignore=DL3008
RUN apt-get update \
 && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
        ca-certificates curl sqlite3 tzdata \
 && rm -rf /var/lib/apt/lists/*

# 固定 UID/GID（10001）。镜像、命名卷、绑定挂载三者的权限因此可预期：
# 命名卷首次挂载时会继承镜像中 /app/data 的属主；绑定挂载需要宿主机 chown 10001:10001。
RUN groupadd --gid 10001 app \
 && useradd --uid 10001 --gid 10001 --no-create-home --shell /usr/sbin/nologin app \
 && mkdir -p /app/data \
 && chown 10001:10001 /app/data \
 && chmod 0755 /app/data

WORKDIR /app

# 二进制与静态资源归 root、全员可读：运行用户即使被攻破也无法改写自己的
# 可执行文件或前端资源（容器内最小化不可变面）。
COPY --from=go-build --chmod=0555 /out/lan-clipboard /app/lan-clipboard
COPY --from=web-build /src/web/dist /app/static
COPY LICENSE THIRD_PARTY_NOTICES.md /usr/share/licenses/lan-clipboard/
COPY --from=go-build /out/licenses /usr/share/licenses/lan-clipboard/go
COPY --from=web-build /out/licenses /usr/share/licenses/lan-clipboard/npm

# 构建期自检：SPA 入口缺失或二进制不可执行时立刻失败，而不是等容器启动才报错。
RUN test -x /app/lan-clipboard && test -s /app/static/index.html

# 默认值逐项与 server/config/config.go 保持一致；数据/静态目录按镜像布局固定。
# 覆盖 CLIP_PORT 时记得同步 compose 的端口映射（容器内默认 8080）。
ENV TZ=Asia/Shanghai \
    GIN_MODE=release \
    CLIP_LISTEN_HOST=0.0.0.0 \
    CLIP_PORT=8080 \
    CLIP_DATA_DIR=/app/data \
    CLIP_STATIC_DIR=/app/static \
    CLIP_MAX_FILE_SIZE=300 \
    CLIP_MAX_TEXT_SIZE=307200 \
    CLIP_RETENTION_DAYS=30 \
    CLIP_THUMBNAIL_THRESHOLD=10

# SQLite（WAL）+ uploads/。同一数据目录只能被一个容器使用；不要放在 NFS/SMB 上。
VOLUME ["/app/data"]

EXPOSE 8080

# /api/stats 会真实查询 SQLite（COUNT/SUM），比 GET / 更能代表“服务可用”。
# 这里必须用 shell 形式，${CLIP_PORT} 才会在容器运行时展开（JSON 形式不会展开）。
# hadolint ignore=DL3025
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -fsS --max-time 4 "http://127.0.0.1:${CLIP_PORT}/api/stats" >/dev/null || exit 1

# Go 进程自己处理 SIGTERM 并做最多 10s 的优雅关闭（server/main.go），不需要额外 init。
STOPSIGNAL SIGTERM

USER 10001:10001

ENTRYPOINT ["/app/lan-clipboard"]
