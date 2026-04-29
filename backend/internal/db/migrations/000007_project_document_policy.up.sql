INSERT INTO app_settings (key, value)
VALUES (
  'project_document_policy',
  $policy$
{
  "documents": [
    {"kind": "novel_core", "title": "小说情感内核", "priority": 0},
    {"kind": "project_brief", "title": "项目简报", "priority": 10},
    {"kind": "reader_contract", "title": "读者承诺", "priority": 20},
    {"kind": "style_guide", "title": "风格指南", "priority": 30},
    {"kind": "taboo", "title": "禁区与避坑", "priority": 40},
    {"kind": "world_engine", "title": "小说世界压力引擎", "priority": 50},
    {"kind": "character_cast", "title": "角色台账", "priority": 55},
    {"kind": "world_rules", "title": "世界规则", "priority": 60},
    {"kind": "power_system", "title": "能力体系", "priority": 70},
    {"kind": "factions", "title": "势力关系", "priority": 80},
    {"kind": "locations", "title": "地点设定", "priority": 90},
    {"kind": "mainline", "title": "主线规划", "priority": 100},
    {"kind": "current_state", "title": "当前状态", "priority": 110}
  ],
  "skill_documents": {
    "novel-world-engine": ["novel_core"],
    "novel-reader-contract": ["novel_core", "world_engine"],
    "novel-character-pressure-engine": ["novel_core", "world_engine", "reader_contract", "project_brief", "taboo"],
    "novel-rules-power-engine": ["novel_core", "world_engine", "reader_contract", "character_cast"],
    "novel-mainline-engine": ["novel_core", "reader_contract", "world_engine", "character_cast", "world_rules", "power_system", "taboo"],
    "novel-opening-package": ["novel_core", "reader_contract", "world_engine", "character_cast", "world_rules", "power_system", "mainline", "taboo", "style_guide"],
    "novel-style-guide": ["novel_core", "reader_contract", "project_brief"],
    "novel-continuity-snapshot": ["novel_core", "mainline", "current_state"]
  }
}
$policy$
)
ON CONFLICT (key) DO NOTHING;
