# RecallLoom Lite (rll)

让项目自己记住自己——轻量、零状态、单二进制的 AI 协作项目记忆层。

rll 在项目的 `.rll/` 目录里用纯 Markdown 维护三份记忆：

- `context_brief.md` — 项目定位、阶段、边界（稳定框架，很少变）
- `rolling_summary.md` — 当前状态快照（每次覆盖式更新）
- `daily_logs/YYYY-MM-DD.md` — 里程碑日志（只追加）

没有 state.json、没有收据、没有版本绑定、没有安全层。文件即全部事实，
允许随时手工修改，`rll validate` 只做五项纯格式检查。`.rll/` 默认通过
`.git/info/exclude` 排除在 git 之外，不污染仓库。

## 安装

从 GitHub Releases 下载对应平台的二进制（linux / darwin / windows × amd64 / arm64），
放到 PATH 上。Skill 包（`rll_skill` 压缩包）解压到宿主的 skills 目录，例如：

```bash
mkdir -p ~/.config/opencode/skills/rll && tar -xzf rll_skill.tar.gz -C ~/.config/opencode/skills/rll
```

## 快速开始

```bash
cd your-project
rll init                     # 创建 .rll/ 骨架并配置 git 排除
cat <<'EOF' | rll write brief
# Context Brief

## What this project is

（让 agent 替你采访并填写）
EOF
rll log append --title "init memory"   # 正文从 stdin 读
rll status
rll resume                    # 冷启动：一次输出全部记忆
```

## 命令一览

| 命令 | 作用 |
|---|---|
| `rll init` | 初始化 `.rll/` 骨架 |
| `rll resume` | 输出全部记忆上下文（冷启动用） |
| `rll status` | 一屏概览 |
| `rll log append --title T` | 追加里程碑日志（stdin 传正文） |
| `rll write brief/summary/protocol` | 覆盖式更新受管文档 |
| `rll query 关键词` | 子串搜索 summary 与日志 |
| `rll validate` | 五项纯格式检查 |
| `rll archive --before 日期` | 归档旧日志（默认预览，--apply 执行） |

所有写命令支持 `--dry-run` 预览与 `--file` 从文件读入。时间戳默认取本机
时间，可用 `--date`/`--time` 补录。

## 设计取舍

- **零状态**：游标、最新日志等全部由目录扫描即时得出，永无同步错误。
- **无安全层**：删除原版 RecallLoom 的收据绑定与写保护，模型一次写入
  成功率优先；记录被人为改动属于正常使用，由开发者自行负责。
- **CLI 与 Skill 分离**：同一 Release 携带全部平台二进制与 skill 压缩包。

本项目是受 RecallLoom 启发的独立 Go 重写（见 NOTICE），不与其数据格式兼容。

## 许可证

Apache-2.0
