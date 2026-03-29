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
  - exposes an OpenAPI terminal tool server for Open WebUI external tools at `/openapi.json`
- `llama-server`
  - listens on `127.0.0.1:8080`
  - serves the single GGUF model

## Data Flow

1. User prompt enters Open WebUI.
2. Open WebUI forwards the request to Agent Gateway.
3. Gateway loads `awdp` core plus `web` and/or `pwn` plugin fragments.
4. Gateway retrieves local context and queries `llama-server`.
5. If the active path uses Gateway-managed legacy tools, the model may emit `tool_call` and Gateway validates it against `action-envelope.schema.json`.
6. Gateway forwards validated legacy tool calls to Tool Router.
7. Tool Router invokes the mapped internal tool executor and returns the result to Gateway.
8. For local terminal actions, Open WebUI may call the Tool Router OpenAPI terminal server directly as an external tool.
9. Gateway performs the final model pass for legacy tool turns and returns the answer to Open WebUI.

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
- Ordinary code blocks and shell snippets are not executable control messages.

## Non-Goals for V1

- No direct Open WebUI -> MCP path
- No direct Open WebUI -> `llama-server` path
- No LAN exposure
- No external web retrieval
- No code-block auto-execution fallback in Gateway
