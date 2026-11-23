# 参与贡献

部署和问题排查见[部署指南](docs/deployment.md)。报告缺陷时说明应用版本、主机架构、部署方式，并提供已移除私有信息的复现步骤。不要提交剪贴板数据、上传文件、私有地址或凭据。安全问题请按[安全策略](SECURITY.md)报告。

安装 Go 1.27.4 或更新版本、Node.js 22.13 或更新版本、C 编译器和 Make。在仓库根目录安装前端依赖并启动服务。

```sh
npm ci --prefix web
make run
```

在浏览器打开 [http://127.0.0.1:8080](http://127.0.0.1:8080)。

提交前运行以下检查。

```sh
make test vet
make smoke
```

本地开发可分别运行 `make dev-server` 和 `make dev-web`，然后打开 Vite 的本机地址。Go 服务依赖 CGO 和 `sqlite_fts5`，Makefile 已包含构建标签。涉及数据迁移的变更应覆盖已有 SQLite 数据库的升级。
