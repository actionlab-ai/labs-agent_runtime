package httpapi

import (
	"fmt"
	"strings"

	"novel-agent-runtime/internal/workflow"
)

func buildProjectBootstrapPlan(stage, input string, overrides map[string]any) (fixedWorkflowPlan, error) {
	normalized := strings.ToLower(strings.TrimSpace(stage))
	if normalized == "" {
		normalized = projectBootstrapStageKick
	}

	switch normalized {
	case projectBootstrapStageKick:
		return buildProjectKickoffPlan(input, overrides), nil
	case projectBootstrapStageCore:
		return buildProjectKernelPlan(input, overrides), nil
	case projectBootstrapStageWorld:
		return fixedWorkflowPlan{
			WorkflowID: projectBootstrapWorkflowID,
			Stage:      projectBootstrapStageWorld,
			Steps: []workflow.WorkflowStep{{
				ID:      "world-power-bootstrap",
				SkillID: "novel-idea-bootstrap",
			}},
			Arguments: mergeMaps(
				map[string]any{
					"task":          strings.TrimSpace(input),
					"question_mode": "clarify_first",
					"focus_scope":   "world_and_power",
				},
				overrides,
			),
		}, nil
	case projectBootstrapStagePower:
		return fixedWorkflowPlan{
			WorkflowID: projectBootstrapWorkflowID,
			Stage:      projectBootstrapStagePower,
			Steps: []workflow.WorkflowStep{{
				ID:      "power-bootstrap",
				SkillID: "novel-idea-bootstrap",
			}},
			Arguments: mergeMaps(
				map[string]any{
					"task":          strings.TrimSpace(input),
					"question_mode": "clarify_first",
					"focus_scope":   "power_only",
				},
				overrides,
			),
		}, nil
	default:
		return fixedWorkflowPlan{}, fmt.Errorf("unsupported workflow stage %q; use kickoff, kernel, world_power, or power_only", stage)
	}
}

func buildProjectKickoffPlan(input string, overrides map[string]any) fixedWorkflowPlan {
	return fixedWorkflowPlan{
		WorkflowID:     projectKickoffWorkflowID,
		Stage:          projectBootstrapStageKick,
		PersistMode:    persistModeExtractSections,
		ClarifyHeading: "需要补充的信息",
		Steps: []workflow.WorkflowStep{{
			ID:      "kickoff",
			SkillID: "novel-project-kickoff",
		}},
		Arguments: mergeMaps(
			map[string]any{
				"task":          strings.TrimSpace(input),
				"question_mode": "clarify_first",
			},
			overrides,
		),
	}
}

func buildProjectKernelPlan(input string, overrides map[string]any) fixedWorkflowPlan {
	return fixedWorkflowPlan{
		WorkflowID:     projectKernelWorkflowID,
		Stage:          projectBootstrapStageCore,
		PersistMode:    persistModeSingleDocument,
		PersistKind:    "novel_core",
		PersistTitle:   "小说情感内核",
		PersistHeading: "小说情感内核",
		ClarifyHeading: "需要补充的信息",
		Steps: []workflow.WorkflowStep{{
			ID:      "kernel",
			SkillID: "novel-emotional-core",
		}},
		Arguments: mergeMaps(
			map[string]any{
				"task":          strings.TrimSpace(input),
				"question_mode": "clarify_first",
				"document_kind": "novel_core",
			},
			overrides,
		),
	}
}

func workflowStepIDs(steps []workflow.WorkflowStep) []string {
	out := make([]string, 0, len(steps))
	for _, step := range steps {
		out = append(out, strings.TrimSpace(step.ID))
	}
	return out
}

func workflowSkillIDs(steps []workflow.WorkflowStep) []string {
	out := make([]string, 0, len(steps))
	for _, step := range steps {
		out = append(out, strings.TrimSpace(step.SkillID))
	}
	return out
}

func mergeMaps(base, override map[string]any) map[string]any {
	out := cloneMap(base)
	for k, v := range override {
		out[k] = v
	}
	return out
}

func cloneMap(src map[string]any) map[string]any {
	if src == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}
