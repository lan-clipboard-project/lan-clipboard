# 维护与发布

## 应用结构

Go 进程同时提供 React 静态资源、REST API 和 WebSocket。数据存储使用 SQLite WAL 和 FTS5，构建时启用 CGO 与 `sqlite_fts5` 标签。数据库、上传文件和预览缓存位于同一个本地数据目录。开发和测试命令见[贡献指南](../CONTRIBUTING.md)，运行配置与备份见[部署指南](deployment.md)。

## 镜像与版本

- GitHub Actions 在 PR、`main` push 和版本标签 push 上运行检查。两个 Linux 架构分别构建并测试，包括 FTS5、WebSocket、健康检查、非 root 用户及容器重建后的数据保留。
- `main` 验收通过后发布 `sha-<完整提交 SHA>`，用于追踪开发构建。
- 稳定版本使用 `vMAJOR.MINOR.PATCH` 标签。`VERSION`、`web/package.json` 与锁文件同步维护。对应版本的镜像通过双架构验收后，自动发布版本标签和 `latest`，并生成 GitHub Release 草稿、锁定镜像摘要的 Compose 附件和 `install.sh`。
- 工作流用 `GITHUB_TOKEN` 发布刚通过验收的构建。镜像发布任务有 Packages 写权限，创建 Release 的任务有 Contents 写权限。

## 首次开源发布的检查

1. 用确认的公开项目账号或组织准备独立仓库，提交经过清理的源码和截图。Git 提交使用该公开身份的 GitHub `noreply` 邮箱。
2. 检查已提交的文件、提交元数据、许可证和示例数据。启用 GitHub 私密漏洞报告。
3. 推送一个与 `VERSION` 一致的版本标签，确认两个架构的验收、镜像发布和 Release 附件均已完成。
4. 组织的 **Settings → Packages → Package Creation** 需要允许公开包。首次创建 GHCR 包后，在包的 **Package settings** 确认要公开的版本，选择 Public。公开容器镜像可匿名拉取。[GitHub Packages 可见性](https://docs.github.com/en/packages/learn-github-packages/configuring-a-packages-access-control-and-visibility)
5. 用未登录 GHCR 的 Docker 环境下载待发布版本的附件，照[手动部署](deployment.md#手动部署)启动。检查页面、双窗口同步、健康状态和更新后数据持久性。发布后可用[安装脚本](README.zh-CN.md#docker-快速开始)验证该版本的下载和启动。

部署的版本由各实例管理员决定。版本发布可能涉及数据库升级，发布说明需要注明备份和兼容性条件。
