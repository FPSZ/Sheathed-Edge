import { useEffect, useMemo, useState } from "react";

import { PageHeader } from "../components/PageHeader";
import { apiGet, apiPost } from "../lib/api";
import type {
  MCPDiscoveredTool,
  MCPDiscoverToolsResponse,
  MCPOpenWebUIPreviewResponse,
  MCPServerProfile,
  MCPServersResponse,
  MCPValidateResponse,
} from "../lib/types";

type ActionState = "idle" | "pending";

function newMCPServerDraft(): MCPServerProfile {
  return {
    id: `mcp-${Math.random().toString(36).slice(2, 8)}`,
    label: "New MCP Server",
    enabled: true,
    kind: "native_streamable_http",
    description: "",
    plugin_scope: ["awdp"],
    auth_type: "none",
    auth_payload: {},
    disabled_tools: [],
    timeout_ms: 30000,
    verify_tls: true,
    notes: "",
    url: "",
    command: [],
    workdir: "",
    env: {},
    headers: {},
  };
}

export function MCPPage() {
  const [data, setData] = useState<MCPServersResponse | null>(null);
  const [preview, setPreview] = useState<MCPOpenWebUIPreviewResponse | null>(null);
  const [servers, setServers] = useState<MCPServerProfile[]>([]);
  const [editingId, setEditingId] = useState("");
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [actionState, setActionState] = useState<ActionState>("idle");
  const [validateResult, setValidateResult] = useState<MCPValidateResponse | null>(null);

  async function load() {
    try {
      const [serversResp, previewResp] = await Promise.all([
        apiGet<MCPServersResponse>("/internal/admin/mcp/servers"),
        apiGet<MCPOpenWebUIPreviewResponse>("/internal/admin/mcp/openwebui-preview"),
      ]);
      setData(serversResp);
      setPreview(previewResp);
      setServers(serversResp.servers.map((item) => ({ ...item.profile })));
      setEditingId((current) => {
        if (current && serversResp.servers.some((item) => item.profile.id === current)) {
          return current;
        }
        return "";
      });
      setError("");
    } catch (err) {
      setError((err as Error).message);
    }
  }

  useEffect(() => {
    load();
  }, []);

  const editingServer = useMemo(
    () => servers.find((item) => item.id === editingId) ?? null,
    [editingId, servers],
  );

  const editingState = useMemo(
    () => data?.servers.find((item) => item.profile.id === editingId) ?? null,
    [data, editingId],
  );

  function updateServer(serverId: string, patch: Partial<MCPServerProfile>) {
    setServers((current) =>
      current.map((item) => (item.id === serverId ? { ...item, ...patch } : item)),
    );
  }

  function updateEditingServer(patch: Partial<MCPServerProfile>) {
    if (!editingId) {
      return;
    }
    updateServer(editingId, patch);
  }

  function addServer() {
    const draft = newMCPServerDraft();
    setServers((current) => [...current, draft]);
    setEditingId(draft.id);
    setValidateResult(null);
    setNotice("");
  }

  function removeEditingServer() {
    if (!editingId) {
      return;
    }
    setServers((current) => current.filter((item) => item.id !== editingId));
    setEditingId("");
    setValidateResult(null);
    setNotice("");
  }

  async function saveServers() {
    setActionState("pending");
    setError("");
    setNotice("");
    try {
      await apiPost<MCPServersResponse>("/internal/admin/mcp/servers", { servers });
      await load();
      setNotice(
        "MCP server profiles saved. Restart Open WebUI after changes if you want connection sync to refresh.",
      );
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setActionState("idle");
    }
  }

  async function validateEditingServer() {
    if (!editingServer) {
      return;
    }
    setActionState("pending");
    setError("");
    setNotice("");
    try {
      const response = await apiPost<MCPValidateResponse>("/internal/admin/mcp/servers/validate", {
        server: editingServer,
      });
      setValidateResult(response);
      setNotice(response.summary);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setActionState("idle");
    }
  }

  async function discoverEditingServerTools() {
    if (!editingServer) {
      return;
    }
    setActionState("pending");
    setError("");
    setNotice("");
    try {
      const response = await apiPost<MCPDiscoverToolsResponse>(
        "/internal/admin/mcp/servers/discover-tools",
        { server_id: editingServer.id },
      );
      setNotice(response.summary);
      await load();
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setActionState("idle");
    }
  }

  function updateListField(kind: "plugin_scope" | "command", value: string) {
    updateEditingServer({ [kind]: parseLines(value) } as Partial<MCPServerProfile>);
  }

  function updateMapField(kind: "env" | "headers" | "auth_payload", value: string) {
    updateEditingServer({ [kind]: parseKeyValueMap(value) } as Partial<MCPServerProfile>);
  }

  function toggleDiscoveredTool(tool: MCPDiscoveredTool) {
    if (!editingServer) {
      return;
    }
    const disabled = new Set(editingServer.disabled_tools);
    if (disabled.has(tool.name)) {
      disabled.delete(tool.name);
    } else {
      disabled.add(tool.name);
    }
    updateEditingServer({ disabled_tools: Array.from(disabled).sort() });
  }

  function toggleServerEnabled(serverId: string, enabled: boolean) {
    updateServer(serverId, { enabled });
  }

  async function copyPreview() {
    if (!preview?.tool_server_connections_json) {
      return;
    }
    await navigator.clipboard.writeText(preview.tool_server_connections_json);
    setNotice("Open WebUI preview JSON copied.");
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="MCP Servers"
        description="Register native MCP and mcpo-backed servers, validate them, discover tools, and preview what Open WebUI will receive on restart."
        action={
          <div className="flex shrink-0 flex-nowrap items-center gap-2">
            <button className="admin-button" type="button" onClick={addServer}>
              New Server
            </button>
            <button
              className="admin-button"
              type="button"
              disabled={actionState === "pending"}
              onClick={saveServers}
            >
              Save Servers
            </button>
          </div>
        }
      />

      {error ? (
        <div className="admin-surface rounded-3xl bg-rose-50 p-4 text-sm text-rose-700">
          {error}
        </div>
      ) : null}

      {notice ? (
        <div className="admin-surface rounded-3xl bg-emerald-50 p-4 text-sm text-emerald-700">
          {notice}
        </div>
      ) : null}

      <section className="admin-surface rounded-3xl px-5 py-4">
        <div className="flex items-center justify-between gap-3">
          <div>
            <div className="text-sm font-semibold text-slate-900">Server List</div>
            <div className="mt-1 text-xs text-slate-500">
              {servers.length} configured MCP endpoints
            </div>
          </div>
          <button
            className="admin-button danger"
            type="button"
            disabled={!editingServer}
            onClick={removeEditingServer}
          >
            Remove
          </button>
        </div>

        <div className="mt-5">
          {servers.map((server, index) => {
            const state = data?.servers.find((item) => item.profile.id === server.id);
            const editing = editingId === server.id;
            const status = state?.runtime_status.status ?? (server.enabled ? "enabled" : "disabled");

            return (
              <div
                key={server.id}
                className={`admin-mcp-row ${editing ? "is-editing" : ""} ${index > 0 ? "with-divider" : ""}`}
              >
                <div className="admin-mcp-row-main">
                  <div className="admin-mcp-row-title">
                    <div className="text-sm font-medium text-slate-950">{server.label}</div>
                    <div className="mt-1 text-xs text-slate-500">
                      {server.kind} · {status}
                    </div>
                  </div>
                  <div className="admin-mcp-row-meta">
                    <span>{server.plugin_scope.join(", ") || "no scope"}</span>
                    <span>{state?.discovered_tools.length ?? 0} tools</span>
                  </div>
                </div>

                <div className="admin-mcp-row-actions">
                  <button
                    className={`admin-button ${editing ? "" : "ghost"}`}
                    type="button"
                    onClick={() => {
                      setEditingId((current) => (current === server.id ? "" : server.id));
                      setValidateResult(null);
                    }}
                  >
                    {editing ? "Close" : "Edit"}
                  </button>
                  <label className="admin-switch" aria-label={`Toggle ${server.label}`}>
                    <input
                      checked={server.enabled}
                      type="checkbox"
                      onChange={(event) => toggleServerEnabled(server.id, event.target.checked)}
                    />
                    <span className="admin-switch-track" />
                  </label>
                </div>
              </div>
            );
          })}

          {servers.length === 0 ? (
            <div className="py-6 text-sm text-slate-500">No MCP servers yet. Create one to get started.</div>
          ) : null}
        </div>
      </section>

      {editingServer ? (
        <section className="space-y-4">
          <section className="admin-surface rounded-3xl p-5">
            <div className="flex flex-wrap items-start justify-between gap-3">
              <div>
                <div className="text-sm font-semibold text-slate-900">Edit Server</div>
                <div className="mt-1 text-xs text-slate-500">
                  The editor only appears after clicking a row. Save still happens at the page
                  level.
                </div>
              </div>
              <div className="flex items-center gap-2">
                <button
                  className="admin-button"
                  type="button"
                  disabled={actionState === "pending"}
                  onClick={validateEditingServer}
                >
                  Validate
                </button>
                <button
                  className="admin-button"
                  type="button"
                  disabled={actionState === "pending"}
                  onClick={discoverEditingServerTools}
                >
                  Discover Tools
                </button>
              </div>
            </div>

            <div className="mt-5 grid gap-4 md:grid-cols-2">
              <Field label="ID">
                <input
                  className="admin-input"
                  value={editingServer.id}
                  onChange={(event) => updateEditingServer({ id: event.target.value })}
                />
              </Field>
              <Field label="Label">
                <input
                  className="admin-input"
                  value={editingServer.label}
                  onChange={(event) => updateEditingServer({ label: event.target.value })}
                />
              </Field>
              <Field label="Kind">
                <select
                  className="admin-input"
                  value={editingServer.kind}
                  onChange={(event) =>
                    updateEditingServer({
                      kind: event.target.value as MCPServerProfile["kind"],
                    })
                  }
                >
                  <option value="native_streamable_http">native_streamable_http</option>
                  <option value="mcpo_stdio">mcpo_stdio</option>
                  <option value="mcpo_sse">mcpo_sse</option>
                </select>
              </Field>
              <Field label="Enabled">
                <div className="admin-mcp-toggle-field">
                  <span>Sync this server to Open WebUI</span>
                  <label className="admin-switch" aria-label="Toggle server enabled">
                    <input
                      checked={editingServer.enabled}
                      type="checkbox"
                      onChange={(event) => updateEditingServer({ enabled: event.target.checked })}
                    />
                    <span className="admin-switch-track" />
                  </label>
                </div>
              </Field>
              <Field label="Plugin Scope">
                <textarea
                  className="admin-input min-h-24"
                  value={editingServer.plugin_scope.join("\n")}
                  onChange={(event) => updateListField("plugin_scope", event.target.value)}
                />
              </Field>
              <Field label="Disabled Tools">
                <textarea
                  className="admin-input min-h-24"
                  value={editingServer.disabled_tools.join("\n")}
                  onChange={(event) =>
                    updateEditingServer({
                      disabled_tools: parseLines(event.target.value),
                    })
                  }
                />
              </Field>
              <Field label="URL">
                <input
                  className="admin-input"
                  value={editingServer.url ?? ""}
                  onChange={(event) => updateEditingServer({ url: event.target.value })}
                />
              </Field>
              <Field label="Workdir">
                <input
                  className="admin-input"
                  value={editingServer.workdir ?? ""}
                  onChange={(event) => updateEditingServer({ workdir: event.target.value })}
                />
              </Field>
              <Field label="Command">
                <textarea
                  className="admin-input min-h-24"
                  value={(editingServer.command ?? []).join("\n")}
                  onChange={(event) => updateListField("command", event.target.value)}
                />
              </Field>
              <Field label="Timeout (ms)">
                <input
                  className="admin-input"
                  type="number"
                  value={editingServer.timeout_ms}
                  onChange={(event) =>
                    updateEditingServer({ timeout_ms: Number(event.target.value) || 30000 })
                  }
                />
              </Field>
              <Field label="Auth Type">
                <select
                  className="admin-input"
                  value={editingServer.auth_type}
                  onChange={(event) =>
                    updateEditingServer({
                      auth_type: event.target.value as MCPServerProfile["auth_type"],
                    })
                  }
                >
                  <option value="none">none</option>
                  <option value="bearer">bearer</option>
                  <option value="basic">basic</option>
                  <option value="header">header</option>
                </select>
              </Field>
              <Field label="Verify TLS">
                <div className="admin-mcp-toggle-field">
                  <span>Reject invalid certificates</span>
                  <label className="admin-switch" aria-label="Toggle TLS verification">
                    <input
                      checked={editingServer.verify_tls}
                      type="checkbox"
                      onChange={(event) => updateEditingServer({ verify_tls: event.target.checked })}
                    />
                    <span className="admin-switch-track" />
                  </label>
                </div>
              </Field>
              <Field label="Auth Payload">
                <textarea
                  className="admin-input min-h-24"
                  value={formatMap(editingServer.auth_payload)}
                  onChange={(event) => updateMapField("auth_payload", event.target.value)}
                />
              </Field>
              <Field label="Headers">
                <textarea
                  className="admin-input min-h-24"
                  value={formatMap(editingServer.headers)}
                  onChange={(event) => updateMapField("headers", event.target.value)}
                />
              </Field>
              <Field label="Env">
                <textarea
                  className="admin-input min-h-24"
                  value={formatMap(editingServer.env)}
                  onChange={(event) => updateMapField("env", event.target.value)}
                />
              </Field>
              <Field label="Description">
                <textarea
                  className="admin-input min-h-24"
                  value={editingServer.description ?? ""}
                  onChange={(event) => updateEditingServer({ description: event.target.value })}
                />
              </Field>
              <Field label="Notes">
                <textarea
                  className="admin-input min-h-24"
                  value={editingServer.notes ?? ""}
                  onChange={(event) => updateEditingServer({ notes: event.target.value })}
                />
              </Field>
            </div>

            {validateResult ? (
              <div className="admin-mcp-note mt-5">
                <div className="font-medium text-slate-900">{validateResult.summary}</div>
                <div className="mt-1 text-xs text-slate-500">
                  {validateResult.effective_openwebui_type || "unknown"} ·{" "}
                  {validateResult.effective_connection_url || "no connection url"}
                </div>
              </div>
            ) : null}
          </section>

          <section className="admin-surface rounded-3xl p-5">
            <div className="text-sm font-semibold text-slate-900">Tool Toggles</div>
            <div className="mt-1 text-xs text-slate-500">
              Discover first, then decide which tools should stay visible to the model.
            </div>

            <div className="mt-4 space-y-2">
              {(editingState?.discovered_tools ?? []).map((tool) => {
                const checked = !editingServer.disabled_tools.includes(tool.name);
                return (
                  <div key={tool.name} className="admin-mcp-tool-row">
                    <div className="min-w-0">
                      <div className="text-sm font-medium text-slate-900">{tool.name}</div>
                      <div className="mt-1 text-xs leading-5 text-slate-500">
                        {tool.description || "No description from discovery."}
                      </div>
                    </div>
                    <label className="admin-switch" aria-label={`Toggle ${tool.name}`}>
                      <input checked={checked} type="checkbox" onChange={() => toggleDiscoveredTool(tool)} />
                      <span className="admin-switch-track" />
                    </label>
                  </div>
                );
              })}
            </div>

            {(editingState?.discovered_tools ?? []).length === 0 ? (
              <div className="mt-4 text-sm text-slate-500">
                No discovered tools yet. Run "Discover Tools" first.
              </div>
            ) : null}
          </section>
        </section>
      ) : null}

      <section className="admin-surface rounded-3xl p-5">
        <div className="flex items-start justify-between gap-3">
          <div>
            <div className="text-sm font-semibold text-slate-900">Open WebUI Preview</div>
            <div className="mt-1 text-xs text-slate-500">
              This is the generated connection payload that Open WebUI will consume after restart.
            </div>
          </div>
          <button className="admin-button" type="button" onClick={copyPreview}>
            Copy JSON
          </button>
        </div>
        <pre className="admin-mcp-preview mt-4">
          {preview?.tool_server_connections_json ?? "[]"}
        </pre>
      </section>
    </div>
  );
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label className="space-y-2">
      <div className="text-xs font-semibold uppercase tracking-[0.16em] text-slate-400">{label}</div>
      {children}
    </label>
  );
}

function parseLines(value: string) {
  return value
    .split(/\r?\n/)
    .map((item) => item.trim())
    .filter(Boolean);
}

function parseKeyValueMap(value: string) {
  const entries = value
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean)
    .map((line) => {
      const pivot = line.indexOf("=");
      if (pivot === -1) {
        return null;
      }
      return [line.slice(0, pivot).trim(), line.slice(pivot + 1).trim()] as const;
    })
    .filter((item): item is readonly [string, string] => Boolean(item && item[0] && item[1]));

  return Object.fromEntries(entries);
}

function formatMap(values: Record<string, string> | undefined) {
  if (!values) {
    return "";
  }

  return Object.entries(values)
    .map(([key, value]) => `${key}=${value}`)
    .join("\n");
}
