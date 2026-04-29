package project

import (
	"encoding/json"
	"sort"
	"strings"
)

const DocumentPolicySettingKey = "project_document_policy"

type DocumentPolicy struct {
	Documents      []DocumentPolicyItem `json:"documents"`
	SkillDocuments map[string][]string  `json:"skill_documents,omitempty"`
}

type DocumentPolicyItem struct {
	Kind     string `json:"kind"`
	Title    string `json:"title"`
	Priority int    `json:"priority"`
}

func DefaultDocumentPolicy() DocumentPolicy {
	return DocumentPolicy{
		Documents: []DocumentPolicyItem{
			{Kind: "novel_core", Title: "小说情感内核", Priority: 0},
			{Kind: "project_brief", Title: "项目简报", Priority: 10},
			{Kind: "reader_contract", Title: "读者承诺", Priority: 20},
			{Kind: "style_guide", Title: "风格指南", Priority: 30},
			{Kind: "taboo", Title: "禁区与避坑", Priority: 40},
			{Kind: "world_engine", Title: "小说世界压力引擎", Priority: 50},
			{Kind: "character_cast", Title: "角色台账", Priority: 55},
			{Kind: "world_rules", Title: "世界规则", Priority: 60},
			{Kind: "power_system", Title: "能力体系", Priority: 70},
			{Kind: "factions", Title: "势力关系", Priority: 80},
			{Kind: "locations", Title: "地点设定", Priority: 90},
			{Kind: "mainline", Title: "主线规划", Priority: 100},
			{Kind: "current_state", Title: "当前状态", Priority: 110},
		},
		SkillDocuments: map[string][]string{
			"novel-world-engine":              []string{"novel_core"},
			"novel-reader-contract":           []string{"novel_core", "world_engine"},
			"novel-character-pressure-engine": []string{"novel_core", "world_engine", "reader_contract", "project_brief", "taboo"},
			"novel-rules-power-engine":        []string{"novel_core", "world_engine", "reader_contract", "character_cast"},
			"novel-mainline-engine":           []string{"novel_core", "reader_contract", "world_engine", "character_cast", "world_rules", "power_system", "taboo"},
			"novel-opening-package":           []string{"novel_core", "reader_contract", "world_engine", "character_cast", "world_rules", "power_system", "mainline", "taboo", "style_guide"},
			"novel-style-guide":               []string{"novel_core", "reader_contract", "project_brief"},
			"novel-continuity-snapshot":       []string{"novel_core", "mainline", "current_state"},
		},
	}
}

func ParseDocumentPolicy(raw string) (DocumentPolicy, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return DefaultDocumentPolicy(), nil
	}
	var policy DocumentPolicy
	if err := json.Unmarshal([]byte(raw), &policy); err != nil {
		return DocumentPolicy{}, err
	}
	return policy.Normalize(), nil
}

func (p DocumentPolicy) Normalize() DocumentPolicy {
	defaults := DefaultDocumentPolicy()
	if len(p.Documents) == 0 {
		p.Documents = defaults.Documents
	}
	seen := map[string]bool{}
	var docs []DocumentPolicyItem
	for i, item := range p.Documents {
		item.Kind = normalizeDocumentKind(item.Kind)
		item.Title = strings.TrimSpace(item.Title)
		if item.Kind == "" || seen[item.Kind] {
			continue
		}
		if item.Title == "" {
			item.Title = item.Kind
		}
		if item.Priority == 0 && i > 0 && item.Kind != NovelCoreKind {
			item.Priority = i * 10
		}
		seen[item.Kind] = true
		docs = append(docs, item)
	}
	sort.SliceStable(docs, func(i, j int) bool {
		if docs[i].Priority != docs[j].Priority {
			return docs[i].Priority < docs[j].Priority
		}
		return docs[i].Kind < docs[j].Kind
	})
	p.Documents = docs
	if p.SkillDocuments == nil {
		p.SkillDocuments = map[string][]string{}
	} else {
		normalized := map[string][]string{}
		for skillID, kinds := range p.SkillDocuments {
			skillID = strings.TrimSpace(skillID)
			if skillID == "" {
				continue
			}
			normalized[skillID] = uniqueKinds(kinds)
		}
		p.SkillDocuments = normalized
	}
	return p
}

func (p DocumentPolicy) IsPersistable(kind string) bool {
	kind = normalizeDocumentKind(kind)
	for _, item := range p.Normalize().Documents {
		if item.Kind == kind {
			return true
		}
	}
	return false
}

func (p DocumentPolicy) Title(kind string) string {
	kind = normalizeDocumentKind(kind)
	for _, item := range p.Normalize().Documents {
		if item.Kind == kind {
			return item.Title
		}
	}
	return kind
}

func (p DocumentPolicy) Kinds() []string {
	p = p.Normalize()
	out := make([]string, 0, len(p.Documents))
	for _, item := range p.Documents {
		out = append(out, item.Kind)
	}
	return out
}

func (p DocumentPolicy) Priority(kind string) int {
	kind = normalizeDocumentKind(kind)
	for _, item := range p.Normalize().Documents {
		if item.Kind == kind {
			return item.Priority
		}
	}
	return 1000
}

func (p DocumentPolicy) RequiredDocumentsForSkill(skillID string) []string {
	p = p.Normalize()
	return uniqueKinds(p.SkillDocuments[strings.TrimSpace(skillID)])
}

func uniqueKinds(kinds []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, kind := range kinds {
		kind = normalizeDocumentKind(kind)
		if kind == "" || seen[kind] {
			continue
		}
		seen[kind] = true
		out = append(out, kind)
	}
	return out
}
