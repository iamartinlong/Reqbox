# 文档索引

本目录存放项目文档与开发过程记录。代码解释「怎么做」，这里解释
「为什么这样做」「还有什么没做」。

文档按**读者的目的**分组，不是按文件名首字母。想知道「下一步做什么」的人
和想知道「当初为什么这么设计」的人，不应该在同一个列表里翻。

---

## 先看这里

> **接手前必读一条**：两个分支的最新版本（`main` 的 `v0.3.0`、legacy 的
> `v0.1.2`）都已发布，功能一致。`legacy` 的 2008 R2 实机复验也已通过
> （2026-10-27），**当前没有遗留的验证项**。

| 你的情况 | 从这篇开始 |
| -------- | ---------- |
| 刚拿到这个仓库 | [onboarding/HANDOFF.md](./onboarding/HANDOFF.md) |
| 想用，不想读代码 | 仓库根目录的 [README](../README.md) |
| 要决定接下来做什么 | [planning/TODO.md](./planning/TODO.md) |
| 某个设计为什么是这样 | [design/DECISIONS.md](./design/DECISIONS.md) |
| 要跑起来 / 改代码 | [development/DEVELOPMENT.md](./development/DEVELOPMENT.md) |
| 部署到旧版 Windows | [development/LEGACY-WINDOWS.md](./development/LEGACY-WINDOWS.md) |
| 想知道最近改了什么 | [history/WORKLOG.md](./history/WORKLOG.md) |
| 要发版 | [development/DEVELOPMENT.md](./development/DEVELOPMENT.md) 的「发版」 |

---

## 目录结构

```
docs/
  README.md            本文件
  onboarding/          接手：跑起来、现在什么状态、坑在哪
  planning/            要做什么、按什么顺序做
  design/              为什么这么设计、安全状况
  development/         怎么构建、怎么验证、旧系统怎么支持
  history/             变过什么、每轮怎么做的、每个版本发了什么
```

### onboarding/ · 接手

| 文档 | 内容 |
| ---- | ---- |
| [HANDOFF.md](./onboarding/HANDOFF.md) | 当前分支的提交、两个分支的关系、环境要求、容易踩的坑、当前状态表 |

面向「我今天刚加入」的读者。只写怎么跑起来和现状，不解释设计理由。

### planning/ · 规划

| 文档 | 内容 |
| ---- | ---- |
| [TODO.md](./planning/TODO.md) | 待办，按优先级；每条都有「为什么」和验收方式 |
| [PLAN.md](./planning/PLAN.md) | 执行版：分阶段步骤、落地要点、里程碑 |

两者分工固定：**TODO 说要做什么，PLAN 说怎么做**。新增待办只写进 TODO；
只有准备动手时才在 PLAN 里补步骤。

### design/ · 设计与风险

| 文档 | 内容 |
| ---- | ---- |
| [DECISIONS.md](./design/DECISIONS.md) | 决策记录（DR-001 …），含「什么情况下重新考虑」 |
| [AUDIT.md](./design/AUDIT.md) | 上线审计：发现的问题、修复情况、遗留风险 |

决策记录的价值在**理由**，不在结论。想推翻一条决策时，先看它写的重议条件。

### development/ · 开发

| 文档 | 内容 |
| ---- | ---- |
| [DEVELOPMENT.md](./development/DEVELOPMENT.md) | 环境、构建、验证清单、CI 跑什么、怎么发版 |
| [LEGACY-WINDOWS.md](./development/LEGACY-WINDOWS.md) | 为什么旧系统要单独构建，以及怎么做 |

### history/ · 历史

| 文档 | 内容 |
| ---- | ---- |
| [CHANGELOG.md](./history/CHANGELOG.md) | 面向使用者：行为变了什么、修了哪些缺陷 |
| [WORKLOG.md](./history/WORKLOG.md) | 面向维护者：每轮做了什么、怎么验证、踩了什么坑 |
| [release-notes/](./history/release-notes/) | 每个版本的手写发布说明，发版工作流从这里取 |

三者的区别，用一次真实改动举例：把端口写死改成可配置，

- CHANGELOG 写「端口可以配置了，冲突时自动顺延」；
- WORKLOG 写「为什么会踩到、为什么选这个优先级、日志里为什么要打印实际端口」；
- release-notes 写「这个版本给到用户什么、怎么验证升级成功」。

`release-notes/<tag>.md` 的文件名必须与 tag 完全一致（例如
`v0.2.0.md`、`v0.1.2-win2008r2.md`）。发版工作流按 tag 名查找它，
**找不到就退回自动生成的提交列表**，不会失败。

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

- **新增文档先归组。** 找不到合适的组，说明该组还没成立 —— 要么并进现有
  文档，要么先想清楚它服务谁。不要往 `docs/` 根目录加文件。
- **移动文档必须同时改链接。** 相对链接会随目录变化而失效；改完跑一遍
  链接检查（见下）。
- **过期比缺失更糟。** 与其留一句已经不对的陈述，不如删掉或标注
  「截至某版本」。文档里出现具体数字（测试数、提交号）时，要写清是哪一天的。
- **区分「描述现在」和「记录过去」。** 这是本仓库最容易反复踩的一条：
  - 「当前 HEAD 是 `abc1234`」「最新发布是 v0.3.0」→ 描述现在 → 数字一变
    就过期，**必须改**。
  - 「2026-10-21 从 `ead2f5f` 起步」→ 记录过去 → **永远不该改**，改了就是
    篡改历史。

  已经因为前者返工过四次。写当前状态时优先用 **tag 或版本号**（不会移动），
  提交号只在「确认代码与发布是否一致」这类必须精确的地方使用，并且记得
  下次推送后回来改。

### 检查链接

移动或新增文档后，确认所有相对链接和锚点都能解析：

```bash
python - <<'PY'
import io, os, posixpath, re
docs = "docs"
bad = []
for dp, _, fs in os.walk(docs):
    for f in fs:
        if not f.endswith(".md"):
            continue
        p = os.path.join(dp, f)
        rel = os.path.relpath(p, docs).replace(os.sep, "/")
        text = io.open(p, encoding="utf-8").read()
        for m in re.finditer(r"\]\(([^)\s]+)\)", text):
            t = m.group(1)
            if t.startswith(("http", "#", "mailto:")):
                continue
            dest = posixpath.normpath(posixpath.join(posixpath.dirname(rel), t.split("#")[0]))
            if not os.path.exists(os.path.join(docs, dest.replace("/", os.sep))):
                bad.append(f"{rel}: {t}")
print("\n".join(bad) if bad else "ok")
PY
```

## 与根目录 README 的分工

- 根目录 `README.md`：面向**使用者**。怎么启动、有哪些功能、API 长什么样。
- `docs/`：面向**维护者**。待办、决策理由、开发流程、变更历史。

避免把维护者向的内容塞进 README —— 使用者不需要知道我们还没做什么。
