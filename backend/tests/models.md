## 测试模型
```bash
curl -X POST "http://localhost:41027/v1/chat/completions" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "model": "qwen_qwen3.5-397b-a17b",
    "messages": [
      {
        "role": "user",
        "content": "回复 OK"
      }
    ],
    "temperature": 0.1,
    "max_tokens": 64
  }'
```


## tool-calling
```bash
curl.exe -X POST "http://localhost:41027/v1/chat/completions" `
  -H "Content-Type: application/json" `
  -H "Authorization: Bearer YOUR_API_KEY" `
  -d '{
    "model": "qwen_qwen3.5-397b-a17b",
    "messages": [
      {
        "role": "user",
        "content": "如果你需要更多信息，请调用 AskHuman 工具。问题：我要写一本都市爽文，但还没确定核心爽点。"
      }
    ],
    "tools": [
      {
        "type": "function",
        "function": {
          "name": "AskHuman",
          "description": "Ask the human for missing information before continuing.",
          "parameters": {
            "type": "object",
            "properties": {
              "reason": {
                "type": "string"
              },
              "questions": {
                "type": "array",
                "items": {
                  "type": "object",
                  "properties": {
                    "field": {
                      "type": "string"
                    },
                    "question": {
                      "type": "string"
                    },
                    "options": {
                      "type": "array",
                      "items": {
                        "type": "object",
                        "properties": {
                          "label": {
                            "type": "string"
                          },
                          "description": {
                            "type": "string"
                          }
                        },
                        "required": ["label"]
                      }
                    }
                  },
                  "required": ["question"]
                }
              }
            },
            "required": ["questions"]
          }
        }
      }
    ],
    "tool_choice": "auto",
    "temperature": 0.1,
    "max_tokens": 256
  }'

```




## powershell

### ok

```powershell
Invoke-RestMethod -Uri "http://localhost:41027/v1/chat/completions" `
  -Method Post `
  -Headers @{
    "Content-Type" = "application/json"
    "Authorization" = "Bearer YOUR_API_KEY"
  } `
  -Body '{
    "model": "qwen_qwen3.5-397b-a17b",
    "messages": [
      {
        "role": "user",
        "content": "回复 OK"
      }
    ],
    "temperature": 0.1,
    "max_tokens": 64
  }'
```

### tools

```powershell   
Invoke-RestMethod -Uri "http://localhost:41027/v1/chat/completions" `
  -Method Post `
  -Headers @{
    "Content-Type" = "application/json"
    "Authorization" = "Bearer YOUR_API_KEY"
  } `
  -Body '{
    "model": "qwen_qwen3.5-397b-a17b",
    "messages": [
      {
        "role": "user",
        "content": "如果你需要更多信息，请调用 AskHuman 工具。问题：我要写一本都市爽文，但还没确定核心爽点。"
      }
    ],
    "tools": [
      {
        "type": "function",
        "function": {
          "name": "AskHuman",
          "description": "Ask the human for missing information before continuing.",
          "parameters": {
            "type": "object",
            "properties": {
              "reason": {
                "type": "string"
              },
              "questions": {
                "type": "array",
                "items": {
                  "type": "object",
                  "properties": {
                    "field": {
                      "type": "string"
                    },
                    "question": {
                      "type": "string"
                    },
                    "options": {
                      "type": "array",
                      "items": {
                        "type": "object",
                        "properties": {
                          "label": {
                            "type": "string"
                          },
                          "description": {
                            "type": "string"
                          }
                        },
                        "required": ["label"]
                      }
                    }
                  },
                  "required": ["question"]
                }
              }
            },
            "required": ["questions"]
          }
        }
      }
    ],
    "tool_choice": "auto",
    "temperature": 0.1,
    "max_tokens": 256
  }'
```







