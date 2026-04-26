# Search Pipeline

这份文档详细说明当前 `tool_search` 在 Go runtime 里到底怎么工作。

## 1. 查询解析

当前 search 分两种模式：

- `select`
  - 例如：`select:webnovel-opening-sniper`
- `keyword`
  - 例如：`+forensic urban opening`

对 `keyword` 模式，runtime 会拆出三组词：

- `required_terms`
  - 所有 `+` 开头的必含词
- `optional_terms`
  - 剩余普通词
- `scoring_terms`
  - 有必含词时：`required + optional`
  - 没必含词时：全部 query terms

此外还会注入一批已知意图词，比如：

- `opening`
- `hook`
- `first chapter`
- `开头`
- `开篇`
- `黄金600`
- `都市`
- `异能`
- `尸检`

## 2. 精确命中快捷路径

进入正常打分前，runtime 先检查 raw query 是否精确命中：

- skill id
- skill name
- alias

如果命中，就直接返回 `exact=true`。

这部分和 `novelcode` 的 exact-name fast path 是同一思路。

## 3. 必含词预过滤

如果 query 里有 `+required` 词，所有不满足这些词的 skill 会先被过滤掉。

必含词可以命中：

- `id`
- `name`
- `aliases`
- `tags`
- `search_hint`
- `when_to_use`
- `description`

这一步的作用是先把候选集收窄，再进入打分。

## 4. 字段加权打分

剩下的 skill 会按 term-by-term 方式打分。

高权重字段：

- `id` 精确部分命中
- `name` 精确部分命中
- `alias` 精确部分命中
- `tag` 精确部分命中
- `search_hint`

中等权重：

- `id/name/alias/tag` 的部分命中
- full-name fallback

低权重：

- `when_to_use`
- `description`

对于“明显是开篇请求”的 query，还会额外加一层 opening-intent bonus。

## 5. activation window

search 排序不是最终结果。

排序后的 top hits 会再进入一层 `score-window` 策略：

1. 首个命中一定保留
2. 后续命中只有仍然处在分数窗口里才会扩进来
3. 总数不会超过 `max_activated_skills`

分数窗口是：

```text
max(activation_min_score, first_score * activation_score_ratio)
```

所以当前 runtime 不会把所有 hit 都暴露给模型，只会暴露“当前仍然很像”的那小簇 skill。

## 6. retained discovered-skill pool

activation window 还不是最后一层。

它会再和旧的 retained pool 合并：

1. 新窗口优先
2. 旧 retained 项目跟在后面
3. 按 `skill_id` 去重
4. 截断到 `max_retained_skills`

这样 runtime 会形成两层状态：

- fresh window
  - 这次 query 新带出来的 callable skills
- retained pool
  - 前面已经 discover 过、暂时继续保留的 callable skills

## 7. 重复 search 的识别

每次 search 都会生成 query fingerprint，包含：

- mode
- raw query
- required terms
- optional terms
- scoring terms

如果下一次 search：

- fingerprint 相同
- retained pool 也没变

那么 activation plan 会被标成 `unchanged=true`。

这可以告诉模型：“这次 search 没带出任何新东西”。

## 8. `tool_reference_like`

当前 `tool_search` 的返回不再只是 hit 列表。

它还会返回显式的激活引用对象，例如：

```json
{
  "type": "tool_reference_like",
  "tool_name": "skill_exec_webnovel_opening_sniper_0c2708f0",
  "skill_id": "webnovel-opening-sniper",
  "skill_name": "网文开篇狙击手",
  "contract": "opening_v1",
  "output_contract": "opening_prose_v1",
  "parameters": {
    "type": "object",
    "properties": {
      "task": {"type": "string"},
      "premise": {"type": "string"},
      "protagonist": {"type": "string"}
    },
    "required": ["task"]
  }
}
```

它不是 wire-level 原生 `tool_reference`，但已经是当前 Go runtime 里最接近的等价物。

## 9. skill-specific schema

如果 skill 在 frontmatter 里声明了 `tool_input_schema`，激活后的 skill tool 就不再只有：

```json
{"task":"..."}
```

而是可以直接暴露结构化字段，比如：

- `premise`
- `protagonist`
- `power`
- `setting`
- `must_use`
- `constraints`

这一步很重要，因为它把当前系统从“通用 dispatcher 调 skill”继续往“更像真实工具 schema”推进了一层。

## 10. search 的真实作用

所以今天的 `tool_search` 本质上已经不是：

```text
帮模型挑一个 skill id
```

而是：

```text
分析 query
-> 选出最相关的一小簇 skills
-> 生成 tool_reference_like
-> 重建后续可调用能力面
```

这才是它现在最有价值的地方。
