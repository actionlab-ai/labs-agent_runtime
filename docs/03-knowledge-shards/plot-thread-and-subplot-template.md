# Plot Thread Shard：伏笔 / 主线 / 支线文件

## 目标

记录一条剧情线的起点、状态、推进、可回收窗口和禁止提前揭露的信息。

## 模板

```yaml
id: plot.old_well_mark
type: plot_thread
name: 井底刻痕

status: active # planted | active | ready_to_payoff | paid_off | abandoned
importance: mainline

emotion_function:
  - 悬疑
  - 危机扩大
  - 世界边界打开

planted_at:
  chapter: ch0312
  anchor: ch0312#p102
  summary: 主角发现井底刻痕不像本地势力留下。

latest_development:
  chapter: ch0876
  anchor: ch0876#p210
  summary: 另一处出现同类刻痕，说明这不是偶发事件。

reader_knows:
  - 井底刻痕不是本地势力留下
  - 刻痕与巡查体系可能有关

protagonist_knows:
  - 刻痕异常
  - 刻痕和旧铜牌形制有联系

not_yet_revealed:
  - 刻痕真正来源
  - 幕后势力身份
  - 旧铜牌原主人

mentions:
  - chapter: ch0312
    anchor: ch0312#p102
    role: planted
  - chapter: ch0478
    anchor: ch0478#p155
    role: reinforced
  - chapter: ch0876
    anchor: ch0876#p210
    role: expanded

payoff_window:
  earliest: ch0950
  ideal: ch1000-ch1030
  latest: ch1100

reveal_policy:
  current_allowed_reveal:
    - 刻痕和巡查体系存在联系
  current_forbidden_reveal:
    - 幕后主使是谁
    - 完整组织结构

retrieval_policy:
  default_strategy: origin_recent_bridge
  must_read:
    - ch0312
    - ch0876
```

## 支线回归要求

支线离开太久，回归前要查：

```text
上次出现在哪
当时悬而未决的问题是什么
读者还记不记得
回归是否需要提醒性场景
回归后服务主线还是独立拖节奏
```
