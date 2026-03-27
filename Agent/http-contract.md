# HTTP Contract

## Endpoints

- `Open WebUI`
  - listens on `127.0.0.1:3000`
  - sends all model traffic to `Agent Gateway`
- `Agent Gateway`
  - listens on `127.0.0.1:8090`
  - exposes OpenAI-compatible chat endpoints to Open WebUI
  - calls `llama-server` on `127.0.0.1:8080`
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
7. Tool Router invokes the mapped MCP server.
8. Tool result is summarized and returned to Gateway.
9. Gateway performs a final model pass and returns the answer to Open WebUI.

## Non-Goals for V1

- No direct Open WebUI -> MCP path
- No direct Open WebUI -> `llama-server` path
- No LAN exposure
- No external web retrieval

