# 生成稿目录

这里用于存放由 `Novel Agent Runtime` 中的小说 skills 自动生成或自动更新的 markdown 文档。

当前约定：

- 技能默认优先把小说产物写到这里
- 新小说项目应先通过 HTTP `POST /v1/projects` 或 `novelrt -create-project "<项目名>"` 创建 PostgreSQL 项目记录
- 后续运行加 `project` 或 `-project "<project-id-or-name>"`，runtime 会从 PostgreSQL 读取项目文档作为当前小说上下文
- 这里放的是运行产物和写作草稿，不是系统设计文档
- 项目状态的 source of truth 是 PostgreSQL，不是这个目录
- 如果用户显式指定了 `document_path`，skill 也可以写到别的位置
