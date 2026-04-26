# Novel Agent Runtime

这是当前 `backend` 的总入口说明。

这套 runtime 现在已经不是最早的“`tool_search + skill_call` 两段式分发器”了，而是一套更接近 `novelcode` 思路的 `deferred skill discovery` 运行时：

1. 启动时只加载 skill metadata，不加载完整 `SKILL.md` 正文。
2. 首轮默认只暴露 `tool_search`。
3. `tool_search` 会返回：
   - 查询解析结果
   - 排序后的 skill 命中
   - `tool_reference_like` 激活对象
   - 本轮激活窗口
   - retained discovered-skill pool
4. 下一轮只暴露：
   - `tool_search`
   - retained skill tools
   - `skill_call` 兼容兜底工具
5. 如果 skill 在 frontmatter 里声明了 `tool_input_schema`，激活后的 skill tool 会直接暴露结构化输入 schema，而不是只给一个 `task:string`。
6. skill executor 内部现在也能走本地工具回合：
   - `Read`
   - `Write`
   - `Edit`
   - `Glob`
   - `Bash`
   - `PowerShell`
7. skill 真执行成功后，默认直接把 skill 输出返回给用户；如果 skill 已把内容落成文档，则返回的是简短文件摘要而不是整篇正文。
8. router 和 skill executor 每一轮在真正发模型前，都会先生成 request assembly，并把预处理后的 messages / tools / prompt / final chat request 落盘。

## 当前状态

截至 `2026-04-27`，当前阶段可以概括为：

- tool layer 主链已经跑通
- DeepSeek 实测调通
- `tool_search -> activation -> retained pool -> structured skill tool` 已落地
- `Read / Write / Edit / Glob / Bash / PowerShell` 已接入 skill executor
- `router / skill executor assembly -> final request -> response analysis` 已落地
- 小说产物默认可以落到 [docs/08-generated-drafts](../docs/08-generated-drafts/README.md)
- `opening_v1` 输出 contract 已开始落地
- 跨轮 discovered memory 还没做，等 mem 系统稳定后再推进
- 小说 skill 库的大规模扩展还没开始，当前重心仍然是把工具调用层磨到很稳

更详细的进度和文档导航见：
[docs/README.md](docs/README.md)

## 常用命令

```bash
cd backend
go build -o novelrt ./cmd/novelrt
```

列出已加载 skills：

```bash
./novelrt -config config.yaml -list-skills
```

只做本地检索，不请求模型：

```bash
./novelrt -config config.yaml -input "+forensic urban opening" -dry-run
```

正常运行：

```bash
./novelrt -config config.yaml -input "帮我写一个都市异能悬疑开篇，主角能从尸检报告里看出隐藏信息"
```

调试模式：

```bash
./novelrt -config config.deepseek-debug.yaml -input "帮我写一个都市异能悬疑开篇" -debug
```

## 新的落盘产物

现在一次真实 run 里，除了原来的 raw request / raw response，还会多出：

- `router/round-XX-assembly.json`
- `router/round-XX-assembly.md`
- `router/round-XX-response-analysis.json`
- `skill-calls/<skill>/round-XX-assembly.json`
- `skill-calls/<skill>/round-XX-assembly.md`
- `skill-calls/<skill>/round-XX-response-analysis.json`

如果你想直接看“这一轮模型到底看到了什么、为什么会调这个工具”，先看这些文件。

## 配置提醒

- `model.api_key_env` 填的是“环境变量名”，不是裸密钥
- 当前正式默认配置使用 `DEEPSEEK_API_KEY`
- `config.deepseek-debug.yaml` 只建议调试时使用
- 当前默认 workspace root 是仓库根目录，默认文档输出目录是 [docs/08-generated-drafts](../docs/08-generated-drafts/README.md)

## 推荐阅读顺序

1. [docs/README.md](docs/README.md)
2. [docs/CONFIG.md](docs/CONFIG.md)
3. [docs/FULL_REQUEST_FLOW.md](docs/FULL_REQUEST_FLOW.md)
4. [docs/IMPLEMENTATION.md](docs/IMPLEMENTATION.md)
5. [docs/FILE_TOOLS_AND_DOCUMENT_OUTPUT.md](docs/FILE_TOOLS_AND_DOCUMENT_OUTPUT.md)
6. [docs/SEARCH_PIPELINE.md](docs/SEARCH_PIPELINE.md)
7. [docs/TOOL_SEARCH_VS_NOVELCODE.md](docs/TOOL_SEARCH_VS_NOVELCODE.md)
8. [docs/NEXT_STEPS.md](docs/NEXT_STEPS.md)
