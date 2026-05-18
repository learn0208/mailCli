# mailCli 路线图

让用户一眼看清：**已经能做什么**、**接下来做什么**。实现状态随版本更新；发版见 [CHANGELOG.md](CHANGELOG.md) 与 [GitHub Releases](https://github.com/learn0208/mailCli/releases)。

---

## 已完成（v0.1.x · EWS）

### 协议与连接

| 能力 | 说明 |
|------|------|
| Exchange EWS | `protocol: ews`（默认） |
| 认证 | Basic、NTLM、OAuth（Access Token） |
| 配置 | YAML profile、`MAILCLI_*` 环境变量（兼容 `EWS_*`） |
| 安全 | 仅 HTTPS 端点；密码推荐走环境变量 |
| 端点提示 | `mailcli discover --user you@domain.com`（静态 URL 提示，非网络 Autodiscover） |

### 命令

| 命令 | 能力 |
|------|------|
| `search` | FindItem：主题/正文/发件人/收件人、时间范围、文件夹、已读/未读、附件、limit、默认最近 N 天 |
| `send` | 纯文本/HTML、附件、抄送/密送、重要性、发件人显示名策略 |
| `send` | 发送后可选复核「已发送」文件夹（`--verify-sent-wait` / `--no-verify-sent`） |
| `show` | 按 ItemId GetItem：text / html / json |
| 输出 | 表格（人读）与 JSON（脚本/CI） |
| 排错 | `--verbose` 打印 HTTP/SOAP |

### 交付

| 能力 | 说明 |
|------|------|
| 跨平台二进制 | Windows / Linux / macOS（amd64、arm64），无 cgo |
| 开源文档 | README、中文使用说明、架构说明、示例配置 |

---

## 进行中 / 下一版

| 项 | 说明 | 状态 |
|----|------|------|
| GitHub Release 自动构建 | 打 tag `v*` 后 CI 发布多平台包 | 已配置 workflow，待首次 tag |
| 网络 Autodiscover | 根据邮箱自动解析 EWS URL | 未开始 |

---

## 计划中（多协议）

优先级为建议顺序，欢迎 Issue/PR 讨论。

### P1 — IMAP / SMTP（通用邮箱）

| 能力 | 说明 |
|------|------|
| IMAP `search` | 文件夹内检索、拉取列表 |
| IMAP `show` | 按 UID/消息 ID 读信 |
| SMTP `send` | 标准 SMTP 发信（TLS/STARTTLS） |
| 配置 | `protocol: imap` + `imap` / `smtp` 嵌套配置块 |
| 统一 CLI | 与 EWS 尽量相同的子命令与 JSON 字段（能对齐的对齐） |

### P2 — 体验与运维

| 能力 | 说明 |
|------|------|
| `list-folders` | 列出邮箱文件夹 |
| 附件下载 | `show` / 独立子命令保存附件二进制 |
| 重试与限速 | 可配置的退避与并发 |
| Shell 补全 | bash/zsh/fish completion |

### P3 — 增强

| 能力 | 说明 |
|------|------|
| EWS Autodiscover | 真实 SOAP Autodiscover |
| OAuth 设备码/刷新令牌 | 简化 Exchange Online 登录 |
| Graph API | 可选后端（与 EWS 并列） |
| 插件或脚本钩子 | 管道友好（stdin/stdout 契约） |

---

## 不参与范围（当前）

- 图形界面、Web UI
- 本地邮件长期存储/全文索引引擎
- 替代 Outlook 的完整客户端能力

---

## 如何参与

1. 在 [Issues](https://github.com/learn0208/mailCli/issues) 讨论优先级  
2. 阅读 [CONTRIBUTING.md](CONTRIBUTING.md) 与 [docs/architecture.md](docs/architecture.md)  
3. 新协议实现请放在 `internal/protocol/<name>/`，并在本文件更新状态  
