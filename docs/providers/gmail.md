# Gmail 配置指南

## 服务器参数

| 项目 | 值 |
|------|-----|
| Provider ID | `gmail` |
| IMAP | `imap.gmail.com:993` |
| SMTP | `smtp.gmail.com:587`（STARTTLS） |

## 认证方式

Gmail 通常 **不能** 用普通 Google 账号密码连接第三方客户端，请任选其一：

### 方式 A：应用专用密码（推荐）

1. Google 账号开启 **两步验证**。
2. 打开 [Google 账号 → 安全性 → 应用专用密码](https://myaccount.google.com/apppasswords)。
3. 生成一个应用密码（16 位）。
4. `MAILCLI_PASSWORD` 填该应用密码；`user` 填完整 Gmail 地址。

### 方式 B：OAuth2

mailCli 尚未内置 Gmail OAuth 设备流；后续版本计划支持。当前请用应用专用密码。

## 开启 IMAP

1. Gmail 网页 → **设置** → **查看所有设置** → **转发和 POP/IMAP**。
2. 启用 **IMAP**。

## 配置示例

[examples/providers/gmail.yaml](../examples/providers/gmail.yaml)

```yaml
profiles:
  gmail:
    protocol: imap
    user: you@gmail.com
```

```bash
export MAILCLI_PASSWORD='应用专用密码'
mailcli --profile gmail search --limit 5
```
