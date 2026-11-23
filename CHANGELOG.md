# Changelog

本项目遵循 [语义化版本 2.0.0](https://semver.org/lang/zh-CN/) 与
[Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/) 规范。

- 版本号以 Git **annotated tag** 为准：`vMAJOR.MINOR.PATCH`；
- 仓库根目录 `VERSION`、`web/package.json` 的 `version` 与 tag 保持同步；
- 新增变更先记录在 `[Unreleased]`，发布时移动到对应版本号下。

## [Unreleased]

<!-- 新的变更先记录在这里，发布时整体移动到下面的版本号小节。 -->

## [1.1.1]

- Release 附带独立的 Docker 初始化脚本，支持英文和简体中文。无法识别终端语言时使用英文，也可用 `--lang` 指定语言。
- 向导可选择访问范围、端口、时区、内容保留天数和 Office 预览。可以立即启动，也可以保存配置后手动启动。
- 仓库首页默认显示英文 README，两个语言版本可相互切换。

## [1.1.0]

- Docker 快速安装使用固定版本的公开镜像与命名卷。可选的 Bind Mount 配置复用群晖等主机上的原数据目录。
- CI 分别构建和验收 amd64、arm64。稳定标签生成 GitHub Release 和 Compose 下载附件。
- 源码启动和 Compose 端口默认仅绑定到本机。限制 API 与 WebSocket 的浏览器来源，自定义代理域名通过 `CLIP_PUBLIC_ORIGIN` 配置。
- 更新已知漏洞影响的 Go 工具链、网络和图像依赖以及 PDF.js。
- 增加日期导航、上传来源、文件卡片预览和界面反馈。Office 预览使用可选的内部转换容器。
- 采用 MIT 许可证，保留第三方署名，并提供快速安装、贡献和安全报告说明。

## [1.0.0] - 2026-10-22

首个正式版本：局域网剪贴板的完整初始功能集。

### Added

- 局域网跨设备剪贴板：文本 / 图片 / 文件三种内容类型，WebSocket 实时广播，
  断线自动重连（指数退避 1s → 30s）并在重连后自动补拉数据。
- 多种输入方式：输入框发送、拖拽上传、粘贴图片、文件选择、`Ctrl/⌘ + Enter` 快捷发送。
- 图片支持：超过 10 MB 的图片自动生成最大宽度 800px、质量 85 的 JPEG 缩略图；
  点击图片打开预览灯箱。
- 全文搜索：SQLite FTS5 与 LIKE 子串双通道合并，支持中文子串（如「实际」）、
  英文前缀（如 `artic` → `article`）、多关键词 AND、类型筛选与关键词高亮。
- 内容整理：置顶 / 取消置顶、单条删除、一键清空、按日期分组、无限滚动。
- 自动清理：非置顶内容默认保留 30 天，每天 03:00 清理数据库记录与磁盘文件，
  并通过 WebSocket 通知在线设备。
- 体验：深色 / 浅色 / 跟随系统主题、中英文切换、Toast 提示、复制按钮
  （含非安全上下文下的 `document.execCommand('copy')` 降级）、移动端布局适配。
- 后端：Go + Gin 单进程同时提供 REST API（12 个端点）与 WebSocket（`/ws`）及 SPA 静态页面；
  SQLite（WAL 模式）+ FTS5 外部内容索引 + 三个同步触发器；UUIDv7 主键；
  基于 `created_at` 的游标分页。
- 安全与健壮性：上传文件名清洗、上传 / 下载路径穿越防护、单文件 300 MB 与
  单条文本 300 KB 上限、优雅关闭、WebSocket 心跳与慢客户端剔除。
- 工程化：Makefile 构建 / 开发 / 测试 / 冒烟入口，项目内独立 Go 工具链引导，
  18 个 Go 测试（含 REST API、WebSocket、文件安全与搜索）以及 `tsc` 严格类型检查。
- 文档：README、技术架构文档（`docs/architecture.html`）、工程化评估报告
  （`docs/ENGINEERING.md`）与 Docker 部署指南（`docs/DOCKER.md`）。
- 容器化：三阶段 `Dockerfile`（CGO + `sqlite_fts5`、非 root、健康检查、多架构）、
  `.dockerignore` 与 `docker-compose.yml`（命名卷、资源上限、日志轮转）。

### Security

- 上传文件仅能通过 API 访问，磁盘路径不对外暴露；文件名与下载路径均做清洗与越界校验。
- 当前版本**没有鉴权与 TLS**，面向受信任的局域网使用。[局域网和 HTTPS](docs/deployment.md#3-局域网和-https)

### Notes

- 前端构建产物不嵌入 Go 二进制，部署时需要 `make static` 或设置 `CLIP_STATIC_DIR`。
- 构建必须使用 `-tags sqlite_fts5`，否则 SQLite 不包含 FTS5 模块，服务会在建表阶段失败。
