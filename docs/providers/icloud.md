# iCloud 邮件配置指南

| 项目 | 值 |
|------|-----|
| Provider ID | `icloud` |
| 域名 | `icloud.com`、`me.com`、`mac.com` |
| IMAP | `imap.mail.me.com:993` |
| SMTP | `smtp.mail.me.com:587` |

## 认证

1. 使用 Apple ID 登录 [appleid.apple.com](https://appleid.apple.com)。
2. **登录与安全** → **App 专用密码**，生成密码。
3. `MAILCLI_PASSWORD` 填 App 专用密码（不是 Apple ID 主密码）。

示例：[examples/providers/icloud.yaml](../examples/providers/icloud.yaml)

```yaml
profiles:
  icloud:
    protocol: imap
    user: you@icloud.com
```
