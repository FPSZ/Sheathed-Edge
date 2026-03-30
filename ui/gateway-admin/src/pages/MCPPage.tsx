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
  const [selectedId, setSelectedId] = useState("");
  const [servers, setServers] = useState<MCPServerProfile[]>([]);
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
      setSelectedId((current) => {
        if (current && serversResp.servers.some((item) => item.profile.id === current)) {
          return current;
        }
        return serversResp.servers[0]?.profile.id ?? "";
      });
      setError("");
    } catch (err) {
      setError((err as Error).message);
    }
  }

  useEffect(() => {
    load();
  }, []);

  const selectedServer = useMemo(
    () => servers.find((item) => item.id === selectedId) ?? null,
    [selectedId, servers],
  );

  const selectedState = useMemo(
    () => data?.servers.find((item) => item.profile.id === selectedId) ?? null,
    [data, selectedId],
  );

  function updateSelectedServer(patch: Partial<MCPServerProfile>) {
    setServers((current) =>
      current.map((item) => (item.id === selectedId ? { ...item, ...patch } : item)),
    );
  }

  function addServer() {
    const draft = newMCPServerDraft();
    setServers((current) => [...current, draft]);
    setSelectedId(draft.id);
    setValidateResult(null);
    setNotice("");
  }

  function removeSelectedServer() {
    if (!selectedId) {
      return;
    }
    const next = servers.filter((item) => item.id !== selectedId);
    setServers(next);
    setSelectedId(next[0]?.id ?? "");
    setValidateResult(null);
  }

  async function saveServers() {
    setActionState("pending");
    setError("");
    setNotice("");
    try {
      await apiPost<MCPServersResponse>("/internal/admin/mcp/servers", { servers });
      await load();
      setNotice("MCP server profiles saved. Restart Open WebUI after changes if you want connection sync to refresh.");
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setActionState("idle");
    }
  }

  async function validateSelectedServer() {
    if (!selectedServer) {
      return;
    }
    setActionState("pending");
    setError("");
    setNotice("");
    try {
      const response = await apiPost<MCPValidateResponse>("/internal/admin/mcp/servers/validate", {
        server: selectedServer,
      });
      setValidateResult(response);
      setNotice(response.summary);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setActionState("idle");
    }
  }

  async function discoverSelectedServerTools() {
    if (!selectedServer) {
      return;
    }
    setActionState("pending");
    setError("");
    setNotice("");
    try {
      const response = await apiPost<MCPDiscoverToolsResponse>(
        "/internal/admin/mcp/servers/discover-tools",
        { server_id: selectedServer.id },
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
    const items = value
      .split(/\r?\n/)
      .map((item) => item.trim())
      .filter(Boolean);
    updateSelectedServer({ [kind]: items } as Partial<MCPServerProfile>);
  }

  function updateMapField(kind: "env" | "headers" | "auth_payload", value: string) {
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
    updateSelectedServer({ [kind]: Object.fromEntries(entries) } as Partial<MCPServerProfile>);
  }

  function toggleDiscoveredTool(tool: MCPDiscoveredTool) {
    if (!selectedServer) {
      return;
    }
    const disabled = new Set(selectedServer.disabled_tools);
    if (disabled.has(tool.name)) {
      disabled.delete(tool.name);
    } else {
      disabled.add(tool.name);
    }
    updateSelectedServer({ disabled_tools: Array.from(disabled).sort() });
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

      <section className="grid gap-4 xl:grid-cols-[19rem,minmax(0,1fr)]">
        <div className="admin-surface rounded-3xl p-4">
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
              disabled={!selectedServer}
              onClick={removeSelectedServer}
            >
              Remove
            </button>
          </div>

          <div className="mt-4 space-y-2">
            {servers.map((server) => {
              const state = data?.servers.find((item) => item.profile.id === server.id);
              const selected = selectedId === server.id;
              return (
                <button
                  key={server.id}
                  type="button"
                  className={`admin-log-item w-full text-left ${selected ? "active" : ""}`}
                  onClick={() => {
                    setSelectedId(server.id);
                    setValidateResult(null);
                  }}
                >
                  <div className="flex items-start justify-between gap-3">
                    <div>
                      <div className="text-sm font-medium text-slate-900">{server.label}</div>
                      <div className="mt-1 text-xs text-slate-500">{server.kind}</div>
                    </div>
                    <span className={`admin-badge ${server.enabled ? "success" : "muted"}`}>
                      {state?.runtime_status.status ?? (server.enabled ? "enabled" : "disabled")}
                    </span>
                  </div>
                </button>
              );
            })}
          </div>
        </div>

        <div className="space-y-4">
          <section className="admin-surface rounded-3xl p-5">
            <div className="flex items-start justify-between gap-3">
              <div>
                <div className="text-sm font-semibold text-slate-900">Server Editor</div>
                <div className="mt-1 text-xs text-slate-500">
                  Edit transport, auth, scope and timeout. Save writes to the MCP registry file.
                </div>
              </div>
              <div className="flex items-center gap-2">
                <button
                  className="admin-button"
                  type="button"
                  disabled={!selectedServer || actionState === "pending"}
                  onClick={validateSelectedServer}
                >
                  Validate
                </button>
                <button
                  className="admin-button"
                  type="button"
                  disabled={!selectedServer || actionState === "pending"}
                  onClick={discoverSelectedServerTools}
                >
                  Discover Tools
                </button>
              </div>
            </div>

            {selectedServer ? (
              <div className="mt-5 grid gap-4 md:grid-cols-2">
                <Field label="ID">
                  <input
                    className="admin-input"
                    value={selectedServer.id}
                    onChange={(event) => updateSelectedServer({ id: event.target.value })}
                  />
                </Field>
                <Field label="Label">
                  <input
                    className="admin-input"
                    value={selectedServer.label}
                    onChange={(event) => updateSelectedServer({ label: event.target.value })}
                  />
                </Field>
                <Field label="Kind">
                  <select
                    className="admin-input"
                    value={selectedServer.kind}
                    onChange={(event) =>
                      updateSelectedServer({
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
                  <label className="flex h-11 items-center gap-3 text-sm text-slate-700">
                    <input
                      checked={selectedServer.enabled}
                      type="checkbox"
                      onChange={(event) => updateSelectedServer({ enabled: event.target.checked })}
                    />
                    Sync this server to Open WebUI
                  </label>
                </Field>
                <Field label="Plugin Scope">
                  <textarea
                    className="admin-input min-h-24"
                    value={selectedServer.plugin_scope.join("\n")}
                    onChange={(event) => updateListField("plugin_scope", event.target.value)}
                  />
                </Field>
                <Field label="Disabled Tools">
                  <textarea
                    className="admin-input min-h-24"
                    value={selectedServer.disabled_tools.join("\n")}
                    onChange={(event) =>
                      updateSelectedServer({
                        disabled_tools: event.target.value
                          .split(/\r?\n/)
                          .map((item) => item.trim())
                          .filter(Boolean),
                      })
                    }
                  />
                </Field>
                <Field label="URL">
                  <input
                    className="admin-input"
                    value={selectedServer.url ?? ""}
                    onChange={(event) => updateSelectedServer({ url: event.target.value })}
                  />
                </Field>
                <Field label="Workdir">
                  <input
                    className="admin-input"
                    value={selectedServer.workdir ?? ""}
                    onChange={(event) => updateSelectedServer({ workdir: event.target.value })}
                  />
                </Field>
                <Field label="Command">
                  <textarea
                    className="admin-input min-h-24"
                    value={(selectedServer.command ?? []).join("\n")}
                    onChange={(event) => updateListField("command", event.target.value)}
                  />
                </Field>
                <Field label="Timeout (ms)">
                  <input
                    className="admin-input"
                    type="number"
                    value={selectedServer.timeout_ms}
                    onChange={(event) =>
                      updateSelectedServer({ timeout_ms: Number(event.target.value) || 30000 })
                    }
                  />
                </Field>
                <Field label="Auth Type">
                  <select
                    className="admin-input"
                    value={selectedServer.auth_type}
                    onChange={(event) =>
                      updateSelectedServer({
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
                  <label className="flex h-11 items-center gap-3 text-sm text-slate-700">
                    <input
                      checked={selectedServer.verify_tls}
                      type="checkbox"
                      onChange={(event) => updateSelectedServer({ verify_tls: event.target.checked })}
                    />
                    Reject invalid certificates
                  </label>
                </Field>
                <Field label="Auth Payload">
                  <textarea
                    className="admin-input min-h-24"
                    value={formatMap(selectedServer.auth_payload)}
                    onChange={(event) => updateMapField("auth_payload", event.target.value)}
                  />
                </Field>
                <Field label="Headers">
                  <textarea
                    className="admin-input min-h-24"
                    value={formatMap(selectedServer.headers)}
                    onChange={(event) => updateMapField("headers", event.target.value)}
                  />
                </Field>
                <Field label="Env">
                  <textarea
                    className="admin-input min-h-24"
                    value={formatMap(selectedServer.env)}
                    onChange={(event) => updateMapField("env", event.target.value)}
                  />
                </Field>
                <Field label="Description">
                  <textarea
                    className="admin-input min-h-24"
                    value={selectedServer.description ?? ""}
                    onChange={(event) => updateSelectedServer({ description: event.target.value })}
                  />
                </Field>
                <Field label="Notes">
                  <textarea
                    className="admin-input min-h-24"
                    value={selectedServer.notes ?? ""}
                    onChange={(event) => updateSelectedServer({ notes: event.target.value })}
                  />
                </Field>
              </div>
            ) : (
              <div className="mt-5 text-sm text-slate-500">Select or create an MCP server to edit it.</div>
            )}

            {validateResult ? (
              <div className="mt-5 rounded-3xl bg-slate-50 px-4 py-3 text-sm text-slate-700">
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
              Discover tools once, then use these toggles to write the server-level disabled list.
            </div>

            <div className="mt-4 grid gap-3 md:grid-cols-2">
              {(selectedState?.discovered_tools ?? []).map((tool) => {
                const checked = !selectedServer?.disabled_tools.includes(tool.name);
                return (
                  <label
                    key={tool.name}
                    className="admin-surface-muted flex items-start gap-3 rounded-3xl px-4 py-3 text-sm text-slate-700"
                  >
                    <input
                      checked={checked}
                      type="checkbox"
                      onChange={() => toggleDiscoveredTool(tool)}
                    />
                    <div>
                      <div className="font-medium text-slate-900">{tool.name}</div>
                      <div className="mt-1 text-xs leading-5 text-slate-500">
                        {tool.description || "No description from discovery."}
                      </div>
                    </div>
                  </label>
                );
              })}
            </div>

            {(selectedState?.discovered_tools ?? []).length === 0 ? (
              <div className="mt-4 text-sm text-slate-500">
                No discovered tools yet. Run “Discover Tools” first.
              </div>
            ) : null}
          </section>

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
            <pre className="mt-4 overflow-x-auto rounded-3xl bg-slate-50 p-4 text-xs leading-6 text-slate-700">
              {preview?.tool_server_connections_json ?? "[]"}
            </pre>
          </section>
        </div>
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

function formatMap(values: Record<string, string> | undefined) {
  if (!values) {
    return "";
  }
  return Object.entries(values)
    .map(([key, value]) => `${key}=${value}`)
    .join("\n");
}
