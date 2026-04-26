# Character Shard：人物知识文件

## 目标

记录人物的当前状态、知识状态、动机、口吻、关系和关键出场。

人物 shard 重点不是外貌小传，而是：

```text
他现在想要什么？
他知道什么？
他不知道什么？
他如何压迫别人？
他什么时候会失态？
读者期待他被如何回收？
```

## 模板

```yaml
id: character.ma_guanshi
type: character
name: 马管事
aliases:
  - 马管事
  - 老马
  - 马账房

role: early_arc_antagonist

first_seen:
  chapter: ch0003
  anchor: ch0003#p044

latest_seen:
  chapter: ch0992
  anchor: ch0992#p177

current_state:
  location: 外城祠堂
  goal: 压住主角，封住巡查单缺页
  resources:
    - 祠堂账本
    - 族老支持
  vulnerability:
    - 账本日期漏洞
    - 私吞巡查银嫌疑

knowledge_state:
  knows:
    - 主角接触过巡查单
  suspects:
    - 主角可能发现了账本异常
  does_not_know:
    - 主角已经将账本页码和井底刻痕联系起来

relationship_to_protagonist:
  current: 敌对压迫
  leverage_over_protagonist:
    - 族规
    - 账本
    - 祠堂秩序

voice_profile:
  typical:
    - 喜欢用“规矩”压人
    - 不直接咆哮，常把威胁包装成程序
  when_panicked:
    - 句子变短
    - 会反复强调“按规矩”
  forbidden:
    - 不能说得像热血莽夫
    - 不能当场承认自己造假

key_appearances:
  - chapter: ch0045
    anchor: ch0045#p031
    role: establishes_pressure_style
    summary: 第一次用祠堂规矩压主角
  - chapter: ch0992
    anchor: ch0992#p177
    role: first_loss_of_control
    summary: 被主角反压后第一次失态

reader_promise:
  hated_for:
    - 用规矩羞辱主角
    - 借族老压人
  expected_payoff:
    - 公开吃瘪
    - 账本问题暴露

retrieval_policy:
  current_state: recent_first
  voice: representative_sample
  origin: origin_first
```

## 使用场景

写该角色对白时，读 voice_profile，必要时读 key_appearances 原文窗口，不要只看人物小传。

判断该角色是否知道某秘密时，查 knowledge_state，再 recent_first 回原文确认。
