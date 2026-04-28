package httpapi

import "novel-agent-runtime/internal/workflow"

const (
	projectBootstrapWorkflowID = "project-bootstrap"
	projectKickoffWorkflowID   = "project-kickoff"
	projectKernelWorkflowID    = "project-kernel"

	projectBootstrapStageKick  = "kickoff"
	projectBootstrapStageCore  = "kernel"
	projectBootstrapStageWorld = "world_power"
	projectBootstrapStagePower = "power_only"

	persistModeExtractSections = "extract_sections"
	persistModeSingleDocument  = "single_document"

	workflowResponseModeDocument      = "document"
	workflowResponseModeClarification = "clarification"
)

type workflowRunRequest struct {
	Input     string         `json:"input" binding:"required"`
	Project   string         `json:"project" binding:"required"`
	Model     string         `json:"model"`
	Stage     string         `json:"stage"`
	DryRun    bool           `json:"dry_run"`
	Debug     *bool          `json:"debug"`
	Arguments map[string]any `json:"arguments"`
}

type fixedWorkflowPlan struct {
	WorkflowID     string                  `json:"workflow_id"`
	Stage          string                  `json:"stage"`
	Arguments      map[string]any          `json:"arguments,omitempty"`
	Steps          []workflow.WorkflowStep `json:"steps"`
	PersistMode    string                  `json:"persist_mode,omitempty"`
	PersistKind    string                  `json:"persist_kind,omitempty"`
	PersistTitle   string                  `json:"persist_title,omitempty"`
	PersistHeading string                  `json:"persist_heading,omitempty"`
	ClarifyHeading string                  `json:"clarify_heading,omitempty"`
}

type workflowDocumentUpdate struct {
	Kind      string `json:"kind"`
	Title     string `json:"title"`
	BodyBytes int    `json:"body_bytes"`
}

type workflowClarificationQuestion struct {
	Field   string   `json:"field,omitempty"`
	Prompt  string   `json:"prompt"`
	Options []string `json:"options,omitempty"`
}

type workflowClarification struct {
	MissingFields   []string                        `json:"missing_fields,omitempty"`
	ConfirmedFacts  []string                        `json:"confirmed_facts,omitempty"`
	BlockingReasons []string                        `json:"blocking_reasons,omitempty"`
	Questions       []workflowClarificationQuestion `json:"questions,omitempty"`
}

type workflowPersistenceResult struct {
	UpdatedDocuments []workflowDocumentUpdate `json:"updated_documents,omitempty"`
	ResponseMode     string                   `json:"response_mode,omitempty"`
	NeedsInput       bool                     `json:"needs_input,omitempty"`
}

type workflowPlanBuilder func(req workflowRunRequest) (fixedWorkflowPlan, error)
