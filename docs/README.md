# 文档索引（兼容分支）

本目录存放 `legacy/win2008r2` 分支的文档。**主分支的文档在另一条分支上**，
结构相同但内容不同——这里只保留与旧系统兼容相关的部分。

---

## 先看这里

| 你的情况 | 从这篇开始 |
| -------- | ---------- |
| 刚接手这个分支 | [onboarding/HANDOFF.md](./onboarding/HANDOFF.md) |
| 要构建兼容包 | [development/LEGACY-WINDOWS.md](./development/LEGACY-WINDOWS.md) |
| 要发版 | [development/LEGACY-WINDOWS.md](./development/LEGACY-WINDOWS.md) 的「CI 与发版」 |
| 还剩什么没做 | [planning/TODO.md](./planning/TODO.md) |
| 为什么某个决定是这样 | [design/DECISIONS.md](./design/DECISIONS.md) |
| 想知道最近改了什么 | [history/CHANGELOG.md](./history/CHANGELOG.md) |

---

## 目录结构

```
docs/
  README.md            本文件
  onboarding/          接手本分支：五条不能违反的规则、坑
  planning/            本分支独有的待办（技术债、功能缺口）
  design/              决策记录
  development/         兼容性原理、构建、CI、发版
  history/             变更记录、每个版本的发布说明
```

### 各组内容

| 组 | 文档 | 内容 |
| -- | ---- | ---- |
| onboarding | [HANDOFF.md](./onboarding/HANDOFF.md) | 分支为什么存在、五条规则、常用命令、坑 |
| planning | [TODO.md](./planning/TODO.md) | 技术债、功能缺口；2008 R2 复验步骤（下次改动时用） |
| design | [DECISIONS.md](./design/DECISIONS.md) | 决策记录 |
| development | [LEGACY-WINDOWS.md](./development/LEGACY-WINDOWS.md) | 兼容性原理、`legacyguard`、构建、CI 与发版 |
| development | [DEVELOPMENT.md](./development/DEVELOPMENT.md) | 本地开发与验证流程 |
| history | [CHANGELOG.md](./history/CHANGELOG.md) | 本分支的变更记录 |
| history | [release-notes/](./history/release-notes/) | 每个兼容包的手写发布说明 |

### 与主分支的差异

| 组 | 本分支 | 主分支 |
| -- | ------ | ------ |
| onboarding | HANDOFF | HANDOFF |
| planning | 只有 TODO | TODO + PLAN |
| design | 只有 DECISIONS | DECISIONS + AUDIT |
| development | LEGACY-WINDOWS + DEVELOPMENT | DEVELOPMENT + 简版 LEGACY-WINDOWS |
| history | CHANGELOG + release-notes | CHANGELOG + WORKLOG + release-notes |

本分支不维护 **PLAN**、**AUDIT** 与 **WORKLOG**：那三份记录的是规划过程与
一轮轮改动的来龙去脉，与工具链版本无关，在 `main` 上有一份就够了。重复维护
只会产生两个版本不一致的副本。

发布说明按 tag 名存放（`v0.1.2-win2008r2.md`），发版工作流只认这个后缀，
主分支的 `v0.2.0` 之类不会误触发本分支的流水线。

---

## 三条约定

1. **待办要写清验收方式。**
   只写「优化性能」等于没写 —— 无法判断何时算完成。每条 TODO 都应能回答
   「怎么确认它做完了」。

2. **决策要记录「当时的理由」和「什么情况下重新考虑」。**
   记录理由，是为了避免后人不知道背景就推翻；记录重议条件，是为了避免
   把已经过时的决定当成不可动摇的教条。

3. **过程文档面向未来的人，不面向写的人。**
   不写「今天调了很久」，而是写「这个坑的表现、根因、如何避免」。
   读者是三个月后已经忘记细节的自己和同事。

## 维护规则

- **新增文档先归组。** 不要往 `docs/` 根目录加文件。
- **移动文档必须同时改链接。** 相对链接会随目录变化而失效。
- **过期比缺失更糟。** 文档里的具体数字（测试数、提交号、验证状态）要写清
  是哪一天的。本分支尤其要注意：`go test` 的数量随主分支移植的代码变化。

## 与根目录 README 的分工

- 根目录 `README.md`：面向**使用者**。怎么启动、有哪些功能、API 长什么样。
- `docs/`：面向**维护者**。待办、决策理由、开发流程、变更历史。

避免把维护者向的内容塞进 README —— 使用者不需要知道我们还没做什么。
