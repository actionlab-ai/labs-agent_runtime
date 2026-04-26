# 信息差、读者承诺、节奏台账

## 1. Info Gap Ledger：信息差台账

信息差必须明确记录：

```yaml
id: info_gap.missing_patrol_form
topic: 巡查单缺页

reader_knows:
  - 巡查单被人动过
  - 缺页日期对应旧井事件
  - 马管事账本可能有关

protagonist_knows:
  - 巡查单缺页
  - 缺页日期异常
  - 账本页码可对上缺页日期

protagonist_does_not_know:
  - 谁撕掉了缺页
  - 缺页具体记录了什么
  - 幕后势力是谁

ma_guanshi_knows:
  - 巡查单缺页存在风险
  - 主角接触过巡查单

ma_guanshi_does_not_know:
  - 主角已发现账本页码问题
  - 主角已关联井底刻痕

hidden_truth:
  - 缺页记录的是巡查队失踪当晚的真实路线

retrieval_policy:
  default_strategy: recent_first
```

使用规则：

```text
角色只能说他知道的信息。
角色可以误判，但不能凭空知道。
读者知道但主角不知道的信息，可以制造紧张。
主角知道但反派不知道的信息，可以制造爽点。
```

## 2. Reader Promise Ledger：读者承诺 / 爽点账本

```yaml
id: promise.ma_guanshi_first_face_slap
type: reader_promise
emotion: 反击爽
target_reader_feeling: 终于让马管事吃瘪

created_at:
  chapter: ch0003
  reason: 马管事第一次用规矩羞辱主角

accumulated_grievance:
  - chapter: ch0003
    event: 马管事逼主角低头过祠堂门槛
  - chapter: ch0012
    event: 马管事当众污蔑主角偷账本

payoff_status: partially_delivered

delivered:
  - chapter: ch0992
    event: 主角让马管事第一次失态

pending:
  - 马管事公开吃瘪
  - 账本问题暴露
  - 族老开始怀疑马管事

risk:
  - 如果 ch1030 前不兑现公开打脸，读者会觉得拖

payoff_window:
  earliest: ch0990
  ideal: ch1000-ch1020
  latest: ch1030
```

## 3. Pacing Ledger：节奏台账

```yaml
scope: volume_03
range: ch0995-ch0999

recent_chapters:
  ch0995:
    primary_emotion: 压迫
    conflict_level: medium
    payoff: none
    hook: 马管事盯上主角
  ch0996:
    primary_emotion: 调查
    payoff: small_clue
    hook: 账本日期异常
  ch0997:
    primary_emotion: 憋屈
    payoff: none
    hook: 主角被带去祠堂
  ch0998:
    primary_emotion: 对峙
    payoff: partial
    hook: 族老质问主角
  ch0999:
    primary_emotion: 反压前夜
    payoff: delayed
    hook: 主角发现账本缺页

pacing_diagnosis:
  pressure_streak: 5
  chapters_since_clear_payoff: 4
  reader_fatigue_risk: high

recommendation_for_next:
  - ch1000 应该兑现一个明确小爽点
  - 不宜继续纯调查
```

## 4. 审稿问题

```text
是否有人说出了他不该知道的信息？
是否有人忘记了他已经知道的信息？
本章是否兑现了之前承诺？
最近是否压迫太久没有爽点？
支线是否离开太久需要回归？
```
