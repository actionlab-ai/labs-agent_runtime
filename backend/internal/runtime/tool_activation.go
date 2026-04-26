package runtime

import (
	"fmt"
	"hash/fnv"
	"sort"
	"strings"

	"novel-agent-runtime/internal/model"
	"novel-agent-runtime/internal/skill"
)

type callableToolReference struct {
	Type           string         `json:"type"`
	ToolName       string         `json:"tool_name"`
	SkillID        string         `json:"skill_id"`
	SkillName      string         `json:"skill_name,omitempty"`
	Description    string         `json:"description,omitempty"`
	Parameters     map[string]any `json:"parameters,omitempty"`
	Contract       string         `json:"contract,omitempty"`
	OutputContract string         `json:"output_contract,omitempty"`
	ArgumentHint   string         `json:"argument_hint,omitempty"`
}

func (r *Runtime) toolSpecs(state runState) []model.ToolSpec {
	specs := []model.ToolSpec{toolSearchSpec()}
	if r.Config.Runtime.ForceToolSearchFirst && !state.SearchPerformed {
		return specs
	}

	for _, activated := range state.RetainedSkills {
		if cmd, ok := r.Registry.Get(activated.SkillID); ok {
			specs = append(specs, activatedSkillToolSpec(cmd, activated.ToolName))
		}
	}
	specs = append(specs, skillCallSpec(state.RetainedSkills))
	return specs
}

func (r *Runtime) activateSkills(state runState, queryInfo skill.QueryExplanation, hits []skill.SearchHit) activationPlan {
	plan := activationPlan{
		Policy:             "score-window",
		RetentionPolicy:    "retain-discovered-skills",
		SearchIndex:        state.SearchCount + 1,
		QueryFingerprint:   searchFingerprint(queryInfo),
		MaxActivatedSkills: r.Config.Runtime.MaxActivatedSkills,
		MaxRetainedSkills:  r.Config.Runtime.MaxRetainedSkills,
		ActivationMinScore: r.Config.Runtime.ActivationMinScore,
		ScoreRatio:         r.Config.Runtime.ActivationScoreRatio,
	}
	window := r.selectActivationWindow(hits, &plan)
	plan.WindowSkills = window
	plan.RetainedSkills, plan.NewlyActivated, plan.ReusedSkills, plan.EvictedSkills = mergeRetainedSkillPool(
		state.RetainedSkills,
		window,
		r.Config.Runtime.MaxRetainedSkills,
	)
	if plan.QueryFingerprint == state.LastSearchFingerprint && sameSkillRefs(plan.RetainedSkills, state.RetainedSkills) {
		plan.Unchanged = true
	}
	if len(plan.WindowSkills) == 0 {
		return plan
	}
	return plan
}

func (r *Runtime) selectActivationWindow(hits []skill.SearchHit, plan *activationPlan) []activatedSkillRef {
	if len(hits) == 0 {
		return nil
	}
	plan.FirstScore = hits[0].Score
	plan.Threshold = maxFloat(r.Config.Runtime.ActivationMinScore, hits[0].Score*r.Config.Runtime.ActivationScoreRatio)

	var window []activatedSkillRef
	for i, hit := range hits {
		if _, ok := r.Registry.Get(hit.ID); !ok {
			continue
		}
		if i > 0 {
			if len(window) >= r.Config.Runtime.MaxActivatedSkills {
				plan.SkippedSkills = append(plan.SkippedSkills, hit.ID)
				continue
			}
			if hit.Score < plan.Threshold {
				plan.SkippedSkills = append(plan.SkippedSkills, hit.ID)
				continue
			}
		}
		window = append(window, activatedSkillRef{
			SkillID:          hit.ID,
			ToolName:         activeSkillToolName(hit.ID),
			Name:             hit.Name,
			Score:            hit.Score,
			ActivationReason: hit.Reason,
		})
		if len(window) >= r.Config.Runtime.MaxActivatedSkills {
			for _, skipped := range hits[i+1:] {
				plan.SkippedSkills = append(plan.SkippedSkills, skipped.ID)
			}
			break
		}
	}

	if len(window) == 0 && len(hits) > 0 {
		hit := hits[0]
		window = append(window, activatedSkillRef{
			SkillID:          hit.ID,
			ToolName:         activeSkillToolName(hit.ID),
			Name:             hit.Name,
			Score:            hit.Score,
			ActivationReason: "forced-first-hit",
		})
	}
	return window
}

func mergeRetainedSkillPool(previous, window []activatedSkillRef, maxRetained int) ([]activatedSkillRef, []activatedSkillRef, []activatedSkillRef, []activatedSkillRef) {
	previous = uniqueSkillRefs(previous)
	window = uniqueSkillRefs(window)

	prevByID := map[string]activatedSkillRef{}
	for _, item := range previous {
		prevByID[item.SkillID] = item
	}

	var retained []activatedSkillRef
	var newlyActivated []activatedSkillRef
	var reused []activatedSkillRef
	inWindow := map[string]bool{}

	for _, item := range window {
		if _, ok := prevByID[item.SkillID]; ok {
			reused = append(reused, item)
		} else {
			newlyActivated = append(newlyActivated, item)
		}
		retained = append(retained, item)
		inWindow[item.SkillID] = true
	}
	for _, item := range previous {
		if inWindow[item.SkillID] {
			continue
		}
		retained = append(retained, item)
	}
	retained = uniqueSkillRefs(retained)

	var evicted []activatedSkillRef
	if maxRetained > 0 && len(retained) > maxRetained {
		evicted = append(evicted, retained[maxRetained:]...)
		retained = retained[:maxRetained]
	}

	return retained, uniqueSkillRefs(newlyActivated), uniqueSkillRefs(reused), uniqueSkillRefs(evicted)
}

func uniqueSkillRefs(items []activatedSkillRef) []activatedSkillRef {
	seen := map[string]bool{}
	var out []activatedSkillRef
	for _, item := range items {
		if strings.TrimSpace(item.SkillID) == "" || seen[item.SkillID] {
			continue
		}
		seen[item.SkillID] = true
		out = append(out, item)
	}
	return out
}

func sameSkillRefs(a, b []activatedSkillRef) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].SkillID != b[i].SkillID || a[i].ToolName != b[i].ToolName {
			return false
		}
	}
	return true
}

func searchFingerprint(queryInfo skill.QueryExplanation) string {
	parts := []string{
		strings.ToLower(strings.TrimSpace(queryInfo.Mode)),
		strings.ToLower(strings.TrimSpace(queryInfo.Raw)),
	}
	if len(queryInfo.RequiredTerms) > 0 {
		parts = append(parts, "required="+joinSortedTerms(queryInfo.RequiredTerms))
	}
	if len(queryInfo.OptionalTerms) > 0 {
		parts = append(parts, "optional="+joinSortedTerms(queryInfo.OptionalTerms))
	}
	if len(queryInfo.ScoringTerms) > 0 {
		parts = append(parts, "scoring="+joinSortedTerms(queryInfo.ScoringTerms))
	}
	return strings.Join(parts, "|")
}

func joinSortedTerms(terms []string) string {
	terms = append([]string{}, terms...)
	for i, term := range terms {
		terms[i] = strings.ToLower(strings.TrimSpace(term))
	}
	sort.Strings(terms)
	return strings.Join(terms, ",")
}

func toolSearchSpec() model.ToolSpec {
	return model.ToolSpec{
		Type: "function",
		Function: model.ToolFunction{
			Name:        "tool_search",
			Description: "Search registered skills by user intent. Supports exact bare query, direct select via select:skill-id, and required terms via +keyword. Returns query analysis, ranked hits, explicit tool-reference-like activation objects, a fresh activation window, and the retained skill tools that remain callable on later rounds.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{"type": "string", "description": "User intent, direct select syntax like select:webnovel-opening-sniper, or keywords with +required terms."},
					"limit": map[string]any{"type": "integer", "description": "Max number of ranked skills to return."},
				},
				"required": []string{"query"},
			},
		},
	}
}

func skillCallSpec(activated []activatedSkillRef) model.ToolSpec {
	properties := map[string]any{
		"task": map[string]any{"type": "string", "description": "Concrete task for the skill. Include user requirements."},
		"arguments": map[string]any{
			"type":        "object",
			"description": "Optional structured arguments for the target skill when you need a compatibility fallback for an activated skill tool with richer input fields.",
		},
	}
	if len(activated) == 0 {
		properties["skill_id"] = map[string]any{"type": "string", "description": "Registered skill id, e.g. webnovel-opening-sniper."}
	} else {
		var ids []string
		for _, item := range activated {
			ids = append(ids, item.SkillID)
		}
		properties["skill_id"] = map[string]any{
			"type":        "string",
			"description": "Skill id from the retained discovered-skill pool built by tool_search.",
			"enum":        ids,
		}
	}
	return model.ToolSpec{
		Type: "function",
		Function: model.ToolFunction{
			Name:        "skill_call",
			Description: "Compatibility shim for executing a retained skill from the discovered pool. Prefer the dynamically retained skill tools when available because they expose the real skill-specific input schema.",
			Parameters: map[string]any{
				"type":       "object",
				"properties": properties,
				"required":   []string{"skill_id", "task"},
			},
		},
	}
}

func activatedSkillToolSpec(cmd skill.Command, toolName string) model.ToolSpec {
	description := resolvedToolDescription(cmd)
	if strings.TrimSpace(description) == "" {
		description = fmt.Sprintf("Execute skill %s after it was activated by tool_search.", cmd.ID)
	}
	return model.ToolSpec{
		Type: "function",
		Function: model.ToolFunction{
			Name:        toolName,
			Description: fmt.Sprintf("Execute activated skill %s (%s). %s", cmd.ID, cmd.Name, description),
			Parameters:  resolvedToolParameters(cmd),
		},
	}
}

func (r *Runtime) buildToolReferences(items []activatedSkillRef) []callableToolReference {
	var refs []callableToolReference
	for _, item := range items {
		cmd, ok := r.Registry.Get(item.SkillID)
		if !ok {
			continue
		}
		refs = append(refs, buildToolReference(cmd, item.ToolName))
	}
	return refs
}

func buildToolReference(cmd skill.Command, toolName string) callableToolReference {
	return callableToolReference{
		Type:           "tool_reference_like",
		ToolName:       toolName,
		SkillID:        cmd.ID,
		SkillName:      cmd.Name,
		Description:    resolvedToolDescription(cmd),
		Parameters:     resolvedToolParameters(cmd),
		Contract:       cmd.ToolContract,
		OutputContract: cmd.ToolOutput,
		ArgumentHint:   cmd.ArgumentHint,
	}
}

func resolvedToolDescription(cmd skill.Command) string {
	return firstNonEmpty(cmd.ToolDescription, cmd.Description)
}

func resolvedToolParameters(cmd skill.Command) map[string]any {
	if len(cmd.ToolInputSchema) > 0 {
		return skill.CloneMap(cmd.ToolInputSchema)
	}
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"task": map[string]any{"type": "string", "description": "Concrete task for this activated skill."},
		},
		"required": []string{"task"},
	}
}

func activeSkillIDForToolName(toolName string, activated []activatedSkillRef) (string, bool) {
	for _, item := range activated {
		if item.ToolName == toolName {
			return item.SkillID, true
		}
	}
	return "", false
}

func activeSkillToolName(skillID string) string {
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(skillID))

	var b strings.Builder
	b.WriteString("skill_exec_")
	for _, r := range strings.ToLower(skillID) {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	name := strings.Trim(b.String(), "_")
	name = collapseUnderscores(name)
	if len(name) > 48 {
		name = name[:48]
		name = strings.TrimRight(name, "_")
	}
	return fmt.Sprintf("%s_%08x", name, hasher.Sum32())
}

func collapseUnderscores(s string) string {
	for strings.Contains(s, "__") {
		s = strings.ReplaceAll(s, "__", "_")
	}
	return s
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
