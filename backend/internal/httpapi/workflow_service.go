package httpapi

import (
	"context"
	"encoding/json"
	"time"

	"go.uber.org/zap"

	"novel-agent-runtime/internal/config"
	"novel-agent-runtime/internal/logging"
	"novel-agent-runtime/internal/store"
	"novel-agent-runtime/internal/workflow"
)

type workflowResponse struct {
	DryRun           bool                     `json:"dry_run,omitempty"`
	WorkflowID       string                   `json:"workflow_id"`
	Stage            string                   `json:"stage"`
	Steps            []workflow.WorkflowStep  `json:"steps"`
	Arguments        map[string]any           `json:"arguments,omitempty"`
	WorkflowOutput   *workflow.WorkflowOutput `json:"workflow_output,omitempty"`
	UpdatedDocuments []workflowDocumentUpdate `json:"updated_documents,omitempty"`
	ResponseMode     string                   `json:"response_mode,omitempty"`
	NeedsInput       bool                     `json:"needs_input,omitempty"`
	Clarification    *workflowClarification   `json:"clarification,omitempty"`
	FinalText        string                   `json:"final_text,omitempty"`
	RunID            string                   `json:"run_id"`
	RunDir           string                   `json:"run_dir"`
	DBRun            *store.Run               `json:"db_run,omitempty"`
	Project          *store.Project           `json:"project,omitempty"`
	Model            *store.ModelProfile      `json:"model,omitempty"`
}

type WorkflowService struct {
	db      appStore
	factory runtimeFactory
}

func NewWorkflowService(cfg config.Config, debug bool, db appStore, models modelConfigStore, projects projectConfigStore) WorkflowService {
	return WorkflowService{
		db: db,
		factory: runtimeFactory{
			cfg:      cfg,
			debug:    debug,
			models:   models,
			projects: projects,
		},
	}
}

func (s WorkflowService) Execute(ctx context.Context, req workflowRunRequest, planBuilder workflowPlanBuilder) (workflowResponse, error) {
	plan, err := planBuilder(req)
	if err != nil {
		return workflowResponse{}, withStatus(400, err)
	}

	session, err := s.factory.resolveWorkflowSession(ctx, req)
	if err != nil {
		return workflowResponse{}, err
	}
	if err := ensureWorkflowSkillsExist(ctx, session.runtime, plan.Steps); err != nil {
		return workflowResponse{}, withStatus(400, err)
	}

	_ = session.runtime.Store.WriteJSON("workflow/plan.json", plan)
	_ = session.runtime.Store.WriteJSON("workflow/context-pack.json", session.contextPack)

	logger := logging.FromContext(ctx)
	if req.DryRun {
		logger.Info("workflow.dry_run.completed",
			zap.String("workflow_id", plan.WorkflowID),
			zap.String("stage", plan.Stage),
			zap.String("project_id", session.activeProject.ID),
			zap.String("profile_id", session.activeModel.ID),
			zap.String("run_id", session.runtime.Store.RunID),
		)
		return workflowResponse{
			DryRun:     true,
			WorkflowID: plan.WorkflowID,
			Stage:      plan.Stage,
			Steps:      plan.Steps,
			Arguments:  plan.Arguments,
			RunID:      session.runtime.Store.RunID,
			RunDir:     session.runtime.Store.Root,
			Project:    session.activeProject,
			Model:      session.activeModel,
		}, nil
	}

	runMeta, _ := json.Marshal(map[string]any{
		"type":         "workflow",
		"workflow_id":  plan.WorkflowID,
		"stage":        plan.Stage,
		"step_ids":     workflowStepIDs(plan.Steps),
		"skill_ids":    workflowSkillIDs(plan.Steps),
		"persist_mode": plan.PersistMode,
		"persist_kind": plan.PersistKind,
	})
	dbRun, err := s.db.CreateRun(ctx, session.activeProject.ID, req.Input, runMeta)
	if err != nil {
		return workflowResponse{}, withStatus(500, err)
	}

	timeout := time.Duration(session.runtimeModel.TimeoutSeconds+30) * time.Second
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	runner := workflow.SequentialWorkflowRunner{
		SkillRunner: workflow.RuntimeSkillRunner{Runtime: session.runtime},
	}
	workflowOut, err := runner.RunWorkflow(runCtx, workflow.WorkflowInput{
		WorkflowID: plan.WorkflowID,
		Request:    req.Input,
		ProjectID:  session.activeProject.ID,
		Context:    session.contextPack,
		Arguments:  cloneMap(plan.Arguments),
		Steps:      append([]workflow.WorkflowStep{}, plan.Steps...),
	})
	if err != nil {
		_, _ = s.db.FailRun(ctx, dbRun.ID, err.Error())
		return workflowResponse{}, withStatus(500, err)
	}
	_ = session.runtime.Store.WriteJSON("workflow/output.json", workflowOut)

	persistenceResult, err := persistWorkflowDocuments(runCtx, s.factory.projects, session.activeProject.ID, plan, workflowOut)
	if err != nil {
		_, _ = s.db.FailRun(ctx, dbRun.ID, err.Error())
		return workflowResponse{}, withStatus(500, err)
	}
	_ = session.runtime.Store.WriteJSON("workflow/persistence-result.json", persistenceResult)
	_ = session.runtime.Store.WriteJSON("workflow/updated-documents.json", persistenceResult.UpdatedDocuments)

	rawFinalText := workflowFinalText(workflowOut)
	finalText, clarification := buildWorkflowResponseDetails(plan, persistenceResult, rawFinalText)
	if clarification != nil {
		_ = session.runtime.Store.WriteJSON("workflow/clarification.json", clarification)
	}
	finished, err := s.db.FinishRun(ctx, dbRun.ID, finalText, session.runtime.Store.Root)
	if err != nil {
		return workflowResponse{}, withStatus(500, err)
	}

	logger.Info("workflow.completed",
		zap.String("workflow_id", plan.WorkflowID),
		zap.String("stage", plan.Stage),
		zap.String("project_id", session.activeProject.ID),
		zap.String("profile_id", session.activeModel.ID),
		zap.String("run_id", session.runtime.Store.RunID),
		zap.Int("updated_document_count", len(persistenceResult.UpdatedDocuments)),
		zap.String("response_mode", persistenceResult.ResponseMode),
		zap.Bool("needs_input", persistenceResult.NeedsInput),
	)

	return workflowResponse{
		WorkflowID:       plan.WorkflowID,
		Stage:            plan.Stage,
		Steps:            plan.Steps,
		WorkflowOutput:   &workflowOut,
		UpdatedDocuments: persistenceResult.UpdatedDocuments,
		ResponseMode:     persistenceResult.ResponseMode,
		NeedsInput:       persistenceResult.NeedsInput,
		Clarification:    clarification,
		FinalText:        finalText,
		RunID:            session.runtime.Store.RunID,
		RunDir:           session.runtime.Store.Root,
		DBRun:            &finished,
		Project:          session.activeProject,
		Model:            session.activeModel,
	}, nil
}
