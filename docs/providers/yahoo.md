# Yahoo Mail 配置指南

| 项目 | 值 |
|------|-----|
| Provider ID | `yahoo` |
| IMAP | `imap.mail.yahoo.com:993` |
| SMTP | `smtp.mail.yahoo.com:465` |

## 认证

1. Yahoo 账号安全设置中生成 **应用密码（App password）**。
2. `MAILCLI_PASSWORD` 使用该应用密码。

示例：[examples/providers/yahoo.yaml](../examples/providers/yahoo.yaml)

```yaml
profiles:
  yahoo:
    protocol: imap
    user: you@yahoo.com
```
