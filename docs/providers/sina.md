# 新浪邮箱配置指南

| 项目 | 值 |
|------|-----|
| Provider ID | `sina` |
| 域名 | `sina.com`、`sina.cn` |
| IMAP | `imap.sina.com:993` |
| SMTP | `smtp.sina.com:465` |

## 认证

1. 登录新浪邮箱网页版，在设置中开启 **IMAP/SMTP**。
2. 若要求 **客户端授权码/独立密码**，将其填入 `MAILCLI_PASSWORD`。

示例：[examples/providers/sina.yaml](../examples/providers/sina.yaml)

```yaml
profiles:
  sina:
    protocol: imap
    user: you@sina.com
```
