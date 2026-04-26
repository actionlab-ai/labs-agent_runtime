# Evidence Card / Fact Card 规范

## 目标

Fact Card 是 SubAgent 查证后返回给 Master 的高密度事实卡。  
Master 不直接相信模型印象，只相信带证据的 fact card。

## 标准结构

```yaml
fact_card:
  id: fact.ch1000.old_copper_token.mark
  query: 旧铜牌上到底刻了什么字？
  answer: 只有半个残缺的“巡”字，不能写成完整“巡查司”。
  confidence: high

  evidence:
    - source_type: raw_chapter
      chapter: ch0312
      anchor: ch0312#p087
      summary: 主角擦掉泥污后，只看见半个残缺的“巡”字。
      quote_policy: summary_only

    - source_type: raw_chapter
      chapter: ch0478
      anchor: ch0478#p142
      summary: 主角再次确认铜牌边缘有火烧缺口。

  allowed_usage:
    - 可以写主角摸到“巡”字残痕。
    - 可以写主角怀疑它与巡查体系有关。

  forbidden_usage:
    - 不能写成完整“巡查司”三个字。
    - 不能写主角已经知道铜牌原主人。

  unresolved:
    - 铜牌原主人未确认。

  followup_queries:
    - 铜牌与井底刻痕是否同源？
```

## 证据等级

```text
high：至少一个原文窗口确认，且无冲突。
medium：来自 ledger / snapshot，缺原文确认。
low：模型推断，未找到明确出处。
```

## Master 使用规则

Master 在写正文时：

```text
can_write 使用 allowed_usage
must_not_write 使用 forbidden_usage
不确定内容不能强行明确化
```

Fact Card 不是正文，它是决策材料。
