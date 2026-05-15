package model

import "net/http"

const (
	RoleUser  Role = "user"
	RoleModel Role = "model"
)

type Role string

type FinishReason string

const (
	FinishReasonStop               FinishReason = "STOP"
	FinishReasonUnexpectedToolCall FinishReason = "UNEXPECTED_TOOL_CALL"
)

type Content struct {
	Parts []*Part `json:"parts,omitempty"`
	Role  Role    `json:"role,omitempty"`
}

type Part struct {
	Text                string               `json:"text,omitempty"`
	InlineData          *Blob                `json:"inlineData,omitempty"`
	FileData            *FileData            `json:"fileData,omitempty"`
	FunctionCall        *FunctionCall        `json:"functionCall,omitempty"`
	FunctionResponse    *FunctionResponse    `json:"functionResponse,omitempty"`
	ExecutableCode      *ExecutableCode      `json:"executableCode,omitempty"`
	CodeExecutionResult *CodeExecutionResult `json:"codeExecutionResult,omitempty"`
	Thought             bool                 `json:"thought,omitempty"`
	ThoughtSignature    []byte               `json:"thoughtSignature,omitempty"`
}

type Blob struct {
	Data        []byte `json:"data,omitempty"`
	MIMEType    string `json:"mimeType,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
}

type FileData struct {
	FileURI  string `json:"fileUri,omitempty"`
	MIMEType string `json:"mimeType,omitempty"`
}

type ExecutableCode struct {
	Language string `json:"language,omitempty"`
	Code     string `json:"code,omitempty"`
}

type CodeExecutionResult struct {
	Outcome string `json:"outcome,omitempty"`
	Output  string `json:"output,omitempty"`
}

type FunctionCall struct {
	ID           string         `json:"id,omitempty"`
	Name         string         `json:"name,omitempty"`
	Args         map[string]any `json:"args,omitempty"`
	PartialArgs  []*PartialArg  `json:"partialArgs,omitempty"`
	WillContinue *bool          `json:"willContinue,omitempty"`
}

type PartialArg struct {
	JsonPath    string   `json:"jsonPath,omitempty"`
	StringValue string   `json:"stringValue,omitempty"`
	NumberValue *float64 `json:"numberValue,omitempty"`
	BoolValue   *bool    `json:"boolValue,omitempty"`
	NULLValue   string   `json:"nullValue,omitempty"`
}

type FunctionResponse struct {
	ID           string                      `json:"id,omitempty"`
	Name         string                      `json:"name,omitempty"`
	Response     map[string]any              `json:"response,omitempty"`
	WillContinue *bool                       `json:"willContinue,omitempty"`
	Scheduling   *FunctionResponseScheduling `json:"scheduling,omitempty"`
}

type FunctionResponseScheduling string

const FunctionResponseSchedulingInterrupt FunctionResponseScheduling = "INTERRUPT"

type Tool struct {
	FunctionDeclarations []*FunctionDeclaration `json:"functionDeclarations,omitempty"`
	GoogleSearch         any                    `json:"googleSearch,omitempty"`
	Retrieval            any                    `json:"retrieval,omitempty"`
	GoogleMaps           any                    `json:"googleMaps,omitempty"`
	URLContext           any                    `json:"urlContext,omitempty"`
}

type FunctionDeclaration struct {
	Parameters           *Schema `json:"parameters,omitempty"`
	Response             *Schema `json:"response,omitempty"`
	Name                 string  `json:"name,omitempty"`
	Description          string  `json:"description,omitempty"`
	ParametersJsonSchema any     `json:"parametersJsonSchema,omitempty"`
	ResponseJsonSchema   any     `json:"responseJsonSchema,omitempty"`
}

type GenerateConfig struct {
	SystemInstruction *Content     `json:"systemInstruction,omitempty"`
	Temperature       *float32     `json:"temperature,omitempty"`
	TopP              *float32     `json:"topP,omitempty"`
	TopK              *float32     `json:"topK,omitempty"`
	MaxOutputTokens   int32        `json:"maxOutputTokens,omitempty"`
	StopSequences     []string     `json:"stopSequences,omitempty"`
	ResponseMIMEType  string       `json:"responseMimeType,omitempty"`
	ResponseSchema    any          `json:"responseSchema,omitempty"`
	Tools             []*Tool      `json:"tools,omitempty"`
	HTTPOptions       *HTTPOptions `json:"-"`
	ThinkingConfig    any          `json:"thinkingConfig,omitempty"`
}

type HTTPOptions struct {
	Headers http.Header
}

type UsageMetadata struct {
	PromptTokenCount        int32                 `json:"promptTokenCount,omitempty"`
	CachedContentTokenCount int32                 `json:"cachedContentTokenCount,omitempty"`
	CandidatesTokenCount    int32                 `json:"candidatesTokenCount,omitempty"`
	TotalTokenCount         int32                 `json:"totalTokenCount,omitempty"`
	PromptTokensDetails     []*ModalityTokenCount `json:"promptTokensDetails,omitempty"`
	CacheTokensDetails      []*ModalityTokenCount `json:"cacheTokensDetails,omitempty"`
	CandidatesTokensDetails []*ModalityTokenCount `json:"candidatesTokensDetails,omitempty"`
}

type ModalityTokenCount struct {
	Modality   string `json:"modality,omitempty"`
	TokenCount int32  `json:"tokenCount,omitempty"`
}

type CitationMetadata struct {
	Citations []*Citation `json:"citations,omitempty"`
}
type Citation struct {
	Title string `json:"title,omitempty"`
	URI   string `json:"uri,omitempty"`
}
type GroundingMetadata struct {
	WebSearchQueries             []string            `json:"webSearchQueries,omitempty"`
	RetrievalQueries             []string            `json:"retrievalQueries,omitempty"`
	GoogleMapsWidgetContextToken string              `json:"googleMapsWidgetContextToken,omitempty"`
	SearchEntryPoint             *SearchEntryPoint   `json:"searchEntryPoint,omitempty"`
	RetrievalMetadata            *RetrievalMetadata  `json:"retrievalMetadata,omitempty"`
	GroundingChunks              []*GroundingChunk   `json:"groundingChunks,omitempty"`
	GroundingSupports            []*GroundingSupport `json:"groundingSupports,omitempty"`
	Raw                          any                 `json:"-"`
}
type LogprobsResult struct {
	Raw any `json:"-"`
}

func NewPartFromText(text string) *Part { return &Part{Text: text} }
func NewContentFromText(text string, role Role) *Content {
	return &Content{Role: role, Parts: []*Part{{Text: text}}}
}
func NewContentFromParts(parts []*Part, role Role) *Content {
	return &Content{Role: role, Parts: parts}
}
func NewContentFromBytes(data []byte, mimeType string, role Role) *Content {
	return &Content{Role: role, Parts: []*Part{{InlineData: &Blob{Data: data, MIMEType: mimeType}}}}
}
func NewPartFromFunctionCall(name string, args map[string]any) *Part {
	return &Part{FunctionCall: &FunctionCall{Name: name, Args: args}}
}
func NewPartFromFunctionResponse(name string, response map[string]any) *Part {
	return &Part{FunctionResponse: &FunctionResponse{Name: name, Response: response}}
}
func NewContentFromFunctionCall(name string, args map[string]any, role Role) *Content {
	return NewContentFromParts([]*Part{NewPartFromFunctionCall(name, args)}, role)
}
func NewContentFromFunctionResponse(name string, response map[string]any, role Role) *Content {
	return NewContentFromParts([]*Part{NewPartFromFunctionResponse(name, response)}, role)
}
func Text(text string) []*Content { return []*Content{NewContentFromText(text, RoleUser)} }

// Backward-compatible names used while internal packages migrate off genai.
type GenerateContentConfig = GenerateConfig
type GenerateContentResponseUsageMetadata = UsageMetadata

type Backend string

const (
	BackendUnspecified Backend = "BACKEND_UNSPECIFIED"
	BackendGeminiAPI   Backend = "GEMINI_API"
	BackendVertexAI    Backend = "VERTEX_AI"
)

type Type string

const (
	TypeObject  Type = "OBJECT"
	TypeString  Type = "STRING"
	TypeInteger Type = "INTEGER"
)

type Schema struct {
	Type        Type               `json:"type,omitempty"`
	Description string             `json:"description,omitempty"`
	Properties  map[string]*Schema `json:"properties,omitempty"`
	Required    []string           `json:"required,omitempty"`
	Items       *Schema            `json:"items,omitempty"`
}

type GoogleSearch struct{}
type Retrieval struct {
	ExternalAPI *ExternalAPI `json:"externalApi,omitempty"`
}
type ExternalAPI struct{}

type SearchEntryPoint struct {
	RenderedContent string `json:"renderedContent,omitempty"`
	SDKBlob         []byte `json:"sdkBlob,omitempty"`
}
type RetrievalMetadata struct {
	GoogleSearchDynamicRetrievalScore float32 `json:"googleSearchDynamicRetrievalScore,omitempty"`
}
type GroundingChunk struct {
	Maps             *GroundingChunkMaps             `json:"maps,omitempty"`
	RetrievedContext *GroundingChunkRetrievedContext `json:"retrievedContext,omitempty"`
	Web              *GroundingChunkWeb              `json:"web,omitempty"`
}
type GroundingChunkMaps struct {
	URI                string                                `json:"uri,omitempty"`
	Title              string                                `json:"title,omitempty"`
	Text               string                                `json:"text,omitempty"`
	PlaceID            string                                `json:"placeId,omitempty"`
	PlaceAnswerSources *GroundingChunkMapsPlaceAnswerSources `json:"placeAnswerSources,omitempty"`
}
type GroundingChunkMapsPlaceAnswerSources struct {
	ReviewSnippets []*GroundingChunkMapsPlaceAnswerSourcesReviewSnippet `json:"reviewSnippets,omitempty"`
}
type GroundingChunkMapsPlaceAnswerSourcesReviewSnippet struct {
	Review        string `json:"review,omitempty"`
	GoogleMapsURI string `json:"googleMapsUri,omitempty"`
	Title         string `json:"title,omitempty"`
}
type GroundingChunkRetrievedContext struct {
	URI          string    `json:"uri,omitempty"`
	Title        string    `json:"title,omitempty"`
	Text         string    `json:"text,omitempty"`
	DocumentName string    `json:"documentName,omitempty"`
	RAGChunk     *RAGChunk `json:"ragChunk,omitempty"`
}
type RAGChunk struct {
	Text     string            `json:"text,omitempty"`
	PageSpan *RAGChunkPageSpan `json:"pageSpan,omitempty"`
}
type RAGChunkPageSpan struct {
	FirstPage int32 `json:"firstPage,omitempty"`
	LastPage  int32 `json:"lastPage,omitempty"`
}
type GroundingChunkWeb struct {
	URI   string `json:"uri,omitempty"`
	Title string `json:"title,omitempty"`
}
type GroundingSupport struct {
	GroundingChunkIndices []int32   `json:"groundingChunkIndices,omitempty"`
	ConfidenceScores      []float32 `json:"confidenceScores,omitempty"`
	Segment               *Segment  `json:"segment,omitempty"`
}
type Segment struct {
	PartIndex  int32  `json:"partIndex,omitempty"`
	StartIndex int32  `json:"startIndex,omitempty"`
	EndIndex   int32  `json:"endIndex,omitempty"`
	Text       string `json:"text,omitempty"`
}

const (
	TypeBoolean Type = "BOOLEAN"
	TypeNumber  Type = "NUMBER"
	TypeArray   Type = "ARRAY"
)

type URLContext struct{}
type GoogleMaps struct{}

func NewPartFromBytes(data []byte, mimeType string) *Part {
	return &Part{InlineData: &Blob{Data: data, MIMEType: mimeType}}
}
