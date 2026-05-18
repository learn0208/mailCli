# 邮件自动化 CLI 工具 PRD（EWS 专版）

> **历史文档**：撰写时产品名为 `ews-cli`。当前实现为 **mailCli**，命令行二进制为 **`mailcli`**，配置默认 `~/.mailcli.yaml`。以 [README.md](../../README.md) 与 [docs/使用说明.md](../使用说明.md) 为准。

| 项目     | 内容       |
| -------- | ---------- |
| 版本     | 1.0        |
| 日期     | 2026-05-14 |
| 状态     | 草案       |

---

## 1. 项目背景与目标

### 1.1 背景

在企业环境中，Microsoft Exchange Server 是部署最广泛的邮件系统之一。运维与开发常需从海量邮件中检索信息（监控告警、周报、审批），或通过脚本发信（CI/CD 通知、自动化报告）。图形客户端与 Web 界面难以嵌入自动化流水线，也缺少程序化批量处理能力。

### 1.2 目标

开发一款面向 Microsoft Exchange 的 **CLI 工具**，通过 **Exchange Web Services (EWS)** 实现：

- **邮件搜索**：服务端过滤，快速检索历史邮件。
- **邮件发送**：纯文本、HTML、附件。
- **自动化集成**：标准化 **JSON** 输出，供 Shell、CI/CD、监控系统调用。

---

## 2. 技术选型

### 2.1 开发语言：Go

| 考量       | 说明 |
| ---------- | ---- |
| 并发       | Goroutine 适合并发 SOAP 请求（多搜索/多发送）。 |
| 交付形态   | 单一静态二进制，不依赖 .NET / JVM，部署轻量。 |
| 生态       | 可选用社区 EWS 客户端（如 `github.com/domino14/ews`）加速实现。 |

### 2.2 协议：EWS

| 考量     | 说明 |
| -------- | ---- |
| 能力     | 原生 API，支持服务端搜索、文件夹遍历等，强于 POP3/IMAP 的简单收发。 |
| 认证     | Basic、NTLM、OAuth 2.0 等，适配不同企业策略。 |
| 数据行为 | 在服务端操作，避免 POP3「独占收取」类问题。 |

---

## 3. 功能需求

### 3.1 连接与认证

| 功能       | 描述                                                         | 优先级 |
| ---------- | ------------------------------------------------------------ | ------ |
| 自动发现   | 通过邮箱地址 Autodiscover EWS 端点 URL。                     | P0     |
| 手动端点   | 用户直接指定 EWS URL。                                       | P0     |
| 多认证方式 | Basic、NTLM、OAuth 2.0（Access Token）。                   | P0     |
| 凭证安全   | 从环境变量、加密配置或交互输入读取；避免敏感信息进 shell 历史。 | P0     |

### 3.2 邮件搜索

| 功能       | 描述                                                         | 优先级 |
| ---------- | ------------------------------------------------------------ | ------ |
| 基础搜索   | 发件人、收件人、主题、正文关键词。                           | P0     |
| 时间范围   | `--since` / `--until`；可选自然语言（如 `--since "7 days ago"`）。 | P0     |
| 文件夹     | 指定文件夹（收件箱、已发送、自定义）；默认收件箱。           | P1     |
| 未读状态   | 按未读/已读过滤。                                            | P1     |
| 附件过滤   | 仅含附件的邮件。                                             | P2     |
| 分页与限制 | `--limit` 等，避免单次拉取过多导致超时。                     | P0     |
| 输出格式   | `table`（终端）与 `json`（脚本）。                           | P0     |

### 3.3 邮件发送

| 功能     | 描述                                   | 优先级 |
| -------- | -------------------------------------- | ------ |
| 基础发送 | 发件人、收件人、抄送、密送。           | P0     |
| 内容格式 | 纯文本 `--text`、HTML `--html`。       | P0     |
| 附件     | `--attach` 路径，支持多个。            | P0     |
| 优先级   | 高 / 普通 / 低。                       | P1     |
| 发送确认 | 成功返回邮件 `ItemId`，便于追踪。      | P1     |

### 3.4 辅助功能

| 功能     | 描述                                      | 优先级 |
| -------- | ----------------------------------------- | ------ |
| 配置文件 | YAML/TOML；支持多账户 **Profile**。       | P0     |
| 日志     | `--verbose` 输出请求/响应级调试信息。     | P1     |
| 版本     | `--version`。                             | P0     |
| 帮助     | 完整 `--help`（子命令与参数）。           | P0     |

---

## 4. 非功能需求

### 4.1 安全性

- **传输**：仅 HTTPS，禁止明文 HTTP。
- **凭证**：
  - 环境变量：`EWS_PASSWORD` 或 `EWS_TOKEN`（见第 8 节）。
  - 交互：`-p` 无值时提示输入密码。
  - 配置文件：密码字段宜加密或对接密钥管理。
- **账号**：建议专用服务账号，最小邮箱权限。

### 4.2 性能

- 搜索：十万级邮箱按主题搜索，目标约 **5 秒内**（受服务器影响）。
- 并发发送：`--concurrency`，默认 **5**。
- 超时：网络请求默认 **30 秒**。

### 4.3 可靠性

- **重试**：对 503、超时等临时故障，指数退避，最多 **3 次**。
- **错误信息**：区分认证失败、网络错误、权限不足等。
- **幂等**：发送可记录已发 `ItemId`，避免误重复（策略待定）。

### 4.4 跨平台

- **目标**：Windows amd64；Linux amd64/arm64；macOS amd64/arm64。
- **路径**：适配各 OS 路径分隔符。

---

## 5. CLI 用户交互设计

### 5.1 命名与结构

- 二进制名：**`mailcli`**（产品名 mailCli）
- 形态：`mailcli [global options] <command> [command options] [arguments...]`

### 5.2 全局参数

| 参数       | 说明           | 默认值           |
| ---------- | -------------- | ---------------- |
| `--config` | 配置文件路径   | `~/.mailcli.yaml` |
| `--profile`| 使用的 Profile | `default`        |
| `--verbose`| 详细日志       | `false`          |

### 5.3 子命令示例

**搜索**

```bash
mailcli search \
  --profile work \
  --subject "周报" \
  --since "2026-05-01" \
  --limit 10 \
  --output json

mailcli search \
  --endpoint "https://mail.company.com/EWS/Exchange.asmx" \
  --user "admin@company.com" \
  --from "boss@company.com" \
  --unread \
  --folder "Inbox"
```

**发送**

```bash
mailcli send \
  --profile work \
  --to "team@company.com" \
  --subject "构建通知" \
  --text "CI/CD 构建成功，版本 v2.3.1"

mailcli send \
  --profile work \
  --to "client@example.com" \
  --cc "manager@company.com" \
  --subject "月度报告" \
  --html "<h1>报告摘要</h1><p>请查收附件。</p>" \
  --attach "./report.pdf"
```

### 5.4 配置文件示例（`~/.mailcli.yaml`）

```yaml
profiles:
  default:
    endpoint: https://outlook.office365.com/EWS/Exchange.asmx
    user: default@company.com
    auth_type: basic
  work:
    endpoint: https://mail.company.com/EWS/Exchange.asmx
    user: admin@company.com
    auth_type: ntlm
    domain: COMPANY
  oauth:
    endpoint: https://outlook.office365.com/EWS/Exchange.asmx
    user: bot@company.com
    auth_type: oauth
    tenant_id: xxx-xxx-xxx
    client_id: xxx-xxx-xxx
```

---

## 6. 开发路线图

### 第一阶段：核心（约 2 周）

- CLI 骨架（建议 **cobra + viper**）。
- EWS 连接与认证：**Basic + NTLM**。
- 搜索：主题、发件人、时间过滤。
- 发送：纯文本 + 附件。
- 配置文件读取。

### 第二阶段：增强（约 1 周）

- OAuth 2.0。
- HTML 发送。
- 重试、超时。
- `--output json`、`--verbose`。

### 第三阶段：测试与发布（约 1 周）

- 单元测试（Mock EWS）。
- 集成测试（测试邮箱）。
- GitHub Actions 多平台构建。
- README、Man page。

---

## 7. 风险与缓解

| 风险               | 影响                         | 缓解措施                                       |
| ------------------ | ---------------------------- | ---------------------------------------------- |
| EWS 版本差异       | 2013/2016/2019/Online 行为不同 | 覆盖主流版本测试；Schema 版本协商              |
| OAuth 配置复杂     | 用户上手成本高               | 文档与脚本示例                                 |
| 大附件超时         | 如 >10MB 易超时              | 分块上传（若协议支持）或明确限制与提示         |
| 超大邮箱搜索慢     | 如 >50 万封                  | 引导使用服务端索引；`--limit` 与分页           |

---

## 8. 环境变量与配置优先级

### 8.1 设计原则

- **安全**：密码、令牌、敏感端点尽量不进入命令行历史、进程列表、日志。
- **12-Factor**：配置与代码分离。
- **优先级**（建议）：**环境变量 > 配置文件 > 命令行**（命令行仅适合非敏感参数，如 `--limit`、`--output`；敏感项禁止仅靠历史中的明文参数）。

### 8.2 环境变量一览

| 变量名            | 说明                         | 必填     | 示例 |
| ----------------- | ---------------------------- | -------- | ---- |
| `EWS_ENDPOINT`    | EWS URL                      | 是*      | `https://mail.company.com/EWS/Exchange.asmx` |
| `EWS_USER`        | 邮箱/用户名                  | 是*      | `admin@company.com` |
| `EWS_PASSWORD`    | 密码或应用密码               | 视认证   | （不设入文档示例） |
| `EWS_AUTH_TYPE`   | `basic` / `ntlm` / `oauth` | 否，默认 `basic` | `ntlm` |
| `EWS_DOMAIN`      | NTLM 域名                    | NTLM 时需要 | `COMPANY` |
| `EWS_TIMEOUT`     | 超时（秒）                   | 否，默认 `30` | `60` |
| `EWS_CONCURRENCY` | 并发数                       | 否，默认 `5`（与 4.2 一致） | `8` |

\* 若已通过 `--config` + `--profile` 提供等价信息，则不必重复设置环境变量。

---

## 9. 文档维护

- 实现过程中若命令或变量与本文不一致，以 **README + `--help`** 为准，并回写同步本 PRD。
