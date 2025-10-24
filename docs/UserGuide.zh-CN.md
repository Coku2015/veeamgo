# VeeamGo 用户指南

[English Guide](UserGuide.md) | 中文

## 使用范围
本指南介绍如何在 Veeam Backup & Replication (VBR) 服务器上登录并使用 VeeamGo CLI，覆盖日常巡检、基础设施管理以及内置的安全分析工具。所有命令默认假设您已经成功登录并拥有所需权限；部分 `add`、`edit`、`delete`、`start` 类命令会修改配置，请在生产环境中谨慎操作。

## 1. 基础用法

### 1.1 动词优先语法
CLI 命令遵循 `veeamgo <动词> <资源> [子命令] [参数]` 的模式，例如：
```bash
veeamgo get job --limit 10
veeamgo describe repository --name "Default Backup Repository"
veeamgo start job --name "Daily Backup"
```

### 1.2 输出格式
默认输出为表格 (`--output table`)，适合终端阅读。指定 `--output json` 可返回 REST API 原始载荷，方便配合 `jq` 等工具处理。

### 1.3 全局参数
| 参数 | 说明 |
| --- | --- |
| `--config <path>` | 指定配置文件路径（默认 `~/.veeamgo/config.yaml`）。 |
| `--profile <name>` | 切换或创建配置文件；默认使用当前激活的 profile。 |
| `--output table|json` | 控制单次调用的输出格式。 |
| `--api-version <rev>` | 强制使用特定 REST API 版本，跳过自动协商。 |
| `--help` | 在任意命令上查看完整帮助与示例。 |

### 1.4 过滤、排序与选择器
- 名称类参数支持 `*` 通配符（示例：`--name *prod*`）。
- 大多数列表命令都提供 `--limit`、`--sort`、`--desc` 进行分页与排序。
- 时间过滤器（如 `--created-since`、`--detected-before`）使用 RFC3339 格式，例如 `2024-05-01T00:00:00Z`。
- 常见资源同时支持 `--name` 与 `--id`；建议仅指定其一以避免歧义。
- 部分参数可重复传入（如 `--type`、`--gateway-id`、`--disk`）以匹配多个值。
- 蓝图类命令（`veeamgo add|edit job`、`veeamgo add objectrepository` 等）可通过 `--set 键=值` 覆盖配置，无需手动编辑 JSON。

### 1.5 会话与安全选项
长耗时操作会返回 Session ID。加入 `--wait` 可等待任务完成，否则可稍后使用 `veeamgo get session logs --id <uuid>` 查看进度。需要确认的命令默认交互提示，使用 `--yes` 可在脚本中跳过；若命令打印出 Session ID，重复执行也不会创建重复资源。

## 2. 认证与配置文件

### 2.1 登录
使用 `veeamgo login` 认证。核心参数如下：

| 参数 | 说明 |
| --- | --- |
| `--server <host|url>` | REST API 基地址或主机名（强制 HTTPS，默认端口 9419）。 |
| `--port <n>` | 在仅提供主机名时覆盖 REST 端口。 |
| `--username <user>` | VBR 登录账号。 |
| `--password <pass>` | 可选的内联密码，留空则终端安全输入。 |
| `--scope <scopes>` | 若服务器启用 OAuth，可指定额外 scope。 |
| `--insecure` | 跳过 TLS 证书校验（仅限实验环境）。 |
| `--save-password` | 将密码以加密形式保存到配置文件，便于静默登录。 |
| `--set-default` | 登录成功后将当前 profile 设为默认。 |

环境变量：
- `VEEAMGO_PASSWORD`：提供密码实现全自动登录。
- `VEEAMGO_API_VERSION`：在登录完成后强制使用指定 API 版本。

登录成功后会生成 `~/.veeamgo/config.yaml`，并在 `~/.veeamgo/sessions.json` 中缓存会话令牌。

### 2.2 登出
- `veeamgo logout`：清理当前 profile 的缓存会话。
- `veeamgo logout --all`：清空所有 profile 的缓存会话（建议切换实验环境或共享机器前执行）。

### 2.3 配置文件存储与切换
配置文件以名称区分，可通过 `--profile qa` 等参数在任一命令上切换。如 profile 不存在，首次登录会自动创建。手动编辑 YAML 时请保留文件权限（目录 `0700`、文件 `0600`），避免敏感信息泄漏。

## 3. 命令参考

除非特别说明，以下命令均支持前文提到的全局参数。示例默认使用表格输出。

### 3.1 服务器与平台配置
| 命令 | 用途 | 关键参数 |
| --- | --- | --- |
| `veeamgo get server info` | 查看服务器平台、版本、数据库与补丁信息。 | 无 |
| `veeamgo get server time` | 获取服务器时间与时区。 | 无 |
| `veeamgo get generaloption` | 查看全局通知、SIEM、阈值设置。 | 无 |
| `veeamgo describe configurationbackup` | 查看配置备份计划与最近执行信息。 | 无 |
| `veeamgo start configurationbackup [--yes] [--wait]` | 立即触发配置备份，可选等待任务结束。 | `--yes`、`--wait` |
| `veeamgo license summary` | 汇总许可证持有人、类型与到期时间。 | 无 |
| `veeamgo license sockets [--limit N]` | 查看基于 socket 的使用情况。 | `--limit` |
| `veeamgo license instances [--limit N]` | 查看实例额度占用与可撤销状态。 | `--limit` |
| `veeamgo license capacity [--limit N]` | 查看容量层使用明细。 | `--limit` |

### 3.2 基础设施与清单

#### 托管服务器
| 命令 | 用途 | 关键参数 |
| --- | --- | --- |
| `veeamgo get managedserver [--type WindowsHost] [--name *prod*] [--limit 50]` | 按类型/名称筛选托管服务器。 | `--type`、`--name`、`--limit` |
| `veeamgo describe managedserver --id <uuid>` | 查看单个托管服务器详情。 | `--id` |
| `veeamgo add managedserver vsphere --name vc01.lab --username svc --password ***** [--thumbprint sha1] [--wait] [--yes]` | 注册 vCenter/ESXi。可用 `--credentials-id` 复用凭据、`--port` 自定义端口。 | `--name`、凭据相关参数、`--port`、`--thumbprint`、`--wait`、`--yes` |
| `veeamgo add managedserver windows --name repo01.lab --connect-mode Credential|Certificate --username admin ...` | 新增 Windows 托管服务器，支持证书模式。 | `--name`、`--connect-mode`、凭据参数、`--wait`、`--yes` |
| `veeamgo add managedserver linux --name repo01.lab --connect-mode Credential|SingleUse|Certificate ...` | 新增 Linux 托管服务器，支持自动抓取指纹与单次凭据 (`--single-use-*`)。 | `--name`、`--connect-mode`、凭据参数、`--ssh-fingerprint`、`--handshake-code`、`--single-use-*`、`--wait`、`--yes` |
| `veeamgo rescan managedserver [--all | --id <uuid> | --name host] [--wait]` | 刷新托管服务器状态。 | `--all`、`--id`、`--name`、`--wait` |
| `veeamgo delete managedserver --id <uuid> [--wait] [--yes]` | 删除托管服务器（也可用 `--name`）。 | `--id`/`--name`、`--wait`、`--yes` |

#### 备份仓库
| 命令 | 用途 | 关键参数 |
| --- | --- | --- |
| `veeamgo get repository [--type WinLocal] [--name *repo*] [--limit 20]` | 查看仓库列表与容量指标。 | `--type`、`--name`、`--limit` |
| `veeamgo describe repository --name "Default Backup Repository"` | 查看仓库配置与 YAML。 | `--name` |
| `veeamgo add repository winlocal --host repo01.lab --path D:\Backups --mount-server repo01.lab [--max-tasks 4] [--wait] [--yes]` | 新增 Windows 仓库。 | `--host`、`--path`、`--mount-server`、`--max-tasks`、`--read-write-limit`、`--import-backup`、`--import-index`、`--disable`、`--wait`、`--yes` |
| `veeamgo add repository linuxlocal --host repo01.lab --path /backups ...` | 新增 Linux 仓库。 | 参数同 Windows 仓库 |
| `veeamgo add repository linuxhardened --host repo01.lab --path /immutable --immutable-days 14 [--fast-clone] ...` | 新增 Hardened 仓库，支持不可变策略。 | `--immutable-days`、`--fast-clone` 及通用参数 |
| `veeamgo add repository smb|nfs --share-path //<server>/<share> --credentials-id <uuid> --mount-server repo01.lab [--gateway-server id] ...` | 新增 SMB/NFS 仓库，可控制网关。 | `--share-path`、`--credentials-id`、`--gateway-auto`、`--gateway-server`、`--mount-server` 等 |
| `veeamgo edit repository --name Repo1 [--new-name Repo2] [--max-tasks 8] ...` | 修改仓库配置，无需手写 JSON。 | `--name`/`--id`、`--type`、性能参数、`--wait`、`--yes` |
| `veeamgo rescan repository [--name Repo1 | --id <uuid> | --all] [--wait]` | 对仓库执行重新扫描。 | `--all`、`--name`、`--id`、`--wait` |
| `veeamgo delete repository --name Repo1 [--wait] [--yes]` | 删除仓库。 | `--name`/`--id`、`--wait`、`--yes` |

#### 对象存储仓库
| 命令 | 用途 | 关键参数 |
| --- | --- | --- |
| `veeamgo get objectrepository [--type AmazonS3] [--name *vault*] [--limit 20]` | 查看对象存储仓库。 | `--type`、`--name`、`--limit` |
| `veeamgo describe objectrepository --name "S3 Offsite"` | 查看对象仓库配置与分层策略。 | `--name` |
| `veeamgo add objectrepository amazon-s3 ...` | 快速创建 Amazon S3 仓库脚手架。 | `--name`、`--description`、`--credentials-id`、`--bucket-name`、`--folder`、`--region-id`、`--region-scope`、`--connection-type`、`--gateway-id`、`--mount-server-id`、`--mount-server-cache`、`--mount-server-vpower`、`--immutability-enabled`、`--immutability-days`、`--spec`、`--set`、`--disable`、`--wait`、`--yes` |
| `veeamgo add objectrepository s3-compatible|wasabi-cloud|azure-blob|azure-archive|veeam-data-cloud-vault ...` | 针对不同云厂商的快捷命令，参数随厂商变化。 | 查看各子命令 `--help` |
| `veeamgo add objectrepository types` | 列出所有受支持的对象存储类型与脚手架情况。 | 无 |
| `veeamgo edit objectrepository --name "S3 Offsite" [--type AmazonS3] [--set account.bucketName=new-bucket] ...` | 修改已存在的对象仓库配置。 | `--name`/`--id`、`--type`、`--set`、`--spec`、`--disable`、`--wait`、`--yes` |
| `veeamgo rescan objectrepository [--name Repo | --id <uuid> | --all] [--wait]` | 对象仓库重新扫描校验访问。 | `--name`/`--id`、`--all`、`--wait` |
| `veeamgo delete objectrepository --name "S3 Offsite" [--wait] [--yes]` | 删除对象仓库。 | `--name`/`--id`、`--wait`、`--yes` |

#### 分层仓库（SOBR）
| 命令 | 用途 | 关键参数 |
| --- | --- | --- |
| `veeamgo get sobr [--name *prod*] [--limit 20]` | 查看 Scale-out 仓库及放置策略。 | `--name`、`--limit` |
| `veeamgo describe sobr --name "SOBR01"` | 查看性能层、容量层、归档层以及 YAML 配置。 | `--name` |

#### WAN 加速器
| 命令 | 用途 | 关键参数 |
| --- | --- | --- |
| `veeamgo get wanaccelerator [--name *source*] [--limit 20]` | 查看 WAN 加速器列表及缓存设置。 | `--name`、`--limit` |
| `veeamgo describe wanaccelerator --name "WAN-A"` | 查看单个加速器的缓存路径、端口等配置。 | `--name` |

#### 代理 (Proxy)
| 命令 | 用途 | 关键参数 |
| --- | --- | --- |
| `veeamgo get proxy list [--type vmware] [--name *proxy*] [--sort name] [--limit 20]` | 查看代理类型、状态与最大并发。 | `--type`、`--name`、`--sort`、`--desc`、`--limit` |
| `veeamgo describe proxy --name Proxy01 [--type vmware]` | 查看代理详细配置与传输模式。 | `--name`、`--type` |
| `veeamgo add proxy --managed-server host01.lab --type vmware --transport-mode auto [--max-tasks 4] [--yes] [--wait]` | 基于已有托管服务器创建代理。 | `--managed-server`、`--type`、`--max-tasks`、VMware 专用参数、`--wait`、`--yes` |
| `veeamgo edit proxy --name Proxy01 [--new-name Proxy02] [--managed-server host02] ...` | 更新代理名称、主机、吞吐限制等。 | `--name`/`--id`、调优参数、`--wait`、`--yes` |
| `veeamgo delete proxy --name Proxy01 [--wait] [--yes]` | 删除代理。 | `--name`/`--id`、`--wait`、`--yes` |
| `veeamgo enable proxy --name Proxy01 [--yes] [--wait]` | 启用已禁用的代理。 | `--name`/`--id`、`--yes`、`--wait` |
| `veeamgo disable proxy --name Proxy01 [--yes] [--wait]` | 禁用代理但保留配置。 | `--name`/`--id`、`--yes`、`--wait` |

#### 流量规则与全局排除
| 命令 | 用途 | 关键参数 |
| --- | --- | --- |
| `veeamgo get trafficrule` | 查看流量限速/加密规则。 | 无 |
| `veeamgo get exclusionvm [--sort id] [--limit 50]` | 列出全局排除的虚拟机。 | `--sort`、`--desc`、`--limit` |
| `veeamgo describe exclusionvm <id>` | 查看指定排除项详情。 | 位置参数 `id` |

#### 清单浏览
| 命令 | 用途 | 关键参数 |
| --- | --- | --- |
| `veeamgo get inventory virtualinfra [--platform vsphere] [--type VCenterServer] [--name *prod*] [--limit 200]` | 列出虚拟化基础设施（vSphere、Hyper-V、VCD）。 | `--platform`、`--type`、`--name`、`--host`、`--limit`、`--sort`、`--desc` |
| `veeamgo get inventory virtualinfra objects --name vc01.lab [--hierarchy HostsAndClusters] [--object-type VirtualMachine]` | 查看指定服务器下的对象树。 | `--name`、`--hierarchy`、`--object-type`、`--object-name`、`--limit`、`--sort`、`--desc` |
| `veeamgo describe inventory virtualinfra --name vc01.lab` | 查看单个虚拟化服务器的元数据（URN、ID）。 | `--name`、`--limit`、`--sort`、`--desc` |
| `veeamgo get inventory unstructured [--type NasServer] [--limit 200]` | 查看 NAS / 非结构化清单。 | `--type`、`--name`、`--limit`、`--sort`、`--desc` |
| `veeamgo describe inventory unstructured --id <uuid>` | 查看指定非结构化对象详情。 | `--id` |
| `veeamgo get inventory protectiongroup [--type Agent] [--name *prod*] [--limit 200]` | 查看保护组及其范围。 | `--type`、`--name`、`--limit`、`--sort`、`--desc` |
| `veeamgo describe inventory protectiongroup --id <uuid>` | 查看保护组配置。 | `--id` |

#### 指纹查询
| 命令 | 用途 | 关键参数 |
| --- | --- | --- |
| `veeamgo get fingerprint --server host --type LinuxHost [--credentials-storage Certificate] [--handshake-code 123456] [--port 22]` | 在添加托管服务器前获取连接指纹或证书。 | `--server`、`--type`、`--credentials-storage`、`--credentials`/`--handshake-code`、`--port` |

### 3.3 作业、备份与恢复数据

#### 作业 (Jobs)
| 命令 | 用途 | 关键参数 |
| --- | --- | --- |
| `veeamgo get job [--type Backup] [--status Running] [--name *sql*] [--limit 50]` | 查看作业状态、下次执行时间及目标仓库。 | `--type`、`--status`、`--result`、`--workload`、`--repository`、`--high-priority`、`--limit` |
| `veeamgo get job history --name "Daily Backup" [--states Running] [--results Failed] [--limit 100]` | 查看指定作业的历史 Session。 | `--name`、`--type`、`--states`、`--results`、`--created-since`、`--created-before`、`--ended-since`、`--ended-before`、`--sort`、`--desc`、`--limit` |
| `veeamgo describe job --name "Daily Backup"` | 查看作业实时配置及 YAML。 | `--name` |
| `veeamgo add job --from path/to/job.yaml [--set spec.schedule.retry=3] [--yes] [--dry-run]` | 基于 YAML/JSON 蓝图或内置模板创建作业。 | `--from`、`--template`、`--template-variant`、`--set`、`--dry-run`、`--yes` |
| `veeamgo edit job --name "Daily Backup" [--from updated.yaml] [--from-live] [--set spec.description="Refreshed"] [--yes]` | 更新作业配置，可从现有作业导出蓝图。 | `--name`/`--id`、`--from`、`--from-live`、`--template`、`--template-variant`、`--set`、`--dry-run`、`--yes` |
| `veeamgo delete job --name "Old Job" [--yes] [--wait]` | 删除作业。 | `--name`/`--id`、`--yes`、`--wait` |
| `veeamgo enable job --name "Daily Backup" [--yes] [--wait]` | 启用作业。 | `--name`/`--id`、`--yes`、`--wait` |
| `veeamgo disable job --name "Daily Backup" [--yes] [--wait]` | 禁用作业。 | `--name`/`--id`、`--yes`、`--wait` |
| `veeamgo start job --name "Daily Backup" [--active-full] [--start-chained]` | 立即启动作业，可选择执行 Active Full 或串联作业。 | `--name`/`--id`、`--active-full`、`--start-chained`、`--sync-restore-points`、`--yes` |
| `veeamgo stop job --name "Daily Backup" [--graceful=false] [--cancel-chained]` | 停止运行中的作业。 | `--name`/`--id`、`--graceful`、`--cancel-chained`、`--yes` |
| `veeamgo retry job --name "Daily Backup" [--start-chained]` | 重试失败或警告状态的作业。 | `--name`/`--id`、`--start-chained`、`--yes` |
| `veeamgo start job quickbackup --vm-name "VM01" [--name JobName]` | 对指定虚拟机执行 Quick Backup，自动匹配可用作业。 | `--vm-name`、`--name`/`--id`、`--yes` |
| `veeamgo clone job --name "Template Job" [--yes]` | 复制现有作业生成新蓝图。 | `--name`/`--id`、`--yes` |
| `veeamgo template job --list` | 列出内置作业模板及可用变体。 | 无 |
| `veeamgo template job --type VSphereBackup --variant full [--write path] [--no-comments]` | 将模板输出到终端或文件。 | `--type`、`--variant`、`--write`、`--no-comments` |

#### 备份与存储
| 命令 | 用途 | 关键参数 |
| --- | --- | --- |
| `veeamgo get backup list [--backup *daily*] [--name "Daily Backup"] [--job-id <uuid>] [--limit 50]` | 查看备份集及目标仓库。 | `--backup`、`--name`、`--job-id`、`--job-type`、`--created-since`、`--created-before`、`--sort`、`--desc`、`--limit` |
| `veeamgo get backup files --name "Daily Backup" [--file *.vbk] [--gfs-period Weekly]` | 查看备份文件（VBK/VIB 等）。 | `--name`/`--backup`、`--file`、`--gfs-period`、`--created-since`、`--created-before`、`--sort`、`--desc`、`--limit` |
| `veeamgo get backup objects --name "Daily Backup"` | 查看备份内包含的对象（虚拟机、代理等）。 | `--name`/`--backup` |

#### 还原点与副本
| 命令 | 用途 | 关键参数 |
| --- | --- | --- |
| `veeamgo get restorepoint --name "Daily Backup" [--backup BackupCopy] [--restorepoint VM01*]` | 列出备份的还原点。 | `--name`/`--backup`、`--restorepoint`、`--object`、`--platform`、`--malware`、`--created-since`、`--created-before`、`--sort`、`--desc`、`--limit` |
| `veeamgo get replica --name "Replica Job" [--replica vm01] [--replica-point point01]` | 查看复制作业的还原点。 | `--name`（必填）、`--replica`、`--replica-point`、`--platform`、`--malware`、`--created-since`、`--created-before`、`--sort`、`--desc`、`--limit` |

#### 发布磁盘（数据集成）
| 命令 | 用途 | 关键参数 |
| --- | --- | --- |
| `veeamgo get publisheddisk --id <mount-id>` | 查看指定发布磁盘的详情。 | `--id` |
| `veeamgo get publisheddisk --all [--limit 20]` | 列出所有正在发布的磁盘。 | `--all`、`--limit` |
| `veeamgo start publishdisk --restore-point-id <uuid> --allowed-ip 10.0.0.10 [--disk disk1.vmdk] [--wait]` | 以 iSCSI 方式发布还原点中的磁盘，至少需指定一个 `--allowed-ip`。 | `--restore-point-id`、`--disk`、`--allowed-ip`、`--wait` |
| `veeamgo stop publishdisk --id <mount-id> [--wait]` | 终止一个或多个发布磁盘会话，`--id` 可重复传入。 | `--id`、`--wait` |

### 3.4 会话、任务与监控
| 命令 | 用途 | 关键参数 |
| --- | --- | --- |
| `veeamgo get session list [--type Backup] [--name *Daily*] [--result Failed]` | 查看作业/任务的 Session 列表。 | `--name`、`--job`、`--type`、`--state`、`--result`、`--created-since`、`--created-before`、`--ended-since`、`--ended-before`、`--sort`、`--desc`、`--limit` |
| `veeamgo describe session --id <uuid>` | 查看单个 Session 的详细信息。 | `--id` |
| `veeamgo get session logs --id <uuid> [--status Warning]` | 获取 Session 日志记录，可按状态过滤。 | `--id`、`--status` |
| `veeamgo get task list [--session-id <uuid>] [--type Task]` | 查看 Session 下的底层任务。 | `--session-id`、`--type`、`--state`、`--result`、`--limit`、`--sort`、`--desc` |
| `veeamgo describe task --id <uuid>` | 查看单个任务详情。 | `--id` |
| `veeamgo get task logs --id <uuid>` | 获取任务日志。 | `--id`、`--status` |

### 3.5 安全与威胁检测
| 命令 | 用途 | 关键参数 |
| --- | --- | --- |
| `veeamgo get securityanalyzer results` | 查看最近一次安全与合规分析结果。 | 无 |
| `veeamgo describe securityanalyzer schedule` | 查看分析任务的调度与通知配置。 | 无 |
| `veeamgo start securityanalyzer [--yes] [--wait]` | 立即运行分析器，可选择等待并输出结果。 | `--yes`、`--wait` |
| `veeamgo get malwaredetectionevent [--type YaraScan] [--severity Infected] [--detected-since 2024-01-01T00:00:00Z]` | 查看可疑活动/恶意事件。 | `--type`、`--state`、`--source`、`--severity`、`--created-by`、`--engine`、`--machine-name`、`--backup-object`、`--detected-since`、`--detected-before`、`--sort`、`--desc`、`--limit` |
| `veeamgo get yararule` | 查看服务器上可用的 YARA 规则包。 | 无 |

### 3.6 预留动词
`veeamgo migrate` 为预留动词，目前暂无子命令。

## 4. 故障排查与建议
- 使用 `veeamgo get session logs --id <uuid>` 或 `veeamgo get task logs --id <uuid>` 排查失败原因。
- 需要精确字段时请切换 `--output json`，便于与自动化脚本对接。
- 登录失败时确认 URL 是否为 HTTPS、证书是否可信；`--insecure` 仅在实验环境下使用。
- 添加新 profile 后执行 `veeamgo login --set-default`，后续命令将默认使用该配置。
- 在交接电脑或切换环境前运行 `veeamgo logout --all` 清理缓存令牌。

## 5. 参考资料
- `README.md` 包含下载、快速上手与发行说明。
- `develop_docs/` 目录存放历史设计文档，已加入 `.gitignore`。
- 任何命令都可以通过 `veeamgo <...> --help` 获取最新、最完整的参数说明；新版本新增参数时请以帮助输出为准。
