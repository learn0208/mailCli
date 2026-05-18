# 邮箱 Profile 示例

将所需文件中的 `profiles` 段合并进 `~/.mailcli.yaml`，或整文件复制后改名为 `~/.mailcli.yaml`。

**切勿**在 YAML 中填写真实密码；使用环境变量：

```bash
export MAILCLI_PASSWORD='授权码或应用专用密码'
export MAILCLI_PROFILE=qq   # 与下面 profile 名称一致
```

| 文件 | 说明 |
|------|------|
| [qq.yaml](qq.yaml) | QQ 邮箱（授权码） |
| [163.yaml](163.yaml) | 网易 163 |
| [126.yaml](126.yaml) | 网易 126 |
| [yeah.yaml](yeah.yaml) | 网易 yeah.net |
| [gmail.yaml](gmail.yaml) | Gmail（应用专用密码） |
| [yahoo.yaml](yahoo.yaml) | Yahoo |
| [outlook.yaml](outlook.yaml) | Outlook / Hotmail |
| [icloud.yaml](icloud.yaml) | iCloud |
| [sina.yaml](sina.yaml) | 新浪 |
| [aliyun.yaml](aliyun.yaml) | 阿里邮箱 |

详细步骤见 [docs/providers/](../../providers/README.md)。
