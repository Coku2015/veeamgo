# VeeamGo CLI 中文指南

[English Guide](README.md) | 中文

VeeamGo 是一个轻量级命令行工具，通过官方 REST API 帮助审计、运维和支持团队快速查看 Veeam Backup & Replication (VBR) 环境的数据，无需登录控制台界面。

## 前置条件
- 可访问的 Veeam Backup & Replication v13.x 服务器，并已启用 REST API（默认 HTTPS 端口 9419）。
- 拥有具备读取权限的 VBR 帐号或 OAuth 应用，便于执行 `veeamgo login`。
- 运行环境需为 macOS 12+/Linux (glibc 2.31+)/Windows 10+ 且 CPU 架构为 x86_64 或 arm64；如需自行编译请安装 Go 1.25+。
- 终端需具备到 VBR REST 接口的网络连通性。

## 下载
- 请前往 [GitHub Releases](https://github.com/Coku2015/veeamgo/releases/latest) 获取已签名的官方二进制文件。
- 每个版本都会提供 macOS、Linux 与 Windows 的压缩包；解压后保留 `veeamgo`（或 `veeamgo.exe`）即可使用。
- 可将二进制放入已在 `PATH` 中的目录，或将其所在目录加入 `PATH`。

```bash
# 示例：macOS / Linux
curl -LO https://github.com/Coku2015/veeamgo/releases/latest/download/veeamgo-darwin-arm64.tar.gz
tar -xzf veeamgo-darwin-arm64.tar.gz
chmod +x veeamgo
```

## 快速上手
1. 下载并解压与平台匹配的二进制。
2. （可选）将二进制移动到 `/usr/local/bin/`（macOS/Linux）或 `%PATH%` 中的目录（Windows）。
3. 登录 VBR：`veeamgo login --server vbr.example.com --username administrator --insecure`
4. 运行任意命令，例如 `veeamgo get job --limit 5` 查看作业列表。
5. 完成后退出：`veeamgo logout`，如需彻底清理请执行 `veeamgo logout --all`。

## 认证要点
- 首次执行 `login` 会提示输入密码（或读取 `VEEAMGO_PASSWORD`），并在 `~/.veeamgo/config.yaml` 中保存配置。
- CLI 会加密缓存会话，后续命令默认复用当前激活的配置，直到执行 `logout`。
- 使用 `--save-password` 可在重启后继续复用凭据，`--set-default` 可将当前配置设为默认。
- 需要清理所有缓存会话时，执行 `veeamgo logout --all`，切换实验环境前请务必执行。

## 从源码构建（可选）
如需调试或本地开发，可运行：

```bash
go build -o veeamgo ./cmd
```

在沙箱中运行时建议设置 `GOCACHE=$(pwd)/.gocache`（以及 `GOMODCACHE=$(pwd)/.gomodcache`）以复用本地缓存。

## 下一步
- 阅读 `docs/UserGuide.zh-CN.md` / `docs/UserGuide.md` 获取完整命令参考。
