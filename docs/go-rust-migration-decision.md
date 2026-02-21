# Skillz Go / Rust 重写决策方案

## 1. 决策目标

将当前 Python 实现迁移到 Go 或 Rust，同时满足：

1. 保持现有外部行为兼容（CLI、Tool 输出、Resource URI、错误语义）
2. 提升可部署性（单二进制、低运行时依赖）
3. 增强稳健性与安全性（尤其是 zip 处理与路径边界）
4. 控制迁移风险与交付周期

---

## 2. 必须保持的兼容契约（迁移红线）

### 2.1 协议与数据

- Skill Tool 输出字段结构不变：`skill/task/metadata/resources/instructions/usage`
- Resource URI 规范不变：`resource://skillz/{slug}/{path}`
- 文本与二进制编码语义不变（utf-8/base64）
- `fetch_resource` 错误返回采用“错误资源对象”而非进程级异常

### 2.2 发现与解析

- 目录优先于 zip/.skill
- zip 支持：根 `SKILL.md` 或“单顶层目录 + SKILL.md”
- front matter 校验规则不变（必填 `name` 与 `description`）
- `SKILL.md` 不作为资源暴露

### 2.3 安全与鲁棒性

- 保留 URI 前缀校验与 path traversal 防护
- 坏 zip / 非法 YAML 保持“跳过并继续”策略

---

## 3. Go 与 Rust 对比（面向本项目）

| 维度 | Go | Rust |
|---|---|---|
| 开发效率 | 高，团队上手快，迭代快 | 中，类型/生命周期学习曲线更陡 |
| 运行时性能 | 足够且稳定 | 更高上限，低内存占用更优 |
| 内存安全 | 依赖工程规范 | 语言级强保障（编译期） |
| 并发模型 | goroutine 简洁实用 | async 生态强但复杂度更高 |
| MCP/HTTP 工程化 | 成熟，构建单文件 CLI 简单 | 可实现但样板与类型系统成本更高 |
| 招聘/维护成本 | 较低 | 中到高 |
| 迁移风险（短期） | 低到中 | 中到高 |

结论倾向：

- 如果优先“快速稳定替换 Python 服务并上线”，Go 更合适
- 如果优先“长期极致安全与性能，并接受更高初期成本”，Rust 更合适

---

## 4. 建议决策（默认场景）

推荐：**先 Go，后 Rust 评估**。

适用前提：

- 当前目标是尽快获得单二进制交付、减少 Python 运行时依赖
- 团队需要控制重写周期和可维护风险
- 业务负载以 I/O 为主，对极限性能并非第一目标

建议里程碑：

1. **M1：兼容内核（2-3 周）**
   - SkillRegistry + front matter 解析 + resource URI + fetch_resource
   - 用现有 Python tests 作为行为基线，做跨语言契约测试
2. **M2：MCP 服务器与 CLI（1-2 周）**
   - stdio/http/sse、list-skills、日志参数齐平
3. **M3：灰度与替换（1-2 周）**
   - 双栈运行（Python/Go）对比输出
   - 完成回归后切换默认实现

---

## 5. 两种实现的技术选型建议

### 5.1 Go 方案（推荐）

- CLI：`cobra` 或标准库 `flag`
- YAML：`gopkg.in/yaml.v3`
- zip：标准库 `archive/zip`
- 路由/HTTP：标准库 `net/http`
- MCP：选稳定 Go MCP SDK；若能力不足，先实现 stdio + HTTP 兼容层

代码结构建议：

- `cmd/skillz`：入口
- `internal/registry`：扫描、解析、去重
- `internal/resource`：URI 编解码、读取、编码策略
- `internal/server`：MCP tool/resource 注册
- `internal/model`：Skill/Metadata 结构定义

### 5.2 Rust 方案（候选）

- CLI：`clap`
- YAML：`serde_yaml`
- zip：`zip`
- async runtime：`tokio`
- 序列化：`serde`
- MCP：选稳定 Rust MCP SDK，或以协议层自行封装

Rust 更适合作为二期（在 Go 版本稳定后，按需替换核心模块）。

---

## 6. 风险清单与缓解

1. **行为偏差风险**：资源 URI/错误文案差异导致客户端异常
   - 缓解：建立 golden tests（输入目录 -> 输出 JSON 快照）
2. **zip 兼容风险**：单顶层目录、macOS 垃圾文件过滤等细节遗漏
   - 缓解：完整迁移现有 `test_zip_skills.py` 场景
3. **MCP SDK 差异风险**：不同语言 SDK 对 resource/tool 语义实现细节不同
   - 缓解：先做协议契约测试，再接 SDK
4. **迁移中断风险**：一次性替换失败
   - 缓解：双栈灰度 + 环境开关回退

---

## 7. 执行建议（你可以直接采用）

- 决策：**Go 重写为主线**
- 策略：**契约先行 + 双栈灰度**
- 交付物：
  1. Go MVP（兼容现有 CLI 与核心工具语义）
  2. 契约测试集（从现有 Python 测试提炼）
  3. 切换与回退手册

如果你希望，我可以下一步直接给出一版“Go 项目脚手架 + 核心接口骨架（含 package 结构和关键函数签名）”，用于立即开工。