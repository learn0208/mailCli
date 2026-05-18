# 发布说明（维护者）

## 自动化 Release（推荐）

推送 **Git 标签** 后，[`.github/workflows/release.yml`](../.github/workflows/release.yml) 会：

1. 运行 `go test ./...`
2. 用 [GoReleaser](https://goreleaser.com/) 交叉编译多平台二进制
3. 上传到 **GitHub Releases**（含 `checksums.txt`）

### 步骤

```bash
# 1. 确保 CHANGELOG、internal/app.Version、标签一致
# 2. 提交并推到 main
git tag v0.1.0
git push origin main
git push origin v0.1.0
```

### 产物命名

`mailcli_<version>_<os>_<arch>.tar.gz`（Windows 为 `.zip`），例如：

- `mailcli_0.1.0_linux_amd64.tar.gz`
- `mailcli_0.1.0_windows_amd64.zip`
- `mailcli_0.1.0_darwin_arm64.tar.gz`

### 仓库设置（首次）

1. **Settings → Actions → General → Workflow permissions** → *Read and write*
2. 仓库地址：[learn0208/mailCli](https://github.com/learn0208/mailCli)
3. GoReleaser 会发布到当前 GitHub 仓库，无需在 `.goreleaser.yaml` 里写死 owner

### 本地试跑（可选）

```bash
go install github.com/goreleaser/goreleaser/v2@latest
goreleaser release --snapshot --clean
# 产物在 dist/
```

## 日常 CI

向 `main` / `master` 的 push 与 PR 会跑 [`.github/workflows/ci.yml`](../.github/workflows/ci.yml)（测试 + 编译冒烟）。

## Release 失败怎么查

1. 打开 [Actions](https://github.com/learn0208/mailCli/actions) → 点失败的 **Release** → 点红色 **goreleaser** job。
2. 展开 **Run GoReleaser**，**最下面几行**通常是真正原因（摘要里的 `exit code 1` 不够具体）。

常见原因：

| 日志关键词 | 原因 | 处理 |
|------------|------|------|
| `no such file` / `cmd/mailcli` | `cmd/mailcli` 未提交到 Git | 确认 `.gitignore` 用 `/mailcli` 而不是 `mailcli`；`git add cmd/mailcli` 后 push |
| `go test` failed | 测试未通过 | 本地 `go test ./...` 修完再发版 |
| `git is dirty` | 工作区有未提交改动 | 提交或清理后再打 tag |
| `version does not exist` | Actions 没有对应 Go 版本 | 在 `go.mod` 改用 Actions 支持的版本（如 `1.23`） |

## 重新发 v0.1.0

若 tag 已存在但 Release 失败：

```bash
# GitHub 网页：Releases 删除失败的 v0.1.0；Code → Tags 删除 v0.1.0
git push origin :refs/tags/v0.1.0   # 删除远程 tag
git tag -d v0.1.0                   # 删除本地 tag
# 修复并 push main 后：
git tag v0.1.0
git push origin v0.1.0
```
