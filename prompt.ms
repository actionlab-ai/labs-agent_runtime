下面是我的 Golang 仓库合并源码。

我的目标不是让你泛泛总结，而是基于现有代码做工程化改造。

我要实现三个核心能力：

1. Tool Search
   - runtime 可以检索本地 skills/tools
   - 返回 tool 名称、描述、参数 schema、入口文件等
   - 给模型作为 tool calling 可用工具列表

2. Golang Runtime
   - 使用 viper 读取 YAML 配置
   - 配置里包含模型 id、base_url、api_key、context_length、tool_calling 开关等
   - runtime 接收用户问题
   - 构造 OpenAI-compatible chat completion 请求
   - 支持 tool calling
   - 判断模型是否要求调用 skill/tool
   - 调用后把结果继续喂给模型
   - 最终返回结果

3. 写工作室写网文的 Skills 一群skill 实现小说创作的工程化解决方案


请你按以下方式输出：

第一步：分析当前仓库结构  
第二步：指出应该新增/修改哪些文件  
第三步：给出模块设计  
第四步：给出接口设计  
第五步：给出最小可运行版本代码  
第六步：给出运行命令  
第七步：给出后续工程化优化方向  