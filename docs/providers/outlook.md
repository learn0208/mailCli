# Outlook / Hotmail / Live 配置指南

| 项目 | 值 |
|------|-----|
| Provider ID | `outlook` |
| 域名 | `outlook.com`、`hotmail.com`、`live.com`、`msn.com` |
| IMAP | `outlook.office365.com:993` |
| SMTP | `smtp.office365.com:587`（STARTTLS） |

## 认证

- 未开启两步验证：通常可使用 Microsoft 账号密码。
- 已开启两步验证：在 Microsoft 账户安全中心创建 **应用密码**，填入 `MAILCLI_PASSWORD`。

需在 Outlook 网页设置中允许 IMAP（默认多数账号已开）。

示例：[examples/providers/outlook.yaml](../examples/providers/outlook.yaml)

```yaml
profiles:
  outlook:
    protocol: imap
    user: you@outlook.com
```
