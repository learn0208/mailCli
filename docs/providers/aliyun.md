# 阿里邮箱（aliyun.com）配置指南

| 项目 | 值 |
|------|-----|
| Provider ID | `aliyun` |
| IMAP | `imap.aliyun.com:993` |
| SMTP | `smtp.aliyun.com:465` |

## 认证

以 [阿里邮箱帮助中心](https://help.aliyun.com/) 当前说明为准：个人邮箱可能使用登录密码或客户端专用密码。

示例：[examples/providers/aliyun.yaml](../examples/providers/aliyun.yaml)

```yaml
profiles:
  aliyun:
    protocol: imap
    user: you@aliyun.com
```
