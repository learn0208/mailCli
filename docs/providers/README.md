# 常见邮箱配置指南

本目录为 **IMAP/SMTP** 各服务商的详细配置说明，便于日后查阅与复制。通用命令见 [使用说明](../使用说明.md)。

## 快速索引

| 服务商 | 文档 | 示例配置 |
|--------|------|----------|
| QQ 邮箱 / Foxmail | [qq.md](qq.md) | [qq.yaml](../examples/providers/qq.yaml) |
| 网易 163 | [163.md](163.md) | [163.yaml](../examples/providers/163.yaml) |
| 网易 126 | [126.md](126.md) | [126.yaml](../examples/providers/126.yaml) |
| 网易 yeah.net | [yeah.md](yeah.md) | [yeah.yaml](../examples/providers/yeah.yaml) |
| Gmail | [gmail.md](gmail.md) | [gmail.yaml](../examples/providers/gmail.yaml) |
| Yahoo | [yahoo.md](yahoo.md) | [yahoo.yaml](../examples/providers/yahoo.yaml) |
| Outlook / Hotmail | [outlook.md](outlook.md) | [outlook.yaml](../examples/providers/outlook.yaml) |
| iCloud | [icloud.md](icloud.md) | [icloud.yaml](../examples/providers/icloud.yaml) |
| 新浪邮箱 | [sina.md](sina.md) | [sina.yaml](../examples/providers/sina.yaml) |
| 阿里邮箱 | [aliyun.md](aliyun.md) | [aliyun.yaml](../examples/providers/aliyun.yaml) |

## 命令行查看

```bash
# 列出内置预设
mailcli providers list

# 查看某一家的连接参数与认证说明
mailcli providers show qq --user 123456789@qq.com

# 在终端输出完整配置文档（与本文档同内容）
mailcli providers doc qq

# 根据邮箱域名给出提示
mailcli discover --user you@163.com
```

## 通用约定

1. 配置文件默认路径：`~/.mailcli.yaml`（UTF-8）。
2. **不要**把授权码/密码写入 YAML 提交到 Git；使用环境变量 `MAILCLI_PASSWORD`。
3. `protocol: imap` 时：`search` / `show` 走 IMAP，`send` 走 SMTP。
4. 仅填写 `user: you@domain.com` 时，程序会按域名自动匹配 `provider` 并填入服务器地址（见各文档「最简配置」）。

## 密码说明（重要）

| 类型 | 填什么 | 典型服务商 |
|------|--------|------------|
| 登录密码 | 网页/App 登录用的密码 | Outlook（无 2FA 时） |
| **授权码 / 客户端密码** | 在邮箱设置里单独生成 | **QQ、163、126** |
| **应用专用密码** | 在账号安全中心生成 | **Gmail、Yahoo、iCloud** |

**QQ 邮箱不支持扫码登录用于 IMAP/SMTP**，必须使用授权码，详见 [qq.md](qq.md)。
