                 ADK Runtime 内部
        ┌──────────────────────────────┐
        │ model.LLMRequest             │
        │ - Contents: genai.Content    │
        │ - Config: genai.Config       │
        │ - Tools: runtime tools       │
        └──────────────┬───────────────┘
                       │
                       ▼
              model.LLM interface
                       │
        ┌──────────────┼────────────────┐
        │              │                │
        ▼              ▼                ▼
   geminiModel    apigeeModel     openAICompatModel
        │              │                │
        │              │                ▼
        │              │       genai -> OpenAI JSON
        │              │                │
        ▼              ▼                ▼
 Gemini SDK      Apigee Proxy      /v1/chat/completions
        │              │                │
        ▼              ▼                ▼
   genai resp     genai resp       OpenAI JSON resp
        │              │                │
        └──────────────┼────────────────┘
                       ▼
        ┌──────────────────────────────┐
        │ model.LLMResponse            │
        │ - Content: genai.Content     │
        │ - FinishReason              │
        │ - ModelVersion              │
        └──────────────────────────────┘