# 读者体验、AI味、连续性质量系统

## 1. Reader Experience Audit

不审文学性，审：

```text
读者会不会继续看？
情绪有没有到位？
爽点有没有兑现？
主角有没有主动性？
设定是否压过剧情？
章尾有没有追读钩子？
```

评分维度：

```yaml
scores:
  opening_grab: 0-10
  protagonist_pressure: 0-10
  protagonist_agency: 0-10
  payoff_delivery: 0-10
  scene_momentum: 0-10
  exposition_control: 0-10
  ending_hook: 0-10
  ai_tone_risk: 0-10
```

高风险弃书点：

```text
开头 500 字无人物冲突
连续大段环境描写
主角被动观察太久
设定解释超过场景需要
反派压迫无代价
主角没有任何判断或动作
章尾只是“他离开了”
情绪用抽象词说明而非动作表达
```

## 2. AI 味手术

AI 味常见于空氛围、抽象情绪、设定说明书、模板哲理句、假口语、章尾套话、所有角色过度理性。

### 空环境手术

AI 写：

```text
夜色很深。空气中弥漫着潮湿的气味。
```

手术：

```text
陈见川被按在祠堂门槛上时，雨水正顺着他的下巴往下滴。
```

### 抽象情绪手术

AI 写：

```text
他心中充满了愤怒和不甘。
```

手术：

```text
他没吵，只把欠条折好，塞进怀里。“签可以。”他抬眼看向族老，“但先把我爹的名字，从死人账上划掉。”
```

### 设定说明书手术

AI 写：

```text
这个世界分为五大城区，外城人地位最低。
```

手术：

```text
门槛不高，只到膝盖。可外城人进祠堂，必须跪着过。
```

## 3. Continuity Audit

硬连续性：

```text
时间线
地点
物品归属
伤势状态
能力限制
资源消耗
角色是否在场
```

软连续性：

```text
人物动机
情绪延续
上一章钩子承接
读者期待承接
主线压力是否持续
支线是否突然消失
```

输出格式：

```yaml
continuity_audit:
  status: pass | warning | fail
  hard_issues:
    - type:
      location:
      problem:
      evidence:
      repair:
  soft_issues:
    - type:
      location:
      problem:
      why_it_matters:
      repair:
  required_fact_checks:
    - query:
      recommended_strategy:
```
