# 参与贡献

部署和问题排查见[部署指南](docs/deployment.md)。报告缺陷时说明应用版本、主机架构、部署方式，并提供已移除私有信息的复现步骤。不要提交剪贴板数据、上传文件、私有地址或凭据。安全问题请按[安全策略](SECURITY.md)报告。

安装 Go、Node.js、C 编译器和 Make 后，从仓库根目录运行：

```sh
npm ci --prefix web
make test vet
make smoke
```

本地开发可分别运行 `make dev-server` 和 `make dev-web`，然后打开 Vite 的本机地址。Go 服务依赖 CGO 和 `sqlite_fts5`，Makefile 已包含构建标签。涉及数据迁移的变更应覆盖已有 SQLite 数据库的升级。
