package runtime

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"novel-agent-runtime/internal/model"
	"novel-agent-runtime/internal/skill"
)

type promptBlock struct {
	Label   string `json:"label"`
	Content string `json:"content"`
}

type preparationStep struct {
	Name    string `json:"name"`
	Applied bool   `json:"applied"`
	Note    string `json:"note,omitempty"`
}

type deferredSkillDescriptor struct {
	SkillID      string   `json:"skill_id"`
	Name         string   `json:"name"`
	Description  string   `json:"description,omitempty"`
	WhenToUse    string   `json:"when_to_use,omitempty"`
	SearchHint   string   `json:"search_hint,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	ContentBytes int      `json:"content_bytes,omitempty"`
}

type localToolDescriptor struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Category    string   `json:"category,omitempty"`
	SchemaKeys  []string `json:"schema_keys,omitempty"`
}

type responseAnalysis struct {
	FinishReason  string   `json:"finish_reason,omitempty"`
	HasToolCalls  bool     `json:"has_tool_calls"`
	ToolCallNames []string `json:"tool_call_names,omitempty"`
	Content       string   `json:"content,omitempty"`
}

type routerAssembly struct {
	Round              int                       `json:"round"`
	SearchPerformed    bool                      `json:"search_performed"`
	SearchCount        int                       `json:"search_count"`
	SystemPromptBlocks []promptBlock             `json:"system_prompt_blocks"`
	SystemPrompt       string                    `json:"system_prompt"`
	DeferredSkills     []deferredSkillDescriptor `json:"deferred_skills,omitempty"`
	RetainedSkillTools []callableToolReference   `json:"retained_skill_tools,omitempty"`
	Preparation        []preparationStep         `json:"preparation"`
	Messages           []model.Message           `json:"messages"`
	Tools              []model.ToolSpec          `json:"tools"`
	ChatRequest        model.ChatRequest         `json:"chat_request"`
}

type skillAssembly struct {
	SkillID            string                `json:"skill_id"`
	Round              int                   `json:"round"`
	SystemPromptBlocks []promptBlock         `json:"system_prompt_blocks"`
	SystemPrompt       string                `json:"system_prompt"`
	CompiledPrompt     string                `json:"compiled_prompt"`
	LocalTools         []localToolDescriptor `json:"local_tools,omitempty"`
	Preparation        []preparationStep     `json:"preparation"`
	Messages           []model.Message       `json:"messages"`
	Tools              []model.ToolSpec      `json:"tools"`
	ChatRequest        model.ChatRequest     `json:"chat_request"`
}

func (r *Runtime) buildRouterAssembly(userInput string, state runState, conversation []model.Message, round int) routerAssembly {
	blocks := r.routerSystemPromptBlocks(state)
	systemPrompt := joinPromptBlocks(blocks)
	toolSpecs := r.toolSpecs(state)
	deferredSkills := r.availableDeferredSkills(state)
	retainedRefs := r.buildToolReferences(state.RetainedSkills)
	messages, steps := buildRouterMessages(systemPrompt, conversation, deferredSkills, retainedRefs)

	chatReq := model.ChatRequest{
		Model:       r.Config.Model.ID,
		Messages:    messages,
		Tools:       toolSpecs,
		ToolChoice:  "auto",
		Temperature: 0.1,
		MaxTokens:   min(2048, r.Config.Model.MaxOutput),
	}
	return routerAssembly{
		Round:              round,
		SearchPerformed:    state.SearchPerformed,
		SearchCount:        state.SearchCount,
		SystemPromptBlocks: blocks,
		SystemPrompt:       systemPrompt,
		DeferredSkills:     deferredSkills,
		RetainedSkillTools: retainedRefs,
		Preparation:        steps,
		Messages:           messages,
		Tools:              toolSpecs,
		ChatRequest:        chatReq,
	}
}

func (r *Runtime) buildSkillAssembly(cmd skill.Command, compiledPrompt string, round int, conversation []model.Message, toolSpecs []model.ToolSpec) skillAssembly {
	systemText := "You are the controlled Skill Executor for Novel Runtime. Execute only the injected skill, do not invent missing user facts, keep durable novel outputs in workspace markdown files when local file tools are available, and use shell tools only for terminal operations. If Read, Write, Edit, or Glob are available, prefer them over shell commands for file manipulation."
	blocks := []promptBlock{
		{Label: "executor_role", Content: systemText},
	}
	steps := []preparationStep{
		{Name: "inject_system_prompt", Applied: true, Note: "Skill executor system prompt rebuilt for this round."},
		{Name: "inject_compiled_skill_prompt", Applied: round == 1, Note: "Compiled skill prompt stays in the user conversation and is sent every round."},
	}
	localTools := describeLocalTools(toolSpecs)
	chatReq := model.ChatRequest{
		Model:       firstNonEmpty(cmd.Model, r.Config.Model.ID),
		Messages:    append([]model.Message{{Role: "system", Content: systemText}}, conversation...),
		Tools:       toolSpecs,
		ToolChoice:  "auto",
		Temperature: r.Config.Model.Temperature,
		MaxTokens:   r.effectiveSkillMaxTokens(),
	}
	return skillAssembly{
		SkillID:            cmd.ID,
		Round:              round,
		SystemPromptBlocks: blocks,
		SystemPrompt:       systemText,
		CompiledPrompt:     compiledPrompt,
		LocalTools:         localTools,
		Preparation:        steps,
		Messages:           chatReq.Messages,
		Tools:              toolSpecs,
		ChatRequest:        chatReq,
	}
}

func (r *Runtime) routerSystemPromptBlocks(state runState) []promptBlock {
	blocks := []promptBlock{
		{
			Label: "router_role",
			Content: `You are the Router for Novel Agent Runtime, not the final writer.

Responsibilities:
1. Classify the user's request.
2. If only tool_search is currently exposed, call tool_search first.
3. tool_search returns candidate skills, explicit tool-reference-like activation objects, a fresh activation window, and an updated retained skill pool for later rounds.
4. Prefer calling those retained skill tools directly because they expose the real skill-specific input schema. skill_call is only a compatibility shim.
5. Do not write the final skill output yourself unless tool_search fails to find a suitable skill.

Constraints:
- tool_search returns activation references and retained callable tool names, not the full SKILL.md.
- Once a skill tool is retained, it stays callable on later rounds until Runtime evicts it from the retained pool.`,
		},
	}
	if r.Config.Runtime.ForceToolSearchFirst && !state.SearchPerformed {
		blocks = append(blocks, promptBlock{
			Label:   "search_first_gate",
			Content: "Current round rule: only tool_search should be used to discover a suitable local skill before any direct skill execution.",
		})
	}
	if len(state.RetainedSkills) > 0 {
		blocks = append(blocks, promptBlock{
			Label:   "retained_tool_bias",
			Content: fmt.Sprintf("Current retained skill tool count: %d. Prefer those activated skill tools over skill_call because they already expose the best schema for the next step.", len(state.RetainedSkills)),
		})
	}
	return blocks
}

func joinPromptBlocks(blocks []promptBlock) string {
	var parts []string
	for _, block := range blocks {
		if strings.TrimSpace(block.Content) == "" {
			continue
		}
		parts = append(parts, block.Content)
	}
	return strings.Join(parts, "\n\n")
}

func buildRouterMessages(systemPrompt string, conversation []model.Message, deferredSkills []deferredSkillDescriptor, retainedRefs []callableToolReference) ([]model.Message, []preparationStep) {
	messages := []model.Message{{Role: "system", Content: systemPrompt}}
	steps := []preparationStep{
		{Name: "inject_system_prompt", Applied: true, Note: "Router system prompt blocks were joined into the first system message."},
	}
	if reminder := formatDeferredSkillsReminder(deferredSkills); reminder != "" {
		messages = append(messages, model.Message{Role: "user", Content: reminder})
		steps = append(steps, preparationStep{
			Name:    "prepend_available_deferred_skills",
			Applied: true,
			Note:    "Novelcode-style deferred skill list was injected as a meta reminder before the real conversation.",
		})
	} else {
		steps = append(steps, preparationStep{
			Name:    "prepend_available_deferred_skills",
			Applied: false,
			Note:    "No undiscovered deferred skills remained outside the retained pool.",
		})
	}
	if reminder := formatRetainedSkillToolsReminder(retainedRefs); reminder != "" {
		messages = append(messages, model.Message{Role: "user", Content: reminder})
		steps = append(steps, preparationStep{
			Name:    "prepend_retained_skill_tools",
			Applied: true,
			Note:    "Activated callable skill tools were summarized as a retained pool reminder.",
		})
	} else {
		steps = append(steps, preparationStep{
			Name:    "prepend_retained_skill_tools",
			Applied: false,
			Note:    "No retained skill tools existed for this round yet.",
		})
	}
	messages = append(messages, cloneMessages(conversation)...)
	steps = append(steps, preparationStep{
		Name:    "append_live_conversation",
		Applied: true,
		Note:    "The actual user/assistant/tool transcript for this run was appended after preprocessing reminders.",
	})
	return messages, steps
}

func (r *Runtime) availableDeferredSkills(state runState) []deferredSkillDescriptor {
	retained := map[string]bool{}
	for _, item := range state.RetainedSkills {
		retained[item.SkillID] = true
	}
	var skillsOut []deferredSkillDescriptor
	for _, cmd := range r.Registry.List() {
		if !cmd.UserInvocable || retained[cmd.ID] {
			continue
		}
		skillsOut = append(skillsOut, deferredSkillDescriptor{
			SkillID:      cmd.ID,
			Name:         cmd.Name,
			Description:  cmd.Description,
			WhenToUse:    cmd.WhenToUse,
			SearchHint:   cmd.SearchHint,
			Tags:         cmd.Tags,
			ContentBytes: cmd.ContentLength,
		})
	}
	return skillsOut
}

func formatDeferredSkillsReminder(items []deferredSkillDescriptor) string {
	if len(items) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("<available-deferred-skills>\n")
	b.WriteString("The following local skills are available through tool_search. Use keywords, +required terms, or select:<skill_id> to activate one.\n")
	b.WriteString("Examples: ")
	maxExamples := min(3, len(items))
	for i := 0; i < maxExamples; i++ {
		if i > 0 {
			b.WriteString(" | ")
		}
		b.WriteString("select:")
		b.WriteString(items[i].SkillID)
	}
	b.WriteString("\n")
	for _, item := range items {
		b.WriteString("- ")
		b.WriteString(item.SkillID)
		b.WriteString(" | name=")
		b.WriteString(item.Name)
		if item.Description != "" {
			b.WriteString(" | description=")
			b.WriteString(item.Description)
		}
		if item.WhenToUse != "" {
			b.WriteString(" | when_to_use=")
			b.WriteString(item.WhenToUse)
		}
		if item.SearchHint != "" {
			b.WriteString(" | search_hint=")
			b.WriteString(item.SearchHint)
		}
		b.WriteString("\n")
	}
	b.WriteString("</available-deferred-skills>")
	return b.String()
}

func formatRetainedSkillToolsReminder(items []callableToolReference) string {
	if len(items) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("<retained-skill-tools>\n")
	b.WriteString("The following activated skill tools are already callable in this round. Prefer them over skill_call.\n")
	for _, item := range items {
		b.WriteString("- ")
		b.WriteString(item.ToolName)
		b.WriteString(" | skill_id=")
		b.WriteString(item.SkillID)
		if item.Contract != "" {
			b.WriteString(" | contract=")
			b.WriteString(item.Contract)
		}
		if item.OutputContract != "" {
			b.WriteString(" | output_contract=")
			b.WriteString(item.OutputContract)
		}
		fields := schemaFieldNames(item.Parameters)
		if len(fields) > 0 {
			b.WriteString(" | fields=")
			b.WriteString(strings.Join(fields, ","))
		}
		b.WriteString("\n")
	}
	b.WriteString("</retained-skill-tools>")
	return b.String()
}

func describeLocalTools(specs []model.ToolSpec) []localToolDescriptor {
	var out []localToolDescriptor
	for _, spec := range specs {
		out = append(out, localToolDescriptor{
			Name:        spec.Function.Name,
			Description: spec.Function.Description,
			Category:    localToolCategory(spec.Function.Name),
			SchemaKeys:  schemaFieldNames(spec.Function.Parameters),
		})
	}
	return out
}

func localToolCategory(name string) string {
	switch name {
	case skillReadToolName, skillWriteToolName, skillEditToolName, skillGlobToolName:
		return "file"
	case skillBashToolName, skillPowerShellToolName:
		return "shell"
	default:
		return "custom"
	}
}

func schemaFieldNames(schema map[string]any) []string {
	properties, ok := schema["properties"].(map[string]any)
	if !ok || len(properties) == 0 {
		return nil
	}
	keys := make([]string, 0, len(properties))
	for k := range properties {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func cloneMessages(in []model.Message) []model.Message {
	out := make([]model.Message, 0, len(in))
	for _, msg := range in {
		copyMsg := msg
		if len(msg.ToolCalls) > 0 {
			copyMsg.ToolCalls = append([]model.ToolCall{}, msg.ToolCalls...)
		}
		out = append(out, copyMsg)
	}
	return out
}

func analyzeChatResponse(resp model.ChatResponse) responseAnalysis {
	if len(resp.Choices) == 0 {
		return responseAnalysis{}
	}
	choice := resp.Choices[0]
	msg := choice.Message
	analysis := responseAnalysis{
		FinishReason: choice.FinishReason,
		HasToolCalls: len(msg.ToolCalls) > 0,
		Content:      msg.Content,
	}
	if len(msg.ToolCalls) > 0 {
		for _, tc := range msg.ToolCalls {
			analysis.ToolCallNames = append(analysis.ToolCallNames, tc.Function.Name)
		}
	}
	return analysis
}

func renderRouterAssemblyMarkdown(assembly routerAssembly) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("# Router Round %02d Assembly\n\n", assembly.Round))
	b.WriteString("## System Prompt Blocks\n\n")
	for _, block := range assembly.SystemPromptBlocks {
		b.WriteString("### " + block.Label + "\n\n")
		b.WriteString(block.Content + "\n\n")
	}
	if len(assembly.DeferredSkills) > 0 {
		b.WriteString("## Deferred Skills\n\n")
		for _, item := range assembly.DeferredSkills {
			b.WriteString("- `" + item.SkillID + "` | " + item.Name + "\n")
		}
		b.WriteString("\n")
	}
	if len(assembly.RetainedSkillTools) > 0 {
		b.WriteString("## Retained Skill Tools\n\n")
		for _, item := range assembly.RetainedSkillTools {
			b.WriteString("- `" + item.ToolName + "` <- `" + item.SkillID + "`\n")
		}
		b.WriteString("\n")
	}
	b.WriteString("## Preparation\n\n")
	for _, step := range assembly.Preparation {
		state := "skipped"
		if step.Applied {
			state = "applied"
		}
		b.WriteString("- `" + step.Name + "`: " + state)
		if step.Note != "" {
			b.WriteString(" | " + step.Note)
		}
		b.WriteString("\n")
	}
	b.WriteString("\n## Final Tool Names\n\n")
	for _, tool := range assembly.Tools {
		b.WriteString("- `" + tool.Function.Name + "`\n")
	}
	return b.String()
}

func renderSkillAssemblyMarkdown(assembly skillAssembly) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("# Skill Assembly: %s Round %02d\n\n", assembly.SkillID, assembly.Round))
	b.WriteString("## Local Tools\n\n")
	for _, tool := range assembly.LocalTools {
		b.WriteString("- `" + tool.Name + "`")
		if tool.Category != "" {
			b.WriteString(" | " + tool.Category)
		}
		if len(tool.SchemaKeys) > 0 {
			b.WriteString(" | fields=" + strings.Join(tool.SchemaKeys, ","))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n## Preparation\n\n")
	for _, step := range assembly.Preparation {
		state := "skipped"
		if step.Applied {
			state = "applied"
		}
		b.WriteString("- `" + step.Name + "`: " + state)
		if step.Note != "" {
			b.WriteString(" | " + step.Note)
		}
		b.WriteString("\n")
	}
	return b.String()
}

func mustPrettyJSON(v any) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}
