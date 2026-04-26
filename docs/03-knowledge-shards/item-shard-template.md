# Item Shard：物品知识文件

## 目标

记录物品的外观、来源、出现章节、当前归属、已确认事实和不能写错的细节。

## 模板

```yaml
id: item.old_copper_token
type: item
name: 旧铜牌
aliases:
  - 铜牌
  - 半枚铜牌
  - 巡牌

canonical_status: active

summary: >
  主角在祠堂后院泥沟中发现的半枚旧铜牌。
  铜牌边缘有火烧缺口，表面有半个残缺的“巡”字。
  目前只确认它与巡查体系有关，尚未确认原主人。

first_seen:
  chapter: ch0312
  scene: 祠堂后院
  anchor: ch0312#p087
  summary: 主角擦掉泥污后，看见半个残缺的“巡”字。
  evidence_level: raw_text_verified

latest_seen:
  chapter: ch0478
  scene: 夜查旧井
  anchor: ch0478#p142
  summary: 主角再次摸到铜牌边缘，确认有火烧缺口。
  evidence_level: raw_text_verified

current_owner:
  holder: character.chen_jianchuan
  confidence: high
  source: ch0478#p142

appearances:
  - chapter: ch0312
    anchor: ch0312#p087
    role: first_appearance
    event: 主角第一次发现旧铜牌
    facts:
      - 半枚铜牌
      - 泥污覆盖
      - 有半个残缺的“巡”字
      - 江叔在场，但没有解释来源

current_canon:
  confirmed:
    - 铜牌是半枚，不完整
    - 铜牌上只有半个残缺的“巡”字
    - 铜牌边缘有火烧痕迹
    - 主角知道它可能与巡查体系有关
  unconfirmed:
    - 铜牌原主人是谁
    - 是否属于巡查司正式信物

usage_rules:
  can_write:
    - 可以写主角摸出旧铜牌确认“巡”字残痕
    - 可以写主角怀疑它和巡查体系有关
  cannot_write:
    - 不能写成完整“巡查司”三个字
    - 不能写主角已经知道铜牌主人

related:
  plot_threads:
    - plot.missing_patrol_form
    - plot.old_well_mark
  locations:
    - location.ancestral_hall
    - location.old_well
  characters:
    - character.chen_jianchuan
    - character.jiang_shu

retrieval_policy:
  default_strategy: origin_first_then_recent_confirmation
  raw_text_required: true
```

## 检索行为

当用户或 Master 提到“旧铜牌”：

```text
knowledge.search 命中 item shard
→ 读 first_seen 和 latest_seen
→ 根据问题类型读原文窗口
→ 生成 fact card
```
