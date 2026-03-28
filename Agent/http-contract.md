# HTTP Contract

## Endpoints

- `Open WebUI`
  - listens on `127.0.0.1:3000`
  - sends all model traffic to `Agent Gateway`
- `Agent Gateway`
  - listens on `127.0.0.1:8090`
  - exposes OpenAI-compatible chat endpoints to Open WebUI
  - calls `llama-server` on `127.0.0.1:8080` first, then falls back to the WSL host-route IP
  - calls `Tool Router` on `127.0.0.1:8091`
- `Tool Router`
  - listens on `127.0.0.1:8091`
  - accepts internal tool execution requests from `Agent Gateway`
- `llama-server`
  - listens on `127.0.0.1:8080`
  - serves the single GGUF model

## Data Flow

1. User prompt enters Open WebUI.
2. Open WebUI forwards the request to Agent Gateway.
3. Gateway loads `awdp` core plus `web` and/or `pwn` plugin fragments.
4. Gateway retrieves local context and queries `llama-server`.
5. If the model emits `tool_call`, Gateway validates it against `action-envelope.schema.json`.
6. Gateway forwards validated calls to Tool Router.
7. Tool Router invokes the mapped MCP tool.
8. Tool result is summarized and returned to Gateway.
9. Gateway performs a final model pass and returns the answer to Open WebUI.

## Action Envelope

All model-controlled actions are expressed through JSON:

```json
{
  "type": "answer | tool_call",
  "tool": "",
  "arguments": {},
  "reason": "why this action is correct",
  "content": "final user-facing answer when type=answer"
}
```

Rules:
- `type="answer"`: `content` must contain the final answer body, `tool=""`.
- `type="tool_call"`: `content=""`, `tool` and `arguments` must pass Gateway and Tool Router validation.
- Malformed JSON triggers one repair retry, then fail-closed fallback.

## Non-Goals for V1

- No direct Open WebUI -> MCP path
- No direct Open WebUI -> `llama-server` path
- No LAN exposure
- No external web retrieval
