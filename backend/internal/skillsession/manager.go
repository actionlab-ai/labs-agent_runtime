package skillsession

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"novel-agent-runtime/internal/runtime"
)

const (
	StatusNeedsInput = runtime.SkillRunStatusNeedsInput
	StatusCompleted  = runtime.SkillRunStatusCompleted
)

type StartInput struct {
	ProjectID string         `json:"project_id,omitempty"`
	SkillID   string         `json:"skill_id"`
	Request   string         `json:"request"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

type ContinueInput struct {
	Input   string            `json:"input,omitempty"`
	Answers map[string]string `json:"answers,omitempty"`
	Notes   string            `json:"notes,omitempty"`
}

type Turn struct {
	Role      string            `json:"role"`
	Content   string            `json:"content,omitempty"`
	Answers   map[string]string `json:"answers,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
}

type Snapshot struct {
	ID        string                   `json:"id"`
	Status    string                   `json:"status"`
	ProjectID string                   `json:"project_id,omitempty"`
	SkillID   string                   `json:"skill_id"`
	Request   string                   `json:"request"`
	Arguments map[string]any           `json:"arguments,omitempty"`
	AskHuman  *runtime.AskHumanRequest `json:"ask_human,omitempty"`
	FinalText string                   `json:"final_text,omitempty"`
	RunID     string                   `json:"run_id,omitempty"`
	RunDir    string                   `json:"run_dir,omitempty"`
	Turns     []Turn                   `json:"turns,omitempty"`
	CreatedAt time.Time                `json:"created_at"`
	UpdatedAt time.Time                `json:"updated_at"`
}

type sessionState struct {
	snapshot Snapshot
	rt       *runtime.Runtime
	state    *runtime.SkillExecutionState
}

type Manager struct {
	mu       sync.RWMutex
	nextID   func() string
	now      func() time.Time
	sessions map[string]*sessionState
}

func NewManager() *Manager {
	return &Manager{
		nextID:   func() string { return "ss_" + time.Now().UTC().Format("20060102T150405.000000000") },
		now:      func() time.Time { return time.Now().UTC() },
		sessions: map[string]*sessionState{},
	}
}

func (m *Manager) Start(ctx context.Context, rt *runtime.Runtime, input StartInput) (Snapshot, error) {
	if rt == nil {
		return Snapshot{}, fmt.Errorf("runtime is required")
	}
	input.SkillID = strings.TrimSpace(input.SkillID)
	input.Request = strings.TrimSpace(input.Request)
	if input.SkillID == "" {
		return Snapshot{}, fmt.Errorf("skill_id is required")
	}
	if input.Request == "" {
		return Snapshot{}, fmt.Errorf("request is required")
	}
	result, err := rt.ExecuteSkillSession(ctx, input.SkillID, input.Request, cloneMap(input.Arguments))
	if err != nil {
		return Snapshot{}, err
	}
	now := m.now()
	s := &sessionState{
		rt: rt,
		snapshot: Snapshot{
			ID:        m.nextID(),
			Status:    result.Status,
			ProjectID: strings.TrimSpace(input.ProjectID),
			SkillID:   input.SkillID,
			Request:   input.Request,
			Arguments: cloneMap(input.Arguments),
			AskHuman:  result.AskHuman,
			FinalText: result.Text,
			RunID:     result.RunID,
			RunDir:    result.RunDir,
			Turns: []Turn{{
				Role:      "user",
				Content:   input.Request,
				CreatedAt: now,
			}},
			CreatedAt: now,
			UpdatedAt: now,
		},
		state: result.State,
	}
	if result.Status == StatusNeedsInput && result.AskHuman != nil {
		s.snapshot.Turns = append(s.snapshot.Turns, Turn{
			Role:      "assistant",
			Content:   askHumanSummary(*result.AskHuman),
			CreatedAt: now,
		})
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[s.snapshot.ID] = s
	return cloneSnapshot(s.snapshot), nil
}

func (m *Manager) Continue(ctx context.Context, id string, input ContinueInput) (Snapshot, error) {
	id = strings.TrimSpace(id)
	m.mu.Lock()
	s, ok := m.sessions[id]
	if !ok {
		m.mu.Unlock()
		return Snapshot{}, fmt.Errorf("skill session %q not found", id)
	}
	if s.snapshot.Status != StatusNeedsInput {
		out := cloneSnapshot(s.snapshot)
		m.mu.Unlock()
		return Snapshot{}, fmt.Errorf("skill session %q is not waiting for input; status=%s", id, out.Status)
	}
	if s.rt == nil || s.state == nil {
		m.mu.Unlock()
		return Snapshot{}, fmt.Errorf("skill session %q cannot resume because runtime state is unavailable", id)
	}
	rt := s.rt
	state := *s.state
	m.mu.Unlock()

	result, err := rt.ContinueSkillInteractive(ctx, state, runtime.AskHumanAnswer{
		Answers: cloneStringMap(input.Answers),
		Notes:   strings.TrimSpace(input.Notes),
	})
	if err != nil {
		return Snapshot{}, err
	}

	now := m.now()
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok = m.sessions[id]
	if !ok {
		return Snapshot{}, fmt.Errorf("skill session %q not found", id)
	}
	s.snapshot.Status = result.Status
	s.snapshot.AskHuman = result.AskHuman
	s.snapshot.FinalText = result.Text
	s.snapshot.RunID = result.RunID
	s.snapshot.RunDir = result.RunDir
	s.snapshot.UpdatedAt = now
	s.state = result.State
	s.snapshot.Turns = append(s.snapshot.Turns, Turn{
		Role:      "user",
		Content:   strings.TrimSpace(input.Input),
		Answers:   cloneStringMap(input.Answers),
		CreatedAt: now,
	})
	if result.Status == StatusNeedsInput && result.AskHuman != nil {
		s.snapshot.Turns = append(s.snapshot.Turns, Turn{
			Role:      "assistant",
			Content:   askHumanSummary(*result.AskHuman),
			CreatedAt: now,
		})
	}
	if result.Status == StatusCompleted && strings.TrimSpace(result.Text) != "" {
		s.snapshot.Turns = append(s.snapshot.Turns, Turn{
			Role:      "assistant",
			Content:   result.Text,
			CreatedAt: now,
		})
	}
	return cloneSnapshot(s.snapshot), nil
}

func (m *Manager) Get(id string) (Snapshot, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[strings.TrimSpace(id)]
	if !ok {
		return Snapshot{}, false
	}
	return cloneSnapshot(s.snapshot), true
}

func askHumanSummary(request runtime.AskHumanRequest) string {
	questions := make([]string, 0, len(request.Questions))
	for _, q := range request.Questions {
		questions = append(questions, q.Question)
	}
	return strings.Join(questions, "\n")
}

func cloneSnapshot(in Snapshot) Snapshot {
	out := in
	out.Arguments = cloneMap(in.Arguments)
	if in.Turns != nil {
		out.Turns = append([]Turn{}, in.Turns...)
	}
	return out
}

func cloneMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
