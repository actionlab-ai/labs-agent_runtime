# 下一步

## 已完成

- `/v1/runs` 通用 runtime
- `/v1/workflows/project-kickoff` 固定直跑 `novel-project-kickoff`
- `/v1/workflows/project-kernel` 固定直跑 `novel-emotional-core`
- `cmd/novelrt` 已瘦身为入口，HTTP 层已下沉到 `internal/httpapi`
- `internal/httpapi` 已引入 service 层，后续可以继续把更多业务编排从 route 下沉
- `kickoff` 固定回写：
  - `project_brief`
  - `reader_contract`
  - `style_guide`
  - `taboo`
- `kernel` 固定回写：
  - `novel_core`

## P0

1. 把 `world_power` 也固定成明确文档产物
   - `world_rules`
   - `power_system`
   - `mainline`
   - `current_state`

2. 补“未 discover 先误调用”的自修复提示
   - 这是 `/v1/runs` 的稳定性问题
   - 和固定 workflow 无关，但仍然重要

3. 给 workflow run 增加 step 级持久化记录

## P1

1. 做 `chapter-draft` 固定 workflow
2. 打磨 `novel-idea-bootstrap`
3. 明确前端固定调用链

## 暂缓

- 跨轮 discovered memory
  - 等 mem 系统稳定后再做
