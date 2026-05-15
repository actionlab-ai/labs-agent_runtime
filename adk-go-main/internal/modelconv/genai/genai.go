// Package genai converts between ADK-owned model types and Google GenAI SDK types.
package genai

import (
	adkmodel "google.golang.org/adk/model"
	googlegenai "google.golang.org/genai"
)

func ToGenaiContents(in []*adkmodel.Content) []*googlegenai.Content {
	out := make([]*googlegenai.Content, 0, len(in))
	for _, c := range in {
		out = append(out, ToGenaiContent(c))
	}
	return out
}

func ToGenaiContent(c *adkmodel.Content) *googlegenai.Content {
	if c == nil {
		return nil
	}
	parts := make([]*googlegenai.Part, 0, len(c.Parts))
	for _, p := range c.Parts {
		parts = append(parts, ToGenaiPart(p))
	}
	return &googlegenai.Content{Role: string(c.Role), Parts: parts}
}

func FromGenaiContent(c *googlegenai.Content) *adkmodel.Content {
	if c == nil {
		return nil
	}
	parts := make([]*adkmodel.Part, 0, len(c.Parts))
	for _, p := range c.Parts {
		parts = append(parts, FromGenaiPart(p))
	}
	return &adkmodel.Content{Role: adkmodel.Role(c.Role), Parts: parts}
}

func ToGenaiPart(p *adkmodel.Part) *googlegenai.Part {
	if p == nil {
		return nil
	}
	out := &googlegenai.Part{Text: p.Text, Thought: p.Thought, ThoughtSignature: p.ThoughtSignature}
	if p.InlineData != nil {
		out.InlineData = &googlegenai.Blob{Data: p.InlineData.Data, MIMEType: p.InlineData.MIMEType}
	}
	if p.FileData != nil {
		out.FileData = &googlegenai.FileData{FileURI: p.FileData.FileURI, MIMEType: p.FileData.MIMEType}
	}
	if p.FunctionCall != nil {
		out.FunctionCall = ToGenaiFunctionCall(p.FunctionCall)
	}
	if p.FunctionResponse != nil {
		out.FunctionResponse = ToGenaiFunctionResponse(p.FunctionResponse)
	}
	return out
}

func FromGenaiPart(p *googlegenai.Part) *adkmodel.Part {
	if p == nil {
		return nil
	}
	out := &adkmodel.Part{Text: p.Text, Thought: p.Thought, ThoughtSignature: p.ThoughtSignature}
	if p.InlineData != nil {
		out.InlineData = &adkmodel.Blob{Data: p.InlineData.Data, MIMEType: p.InlineData.MIMEType}
	}
	if p.FileData != nil {
		out.FileData = &adkmodel.FileData{FileURI: p.FileData.FileURI, MIMEType: p.FileData.MIMEType}
	}
	if p.FunctionCall != nil {
		out.FunctionCall = FromGenaiFunctionCall(p.FunctionCall)
	}
	if p.FunctionResponse != nil {
		out.FunctionResponse = FromGenaiFunctionResponse(p.FunctionResponse)
	}
	return out
}

func ToGenaiFunctionCall(fc *adkmodel.FunctionCall) *googlegenai.FunctionCall {
	if fc == nil {
		return nil
	}
	return &googlegenai.FunctionCall{ID: fc.ID, Name: fc.Name, Args: fc.Args, PartialArgs: ToGenaiPartialArgs(fc.PartialArgs), WillContinue: fc.WillContinue}
}

func FromGenaiFunctionCall(fc *googlegenai.FunctionCall) *adkmodel.FunctionCall {
	if fc == nil {
		return nil
	}
	return &adkmodel.FunctionCall{ID: fc.ID, Name: fc.Name, Args: fc.Args, PartialArgs: FromGenaiPartialArgs(fc.PartialArgs), WillContinue: fc.WillContinue}
}

func ToGenaiFunctionResponse(fr *adkmodel.FunctionResponse) *googlegenai.FunctionResponse {
	if fr == nil {
		return nil
	}
	out := &googlegenai.FunctionResponse{ID: fr.ID, Name: fr.Name, Response: fr.Response, WillContinue: fr.WillContinue}
	if fr.Scheduling != nil {
		out.Scheduling = googlegenai.FunctionResponseScheduling(*fr.Scheduling)
	}
	return out
}

func FromGenaiFunctionResponse(fr *googlegenai.FunctionResponse) *adkmodel.FunctionResponse {
	if fr == nil {
		return nil
	}
	out := &adkmodel.FunctionResponse{ID: fr.ID, Name: fr.Name, Response: fr.Response, WillContinue: fr.WillContinue}
	if fr.Scheduling != "" {
		s := adkmodel.FunctionResponseScheduling(fr.Scheduling)
		out.Scheduling = &s
	}
	return out
}

func ToGenaiPartialArgs(in []*adkmodel.PartialArg) []*googlegenai.PartialArg {
	out := make([]*googlegenai.PartialArg, 0, len(in))
	for _, a := range in {
		if a != nil {
			out = append(out, &googlegenai.PartialArg{JsonPath: a.JsonPath, StringValue: a.StringValue, NumberValue: a.NumberValue, BoolValue: a.BoolValue, NULLValue: a.NULLValue})
		} else {
			out = append(out, nil)
		}
	}
	return out
}
func FromGenaiPartialArgs(in []*googlegenai.PartialArg) []*adkmodel.PartialArg {
	out := make([]*adkmodel.PartialArg, 0, len(in))
	for _, a := range in {
		if a != nil {
			out = append(out, &adkmodel.PartialArg{JsonPath: a.JsonPath, StringValue: a.StringValue, NumberValue: a.NumberValue, BoolValue: a.BoolValue, NULLValue: a.NULLValue})
		} else {
			out = append(out, nil)
		}
	}
	return out
}

func ToGenaiConfig(c *adkmodel.GenerateConfig) *googlegenai.GenerateContentConfig {
	if c == nil {
		return nil
	}
	out := &googlegenai.GenerateContentConfig{SystemInstruction: ToGenaiContent(c.SystemInstruction), Temperature: c.Temperature, TopP: c.TopP, TopK: c.TopK, MaxOutputTokens: c.MaxOutputTokens, StopSequences: c.StopSequences, ResponseMIMEType: c.ResponseMIMEType, ResponseJsonSchema: c.ResponseSchema}
	if c.HTTPOptions != nil {
		out.HTTPOptions = &googlegenai.HTTPOptions{Headers: c.HTTPOptions.Headers}
	}
	for _, t := range c.Tools {
		out.Tools = append(out.Tools, ToGenaiTool(t))
	}
	return out
}

func FromGenaiConfig(c *googlegenai.GenerateContentConfig) *adkmodel.GenerateConfig {
	if c == nil {
		return nil
	}
	out := &adkmodel.GenerateConfig{SystemInstruction: FromGenaiContent(c.SystemInstruction), Temperature: c.Temperature, TopP: c.TopP, TopK: c.TopK, MaxOutputTokens: c.MaxOutputTokens, StopSequences: c.StopSequences, ResponseMIMEType: c.ResponseMIMEType, ResponseSchema: c.ResponseJsonSchema}
	if c.HTTPOptions != nil {
		out.HTTPOptions = &adkmodel.HTTPOptions{Headers: c.HTTPOptions.Headers}
	}
	for _, t := range c.Tools {
		out.Tools = append(out.Tools, FromGenaiTool(t))
	}
	return out
}

func ToGenaiTool(t *adkmodel.Tool) *googlegenai.Tool {
	if t == nil {
		return nil
	}
	out := &googlegenai.Tool{}
	for _, d := range t.FunctionDeclarations {
		out.FunctionDeclarations = append(out.FunctionDeclarations, ToGenaiFunctionDeclaration(d))
	}
	if t.GoogleSearch != nil {
		out.GoogleSearch = &googlegenai.GoogleSearch{}
	}
	if t.GoogleMaps != nil {
		out.GoogleMaps = &googlegenai.GoogleMaps{}
	}
	if t.URLContext != nil {
		out.URLContext = &googlegenai.URLContext{}
	}
	if t.Retrieval != nil {
		out.Retrieval = &googlegenai.Retrieval{}
		if r, ok := t.Retrieval.(*adkmodel.Retrieval); ok && r.ExternalAPI != nil {
			out.Retrieval.ExternalAPI = &googlegenai.ExternalAPI{}
		}
	}
	return out
}
func FromGenaiTool(t *googlegenai.Tool) *adkmodel.Tool {
	if t == nil {
		return nil
	}
	out := &adkmodel.Tool{}
	for _, d := range t.FunctionDeclarations {
		out.FunctionDeclarations = append(out.FunctionDeclarations, FromGenaiFunctionDeclaration(d))
	}
	return out
}
func ToGenaiFunctionDeclaration(d *adkmodel.FunctionDeclaration) *googlegenai.FunctionDeclaration {
	if d == nil {
		return nil
	}
	return &googlegenai.FunctionDeclaration{Name: d.Name, Description: d.Description, Parameters: ToGenaiSchema(d.Parameters), Response: ToGenaiSchema(d.Response), ParametersJsonSchema: d.ParametersJsonSchema, ResponseJsonSchema: d.ResponseJsonSchema}
}
func FromGenaiFunctionDeclaration(d *googlegenai.FunctionDeclaration) *adkmodel.FunctionDeclaration {
	if d == nil {
		return nil
	}
	return &adkmodel.FunctionDeclaration{Name: d.Name, Description: d.Description, Parameters: FromGenaiSchema(d.Parameters), Response: FromGenaiSchema(d.Response), ParametersJsonSchema: d.ParametersJsonSchema, ResponseJsonSchema: d.ResponseJsonSchema}
}

func ToGenaiSchema(s *adkmodel.Schema) *googlegenai.Schema {
	if s == nil {
		return nil
	}
	out := &googlegenai.Schema{Type: googlegenai.Type(s.Type), Description: s.Description, Required: s.Required}
	if len(s.Properties) > 0 {
		out.Properties = make(map[string]*googlegenai.Schema, len(s.Properties))
		for k, v := range s.Properties {
			out.Properties[k] = ToGenaiSchema(v)
		}
	}
	out.Items = ToGenaiSchema(s.Items)
	return out
}

func FromGenaiSchema(s *googlegenai.Schema) *adkmodel.Schema {
	if s == nil {
		return nil
	}
	out := &adkmodel.Schema{Type: adkmodel.Type(s.Type), Description: s.Description, Required: s.Required}
	if len(s.Properties) > 0 {
		out.Properties = make(map[string]*adkmodel.Schema, len(s.Properties))
		for k, v := range s.Properties {
			out.Properties[k] = FromGenaiSchema(v)
		}
	}
	out.Items = FromGenaiSchema(s.Items)
	return out
}

func FromGenaiUsageMetadata(u *googlegenai.GenerateContentResponseUsageMetadata) *adkmodel.UsageMetadata {
	if u == nil {
		return nil
	}
	return &adkmodel.UsageMetadata{PromptTokenCount: u.PromptTokenCount, CachedContentTokenCount: u.CachedContentTokenCount, CandidatesTokenCount: u.CandidatesTokenCount, TotalTokenCount: u.TotalTokenCount, PromptTokensDetails: fromModalityCounts(u.PromptTokensDetails), CacheTokensDetails: fromModalityCounts(u.CacheTokensDetails), CandidatesTokensDetails: fromModalityCounts(u.CandidatesTokensDetails)}
}
func fromModalityCounts(in []*googlegenai.ModalityTokenCount) []*adkmodel.ModalityTokenCount {
	out := make([]*adkmodel.ModalityTokenCount, 0, len(in))
	for _, m := range in {
		if m != nil {
			out = append(out, &adkmodel.ModalityTokenCount{Modality: string(m.Modality), TokenCount: m.TokenCount})
		} else {
			out = append(out, nil)
		}
	}
	return out
}

func FromGenaiGenerateContentResponse(res *googlegenai.GenerateContentResponse) *adkmodel.LLMResponse {
	usageMetadata := FromGenaiUsageMetadata(res.UsageMetadata)
	if len(res.Candidates) > 0 && res.Candidates[0] != nil {
		c := res.Candidates[0]
		r := &adkmodel.LLMResponse{Content: FromGenaiContent(c.Content), GroundingMetadata: fromGenaiGroundingMetadata(c.GroundingMetadata), CitationMetadata: fromGenaiCitationMetadata(c.CitationMetadata), FinishReason: adkmodel.FinishReason(c.FinishReason), AvgLogprobs: c.AvgLogprobs, LogprobsResult: fromGenaiLogprobsResult(c.LogprobsResult), UsageMetadata: usageMetadata, ModelVersion: res.ModelVersion}
		if (c.Content != nil && len(c.Content.Parts) > 0) || c.FinishReason == googlegenai.FinishReasonStop {
			return r
		}
		r.ErrorCode = string(c.FinishReason)
		r.ErrorMessage = c.FinishMessage
		return r
	}
	if res.PromptFeedback != nil {
		return &adkmodel.LLMResponse{ErrorCode: string(res.PromptFeedback.BlockReason), ErrorMessage: res.PromptFeedback.BlockReasonMessage, UsageMetadata: usageMetadata, ModelVersion: res.ModelVersion}
	}
	return &adkmodel.LLMResponse{ErrorCode: "UNKNOWN_ERROR", ErrorMessage: "Unknown error.", UsageMetadata: usageMetadata, ModelVersion: res.ModelVersion}
}

func fromGenaiCitationMetadata(c *googlegenai.CitationMetadata) *adkmodel.CitationMetadata {
	if c == nil {
		return nil
	}
	out := &adkmodel.CitationMetadata{}
	for _, ct := range c.Citations {
		if ct != nil {
			out.Citations = append(out.Citations, &adkmodel.Citation{Title: ct.Title, URI: ct.URI})
		}
	}
	return out
}
func fromGenaiGroundingMetadata(g *googlegenai.GroundingMetadata) *adkmodel.GroundingMetadata {
	if g == nil {
		return nil
	}
	return &adkmodel.GroundingMetadata{Raw: g}
}
func fromGenaiLogprobsResult(l *googlegenai.LogprobsResult) *adkmodel.LogprobsResult {
	if l == nil {
		return nil
	}
	return &adkmodel.LogprobsResult{Raw: l}
}
