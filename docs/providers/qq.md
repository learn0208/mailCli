# QQ 邮箱 / Foxmail 配置指南

## 服务器参数

| 项目 | 值 |
|------|-----|
| Provider ID | `qq` |
| 域名 | `qq.com`、`foxmail.com` |
| IMAP | `imap.qq.com:993`（SSL/TLS） |
| SMTP | `smtp.qq.com:465`（SSL/TLS） |
| 已发送文件夹（校验用） | `Sent Messages`、`已发送` |

## 能否扫码登录？

**不能。** QQ 邮箱网页/App 的扫码登录仅用于腾讯 Web 会话；mailCli 使用标准 **IMAP/SMTP**，只能使用 **邮箱地址 + 授权码**。

## 开启 IMAP/SMTP 并获取授权码

1. 浏览器打开 [https://mail.qq.com](https://mail.qq.com) 并登录。
2. **设置** → **账户**。
3. 找到 **POP3/IMAP/SMTP/Exchange/CardDAV/CalDAV 服务**，开启 **IMAP/SMTP 服务**。
4. 按提示完成安全验证（密保手机等）。
5. 点击 **生成授权码**，按短信/提示操作，得到 **16 位授权码**（仅显示一次，请妥善保存）。
6. 将授权码用于 `MAILCLI_PASSWORD`，**不要**使用 QQ 登录密码。

## 配置文件

复制 [examples/providers/qq.yaml](../examples/providers/qq.yaml) 到 `~/.mailcli.yaml`，修改 `user` 为你的 QQ 邮箱地址。

**最简配置（推荐）** — 只需 `provider` + `user`，**不必**写 `imap` / `smtp`，程序会自动填入 `imap.qq.com:993`、`smtp.qq.com:465` 及 TLS：

```yaml
profiles:
  qq:
    protocol: imap
    provider: qq
    user: 123456789@qq.com
```

也可省略 `provider`，仅写 `user: xxx@qq.com`（按 `@qq.com` 域名自动识别）。

查看合并后的实际参数：

```bash
mailcli profile show
```

**仅在需要覆盖默认值时** 才手写 `imap` / `smtp` 段（例如非标准端口）。

## 环境变量

```bash
export MAILCLI_PROFILE=qq
export MAILCLI_PASSWORD='你的16位授权码'
# 可选：覆盖主机
# export MAILCLI_IMAP_HOST=imap.qq.com:993
# export MAILCLI_SMTP_HOST=smtp.qq.com:465
```

## 验证（search → show → send）

```bash
export MAILCLI_PROFILE=qq
export MAILCLI_PASSWORD='你的16位授权码'

# 搜索（主题 / 关键词）
mailcli search --subject "招商银行信用卡"
mailcli search "信用管家" --limit 5
mailcli search --query "招商银行" --output table

# 读信（ITEM_ID 列为 UID）
mailcli show --item-id 1460 --format text
mailcli show --item-id 1485 --format html

# 发信
mailcli send --to 你的其他邮箱@example.com --subject "mailcli 测试" --text "hello"
```

## 常见问题

| 现象 | 处理 |
|------|------|
| 登录失败 / 认证错误 | 确认用的是 **授权码**，不是 QQ 密码；授权码是否过期需重新生成 |
| 未开启 IMAP | 在网页设置中重新开启 IMAP/SMTP |
| 发送成功但校验不到已发送 | 不同账号「已发送」文件夹名可能不同；可用 `--no-verify-sent` 跳过 |
