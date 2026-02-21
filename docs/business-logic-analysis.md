# Skillz 业务逻辑分析

## 1. 项目定位

Skillz 是一个 MCP Server。它把本地的“Anthropic 风格 Skills”（`SKILL.md` + 资源文件）暴露为：

- 可调用的 MCP Tool（每个 skill 对应一个 tool）
- 可读取的 MCP Resource（skill 内除 `SKILL.md` 外的文件）
- 一个兜底工具 `fetch_resource`（供不支持原生 MCP Resource 的客户端使用）

核心代码入口与职责：

- `src/skillz/__main__.py`：CLI 入口
- `src/skillz/_server.py`：绝大部分业务逻辑（解析、发现、注册、资源读取、服务运行）
- `tests/*.py`：行为规范，覆盖目录技能、zip/.skill 技能、资源读取、安全校验

---

## 2. 业务对象模型

### 2.1 SkillMetadata

来自 `SKILL.md` YAML front matter：

- `name`（必填）
- `description`（必填）
- `license`（可选）
- `allowed-tools` / `allowed_tools`（可选）
- 其余字段进入 `extra`

### 2.2 Skill

运行时技能实体，支持两类后端载体：

1. 目录技能（`<dir>/SKILL.md`）
2. 压缩包技能（`.zip` 或 `.skill`）

关键行为：

- `read_body()`：读取并剥离 front matter，返回指令正文
- `iter_resource_paths()`：枚举资源（排除 `SKILL.md` 与 macOS 元数据）
- `open_bytes()` / `exists()`：统一目录与 zip 的文件访问

### 2.3 SkillRegistry

技能注册中心，职责：

- 从 root 递归扫描技能
- 按 `slug` 与 `name` 去重
- 持有 `skills` 集合并支持 `get(slug)` 查询

---

## 3. 核心业务流程

### 3.1 启动流程

1. CLI 解析参数（skills 根目录、transport、host/port/path、日志、`--list-skills`）
2. 初始化日志
3. `SkillRegistry.load()` 扫描并加载技能
4. 若 `--list-skills`：打印并退出
5. 构建 FastMCP Server：
   - 注册 `fetch_resource`
   - 为每个 skill 注册 resources 与 tool
6. 按 transport 运行服务

### 3.2 技能发现流程

递归扫描规则：

- 目录下若有 `SKILL.md`：该目录即 skill，停止继续下钻
- 先递归子目录，再处理当前目录 `.zip`/`.skill`（目录技能优先）
- zip skill 仅在以下结构时有效：
  - `SKILL.md` 在压缩包根目录
  - 或存在唯一顶层目录，且其中有 `<top>/SKILL.md`

### 3.3 技能解析与校验

front matter 校验：

- 必须存在 YAML front matter（`---` 包裹）
- `name`/`description` 必填
- `allowed-tools` 兼容字符串（逗号分割）或数组

校验失败策略：

- 单个技能失败仅跳过，不阻断整体加载

### 3.4 Tool 调用流程（以某个 skill tool 为例）

输入：`task`（非空字符串）

输出结构：

- `skill`、`task`
- `metadata`
- `resources`（URI + name + mime_type）
- `instructions`（SKILL.md 正文）
- `usage`（如何应用 skill 与获取资源的统一指导）

业务本质：

- skill tool 不直接“执行任务”，而是返回专家指令与资源索引

### 3.5 Resource 读取流程

两条路径：

1. MCP 原生 resources（优先）
2. `fetch_resource(resource_uri=...)`（兜底）

URI 规范：

- `resource://skillz/{skill-slug}/{path}`

内容返回：

- 文本：`encoding=utf-8`
- 二进制：`encoding=base64`

错误处理：

- 返回“错误资源对象”（不抛出到客户端）

---

## 4. 安全与边界控制

已实现：

- URI 前缀白名单（仅 `resource://skillz/`）
- 禁止路径穿越（拒绝 `..` 与绝对路径）
- zip 异常容错（坏包/乱码不导致进程崩溃）

需关注：

- 当前属于实验性 PoC，README 明确提示需在隔离环境运行
- `fetch_resource` 是兼容兜底路径，实际生产应优先 MCP 原生资源机制

---

## 5. 行为特征（从测试反推的稳定契约）

已覆盖并可视作“外部契约”的行为：

- 默认 skills 目录为 `~/.skillz`
- 目录技能与 zip/.skill 技能可共存
- 目录技能优先于同名 zip 技能
- 资源元数据仅保留 `uri/name/mime_type`
- `SKILL.md` 只用于 tool instructions，不作为资源暴露
- zip 中 `__MACOSX/` 与 `.DS_Store` 会被过滤

这部分在重写时必须保持兼容，否则会破坏现有客户端行为。

---

## 6. 当前架构总结

这是一个“技能索引 + 指令分发 + 资源分发”的轻状态服务：

- 计算量不高，I/O 与协议适配为主
- 业务复杂度主要在“发现规则、兼容细节、错误兜底语义”
- 对外核心价值是“统一 skill 语义 + MCP 标准接入”
