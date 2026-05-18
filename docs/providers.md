# 常见邮箱配置（索引）

完整配置文档与可复制 YAML 示例已整理在专用目录，便于日后查阅：

| 内容 | 路径 |
|------|------|
| **配置指南目录（推荐）** | [docs/providers/README.md](providers/README.md) |
| **YAML 示例** | [docs/examples/providers/](examples/providers/) |
| 命令行速查 | `mailcli providers doc <id>` |

## 支持的服务商

`gmail` · `yahoo` · `outlook` · `icloud` · `qq` · `163` · `126` · `yeah` · `sina` · `aliyun`

```bash
mailcli providers list
mailcli providers doc qq      # 打印 QQ 邮箱完整配置说明
mailcli providers show qq --user you@qq.com
```

## 一键合并示例配置

将示例合并到用户配置（需自行修改 `user`）：

```bash
# 查看示例路径
ls docs/examples/providers/

# 手动合并 qq.yaml 中的 profiles 段到 ~/.mailcli.yaml
export MAILCLI_PASSWORD='你的授权码'
mailcli --profile qq search --limit 3
```

各服务商的特殊步骤（授权码、应用专用密码、是否支持扫码等）见对应 `docs/providers/<id>.md`。
