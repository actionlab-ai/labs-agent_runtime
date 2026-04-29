# 网文初始化 Skill 调用实例

这份文件给出正式开书初始化链路的 `skill-session` 调用样例。

统一入口：

```http
POST /v1/skill-sessions
Content-Type: application/json
```

统一继续入口：

```http
POST /v1/skill-sessions/{session_id}/turns
Content-Type: application/json
```

约定：

- `project` 替换成你的项目 ID。
- `model` 替换成你的模型配置 ID；如果已经设置 default model，可以省略。
- 如果返回 `session.status = "needs_input"`，不要重开任务，用 `POST /v1/skill-sessions/{id}/turns` 继续。
- 每个 skill 需要的上游文档由 `project_document_policy` 自动注入，不要手动拼文件路径。

## 0. 创建项目

```bash
curl -X POST http://127.0.0.1:8080/v1/projects \
  -H "Content-Type: application/json" \
  -d '{
    "id": "urban-rebirth",
    "name": "都市重生",
    "description": "都市重生安全感成长流",
    "storage_provider": "filesystem",
    "storage_prefix": "urban-rebirth",
    "metadata": {}
  }'
```

## 1. 创建小说情感内核：`novel-emotional-core`

输出文档：

```text
novel_core
```

```bash
curl -X POST http://127.0.0.1:8080/v1/skill-sessions \
  -H "Content-Type: application/json" \
  -d '{
    "project": "urban-rebirth",
    "model": "deepseek-flash",
    "skill_id": "novel-emotional-core",
    "input": "先创建小说情感内核。题材是都市重生，主角中年销售，被债务、KPI 和家庭责任压着。我想写安全感、尊严和持续变强。如果信息不足就 AskHuman，不要直接瞎补。",
    "arguments": {
      "task": "创建小说情感内核",
      "genre": "都市重生",
      "target_reader": "被现实压力压住、想看中年人翻身和重新被尊重的男频读者",
      "protagonist_seed": "中年销售，债务和 KPI 双重压迫，家庭责任重",
      "emotional_need": "夺回尊严、稳定安全感、被重新看见",
      "pressure_source": "债务、职场规则、年龄焦虑、家庭责任",
      "payoff": "稳步翻身、建立安全区、让曾经轻视他的人重新评价他",
      "avoid": "不要无脑暴富，不要纯复仇黑化，不要开局无敌",
      "document_kind": "novel_core"
    },
    "debug": true
  }'
```

## 2. 创建世界压力引擎：`novel-world-engine`

自动注入：

```text
novel_core
```

输出文档：

```text
world_engine
```

```bash
curl -X POST http://127.0.0.1:8080/v1/skill-sessions \
  -H "Content-Type: application/json" \
  -d '{
    "project": "urban-rebirth",
    "model": "deepseek-flash",
    "skill_id": "novel-world-engine",
    "input": "基于已经保存的小说情感内核，设计这个世界如何在遵循 core 的前提下持续制造冲突。缺信息就 AskHuman。",
    "arguments": {
      "task": "设计世界压力引擎",
      "genre": "都市重生",
      "pressure_axis": "资源垄断、职场资格、年龄歧视、债务信用",
      "resource_system": "客户资源、信用额度、行业资格、稳定现金流、关键情报",
      "progression_seed": "主角靠重生后的经验和规则漏洞，逐步拿回资格、资源和安全区",
      "avoid": "不要写成单纯商业开挂，不要让重生经验直接碾压一切",
      "document_kind": "world_engine"
    },
    "debug": true
  }'
```

## 3. 创建读者契约、项目简报、禁区：`novel-reader-contract`

自动注入：

```text
novel_core
world_engine
```

输出文档：

```text
reader_contract
project_brief
taboo
```

```bash
curl -X POST http://127.0.0.1:8080/v1/skill-sessions \
  -H "Content-Type: application/json" \
  -d '{
    "project": "urban-rebirth",
    "model": "deepseek-flash",
    "skill_id": "novel-reader-contract",
    "input": "基于 novel_core 和 world_engine，生成读者承诺、项目简报和禁区。重点明确前三章钩子、前十章回报节奏、目标读者和不能写偏的方向。",
    "arguments": {
      "task": "生成读者契约与项目定调",
      "platform": "番茄/免费文气质，节奏直给，情绪回报要快",
      "target_reader": "现实压力很重、想看稳步翻身和安全感建立的男频读者",
      "genre": "都市重生安全感成长流",
      "commercial_angle": "中年低谷主角靠经验和规则漏洞重建现金流、安全区和尊严",
      "opening_hook": "主角重生回到债务和 KPI 爆雷前夜，发现一个可以提前止血的规则漏洞",
      "first_10_chapters_payoff": "止住第一次损失，拿回一个关键客户，第一次让轻视他的人误判失败",
      "avoid": "不要开局暴富，不要让家庭线狗血化，不要把职场写成无脑反派集合"
    },
    "debug": true
  }'
```

## 4. 创建角色台账：`novel-character-pressure-engine`

自动注入：

```text
novel_core
world_engine
reader_contract
project_brief
taboo
```

输出文档：

```text
character_cast
```

```bash
curl -X POST http://127.0.0.1:8080/v1/skill-sessions \
  -H "Content-Type: application/json" \
  -d '{
    "project": "urban-rebirth",
    "model": "deepseek-flash",
    "skill_id": "novel-character-pressure-engine",
    "input": "基于已有项目资产，建立主角、核心压迫者、关键配角、关系变化和信息差台账。不要写人物百科，要让每个人能制造剧情压力。",
    "arguments": {
      "task": "生成角色台账",
      "protagonist_seed": "中年销售，重生前被债务和职场淘汰，重生后想先保住家庭和现金流",
      "antagonist_seed": "掌握客户资源和考核权的上级、利用规则吃人的同事、催债压力方",
      "supporting_cast_seed": "家人、老客户、年轻同事、曾经误判主角的人",
      "relationship_seed": "主角知道未来风险，别人不知道；上级误判主角已经无路可退",
      "avoid": "不要脸谱化反派，不要让家人成为纯拖累，不要强行暧昧线",
      "document_kind": "character_cast"
    },
    "debug": true
  }'
```

## 5. 创建世界规则和能力体系：`novel-rules-power-engine`

自动注入：

```text
novel_core
world_engine
reader_contract
character_cast
```

输出文档：

```text
world_rules
power_system
```

```bash
curl -X POST http://127.0.0.1:8080/v1/skill-sessions \
  -H "Content-Type: application/json" \
  -d '{
    "project": "urban-rebirth",
    "model": "deepseek-flash",
    "skill_id": "novel-rules-power-engine",
    "input": "基于已有项目资产，设计世界规则和主角能力体系。重点是规则、边界、代价、升级资源和爽点兑现，不要写百科。",
    "arguments": {
      "task": "生成世界规则与能力体系",
      "rule_seed": "现实职场、债务信用、客户资源、行业资格构成世界硬规则",
      "power_seed": "重生记忆不是外挂碾压，而是风险预判和规则漏洞识别能力",
      "progression_seed": "从止损、拿客户、建现金流，到建立自己的安全区和资源网络",
      "cost_seed": "每次提前行动都会消耗信任、现金流、人情和信息优势",
      "resource_seed": "客户信任、现金流、信用额度、行业情报、关键时间窗口",
      "avoid": "不要系统面板，不要直接预知彩票暴富，不要无代价商业开挂"
    },
    "debug": true
  }'
```

## 6. 创建主线规划：`novel-mainline-engine`

自动注入：

```text
novel_core
reader_contract
world_engine
character_cast
world_rules
power_system
taboo
```

输出文档：

```text
mainline
```

```bash
curl -X POST http://127.0.0.1:8080/v1/skill-sessions \
  -H "Content-Type: application/json" \
  -d '{
    "project": "urban-rebirth",
    "model": "deepseek-flash",
    "skill_id": "novel-mainline-engine",
    "input": "基于已有项目资产，规划第一卷核心冲突、前三章钩子、前 15 章推进和 30-50 章中期方向。",
    "arguments": {
      "task": "生成主线规划",
      "volume_goal": "第一卷目标：主角从债务和职场双重崩盘前夜，建立第一条稳定现金流和可依赖安全区",
      "first_3_chapters": "重生回爆雷前夜，发现第一个止损窗口，第一次利用规则漏洞反制误判",
      "first_15_chapters": "止损、保客户、反误判、拿回资格、建立小型资源闭环",
      "midterm_direction": "从自救进入主动布局，建立自己的客户池和信用网络",
      "antagonist_arc": "上级和同事先误判主角无路可走，再发现主角开始绕过旧规则",
      "avoid": "不要过早开公司碾压，不要第一卷就解决所有债务"
    },
    "debug": true
  }'
```

## 7. 可选：创建风格指南：`novel-style-guide`

自动注入：

```text
novel_core
reader_contract
project_brief
```

输出文档：

```text
style_guide
```

```bash
curl -X POST http://127.0.0.1:8080/v1/skill-sessions \
  -H "Content-Type: application/json" \
  -d '{
    "project": "urban-rebirth",
    "model": "deepseek-flash",
    "skill_id": "novel-style-guide",
    "input": "基于项目定位和读者承诺，生成正文文风指南。重点控制叙述距离、对白、节奏、爽点落句和去 AI 味。",
    "arguments": {
      "task": "生成风格指南",
      "platform_style": "番茄免费文，节奏直给，但不要口水化",
      "narration_distance": "贴近主角，但不要过度解释心理",
      "dialogue_preference": "对白生活化，有职场压迫感，不要鸡汤",
      "avoid": "不要AI腔，不要排比式总结，不要每段都解释情绪"
    },
    "debug": true
  }'
```

## 8. 创建开篇执行包：`novel-opening-package`

自动注入：

```text
novel_core
reader_contract
world_engine
character_cast
world_rules
power_system
mainline
taboo
style_guide
```

输出文档：

```text
current_state
```

```bash
curl -X POST http://127.0.0.1:8080/v1/skill-sessions \
  -H "Content-Type: application/json" \
  -d '{
    "project": "urban-rebirth",
    "model": "deepseek-flash",
    "skill_id": "novel-opening-package",
    "input": "基于已有项目资产，生成第一章开篇执行包和初始 current_state。不要写正文，只给第一章可执行约束。",
    "arguments": {
      "task": "生成开篇执行包",
      "opening_scene": "主角在深夜办公室醒来，发现自己回到债务和项目爆雷前夜",
      "first_pressure_event": "上级逼他签下会让自己背锅的客户确认单",
      "chapter_goal": "让主角意识到重生窗口，并完成第一次小止损",
      "ending_hook": "主角发现真正导致前世崩盘的关键客户明天会提前出现",
      "style_preference": "开局压迫直给，少解释，多用动作和细节体现中年人的疲惫和清醒",
      "avoid": "不要第一章暴富，不要大段回忆，不要直接解释完整未来"
    },
    "debug": true
  }'
```

## 9. 章后更新状态：`novel-continuity-snapshot`

自动注入：

```text
novel_core
mainline
current_state
```

输出文档：

```text
current_state
```

```bash
curl -X POST http://127.0.0.1:8080/v1/skill-sessions \
  -H "Content-Type: application/json" \
  -d '{
    "project": "urban-rebirth",
    "model": "deepseek-flash",
    "skill_id": "novel-continuity-snapshot",
    "input": "根据第一章正文或摘要，更新 current_state。只记录已发生事实、人物状态、信息差和下一章约束，不要续写正文。",
    "arguments": {
      "task": "章后更新当前状态",
      "chapter_summary": "第一章中，主角在办公室重生，识破上级让他背锅的确认单，暂时拖住签字，并发现关键客户会在第二天提前出现。",
      "changed_facts": "主角已确认自己回到爆雷前夜；确认单暂未签；上级仍以为主角会屈服。",
      "character_changes": "主角从崩溃转为清醒；上级对主角仍保持轻视。",
      "next_constraints": "第二章必须接关键客户提前出现；主角不能直接摊牌重生；仍要保留现金流压力。"
    },
    "debug": true
  }'
```

## 10. needs_input 后继续 session

如果任意一步返回：

```json
{
  "session": {
    "id": "ss_xxx",
    "status": "needs_input",
    "ask_human": {
      "questions": []
    }
  }
}
```

用同一个 `session.id` 继续：

```bash
curl -X POST http://127.0.0.1:8080/v1/skill-sessions/ss_xxx/turns \
  -H "Content-Type: application/json" \
  -d '{
    "input": "我补充：目标读者是被现实压力压住、想看稳步翻身和安全感建立的中年男读者。前三章先打压迫和第一次止损。",
    "answers": {
      "target_reader": "被现实压力压住、想看稳步翻身和安全感建立的中年男读者",
      "opening_hook": "债务和职场背锅压力下的第一次止损",
      "first_10_chapters_payoff": "止损、拿回客户、反制误判、建立第一条现金流"
    },
    "notes": "不要开局暴富，不要写成纯复仇。"
  }'
```

## 11. 推荐执行顺序

```text
1. novel-emotional-core
2. novel-world-engine
3. novel-reader-contract
4. novel-character-pressure-engine
5. novel-rules-power-engine
6. novel-mainline-engine
7. novel-style-guide
8. novel-opening-package
9. novel-continuity-snapshot
```

`novel-style-guide` 可选，但建议在 `novel-opening-package` 前执行。
