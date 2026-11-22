# 项目开发指引

开始处理本项目的新任务时，先查阅[开发备忘录](docs/DEV_MEMO.md)，了解待办和待讨论事项。根据当前用户请求决定是否推进其中的内容。

安装、健康检查、数据路径和更新命令见 [部署指南](docs/deployment.md)。
从源码开发时，在仓库根目录运行 `npm ci --prefix web`，再运行 `make test vet`。
编译和测试 Go 时使用 `sqlite_fts5` 标签，Makefile 已包含该选项。
