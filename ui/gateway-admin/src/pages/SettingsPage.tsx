import { useEffect, useMemo, useState } from "react";

import { PageHeader } from "../components/PageHeader";
import { apiGet, apiPost } from "../lib/api";
import type {
  ServicesResponse,
  SSHBindingsResponse,
  SSHHostProfile,
  SSHHostsResponse,
  SSHHostTestResponse,
  SSHUserBinding,
  TerminalPathsSettings,
} from "../lib/types";

type ActionState = "idle" | "pending";

function newHostDraft(): SSHHostProfile {
  return {
    id: `host-${Math.random().toString(36).slice(2, 8)}`,
    label: "New SSH Host",
    enabled: true,
    host: "",
    port: 22,
    username: "",
    auth_type: "password",
    password: "",
    private_key: "",
    passphrase: "",
    remote_shell_default: "bash",
    allowed_paths: ["/home/ctf"],
    default_workdir: "",
    host_key_status: "unknown",
    host_key_fingerprint: "",
  };
}

function newBindingDraft(): SSHUserBinding {
  return {
    user_email: "",
    default_host_id: "",
  };
}

export function SettingsPage() {
  const [services, setServices] = useState<ServicesResponse | null>(null);
  const [settings, setSettings] = useState<TerminalPathsSettings | null>(null);
  const [draft, setDraft] = useState("");
  const [sshConfigPath, setSshConfigPath] = useState("");
  const [sshBindingConfigPath, setSshBindingConfigPath] = useState("");
  const [sshHosts, setSshHosts] = useState<SSHHostProfile[]>([]);
  const [sshBindings, setSshBindings] = useState<SSHUserBinding[]>([]);
  const [selectedHostId, setSelectedHostId] = useState("");
  const [sshTestResult, setSshTestResult] = useState<SSHHostTestResponse | null>(null);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [pathActionState, setPathActionState] = useState<ActionState>("idle");
  const [sshHostActionState, setSshHostActionState] = useState<ActionState>("idle");
  const [sshBindingActionState, setSshBindingActionState] = useState<ActionState>("idle");

  async function load() {
    try {
      const [servicesResp, settingsResp, sshHostsResp, sshBindingsResp] = await Promise.all([
        apiGet<ServicesResponse>("/internal/admin/services"),
        apiGet<TerminalPathsSettings>("/internal/admin/settings/terminal-paths"),
        apiGet<SSHHostsResponse>("/internal/admin/ssh/hosts"),
        apiGet<SSHBindingsResponse>("/internal/admin/ssh/bindings"),
      ]);
      setServices(servicesResp);
      setSettings(settingsResp);
      setDraft(settingsResp.allowed_paths.join("\n"));
      setSshHosts(sshHostsResp.hosts);
      setSshBindings(sshBindingsResp.bindings);
      setSshConfigPath(sshHostsResp.config_path);
      setSshBindingConfigPath(sshBindingsResp.config_path);
      setSelectedHostId((current) => {
        if (current && sshHostsResp.hosts.some((host) => host.id === current)) {
          return current;
        }
        return sshHostsResp.hosts[0]?.id ?? "";
      });
      setError("");
    } catch (err) {
      setError((err as Error).message);
    }
  }

  useEffect(() => {
    load();
  }, []);

  const selectedHost = useMemo(
    () => sshHosts.find((host) => host.id === selectedHostId) ?? null,
    [selectedHostId, sshHosts],
  );

  const toolRouterStatus = useMemo(
    () => services?.services.find((item) => item.name === "tool-router") ?? null,
    [services],
  );

  const toolServerInfo = useMemo(() => {
    const rawAddress = toolRouterStatus?.address?.trim() || "http://127.0.0.1:8091";
    const baseUrl = rawAddress.replace(/\/healthz$/i, "");
    return {
      baseUrl,
      openapiUrl: `${baseUrl}/openapi.json`,
      terminalUrl: `${baseUrl}/api/tools/terminal`,
    };
  }, [toolRouterStatus]);

  function updateSelectedHost(mutator: (host: SSHHostProfile) => SSHHostProfile) {
    setSshHosts((current) =>
      current.map((host) => (host.id === selectedHostId ? mutator(host) : host)),
    );
  }

  function addSSHHost() {
    const host = newHostDraft();
    setSshHosts((current) => [...current, host]);
    setSelectedHostId(host.id);
    setSshTestResult(null);
  }

  function removeSelectedHost() {
    if (!selectedHostId) {
      return;
    }
    const nextHosts = sshHosts.filter((host) => host.id !== selectedHostId);
    setSshHosts(nextHosts);
    setSelectedHostId(nextHosts[0]?.id ?? "");
    setSshTestResult(null);
  }

  function addBinding() {
    setSshBindings((current) => [...current, newBindingDraft()]);
  }

  function updateBinding(index: number, patch: Partial<SSHUserBinding>) {
    setSshBindings((current) =>
      current.map((binding, currentIndex) =>
        currentIndex === index ? { ...binding, ...patch } : binding,
      ),
    );
  }

  function removeBinding(index: number) {
    setSshBindings((current) => current.filter((_, currentIndex) => currentIndex !== index));
  }

  async function savePaths() {
    setPathActionState("pending");
    setError("");
    setNotice("");
    try {
      const allowedPaths = draft
        .split(/\r?\n/)
        .map((item) => item.trim())
        .filter(Boolean);
      const response = await apiPost<TerminalPathsSettings>(
        "/internal/admin/settings/terminal-paths",
        {
          allowed_paths: allowedPaths,
          restart_now: false,
        },
      );
      setSettings(response);
      setDraft(response.allowed_paths.join("\n"));
      setNotice("Terminal allowed paths saved. Restart Tool Router if you want the new paths to take effect immediately.");
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setPathActionState("idle");
    }
  }

  async function saveSSHHosts() {
    setSshHostActionState("pending");
    setError("");
    setNotice("");
    try {
      const response = await apiPost<SSHHostsResponse>("/internal/admin/ssh/hosts", {
        hosts: sshHosts,
      });
      setSshHosts(response.hosts);
      setSelectedHostId((current) => {
        if (current && response.hosts.some((host) => host.id === current)) {
          return current;
        }
        return response.hosts[0]?.id ?? "";
      });
      setSshConfigPath(response.config_path);
      setNotice("SSH host profiles saved.");
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setSshHostActionState("idle");
    }
  }

  async function testSelectedSSHHost() {
    if (!selectedHost) {
      return;
    }
    setSshHostActionState("pending");
    setError("");
    setNotice("");
    setSshTestResult(null);
    try {
      const response = await apiPost<SSHHostTestResponse>("/internal/admin/ssh/hosts/test", {
        host: selectedHost,
        timeout_ms: 10000,
      });
      setSshTestResult(response);
      setNotice(response.summary);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setSshHostActionState("idle");
    }
  }

  async function confirmSelectedSSHHostKey() {
    if (!selectedHost || !sshTestResult?.host_key_fingerprint) {
      return;
    }
    setSshHostActionState("pending");
    setError("");
    setNotice("");
    try {
      const response = await apiPost<SSHHostsResponse>(
        "/internal/admin/ssh/hosts/confirm-host-key",
        {
          host_id: selectedHost.id,
          fingerprint: sshTestResult.host_key_fingerprint,
        },
      );
      setSshHosts(response.hosts);
      setSshConfigPath(response.config_path);
      setNotice("SSH host key fingerprint saved as trusted.");
      await testSelectedSSHHost();
    } catch (err) {
      setError((err as Error).message);
      setSshHostActionState("idle");
    }
  }

  async function saveSSHBindings() {
    setSshBindingActionState("pending");
    setError("");
    setNotice("");
    try {
      const response = await apiPost<SSHBindingsResponse>("/internal/admin/ssh/bindings", {
        bindings: sshBindings,
      });
      setSshBindings(response.bindings);
      setSshBindingConfigPath(response.config_path);
      setNotice("Default SSH host bindings saved.");
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setSshBindingActionState("idle");
    }
  }

  async function copyText(value: string, label: string) {
    try {
      await navigator.clipboard.writeText(value);
      setNotice(`${label} copied.`);
      setError("");
    } catch (err) {
      setError((err as Error).message);
    }
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="Settings"
        description="Manage terminal path boundaries, SSH host profiles, and per-user default remote targets."
        action={
          <div className="flex shrink-0 flex-nowrap items-center gap-2">
            <button className="admin-button" disabled={pathActionState === "pending"} onClick={savePaths}>
              Save Paths
            </button>
          </div>
        }
      />

      {error ? (
        <div className="admin-surface rounded-3xl bg-rose-50 p-4 text-sm text-rose-700">{error}</div>
      ) : null}
      {notice ? (
        <div className="admin-surface rounded-3xl bg-emerald-50 p-4 text-sm text-emerald-700">
          {notice}
        </div>
      ) : null}

      <section className="admin-surface rounded-3xl p-5">
        <div className="flex items-start justify-between gap-4">
          <div>
            <div className="text-sm font-semibold text-slate-900">Terminal Path Allowlist</div>
            <div className="mt-1 text-xs leading-6 text-slate-500">
              One absolute path per line. Local terminal workdir must stay inside these roots.
            </div>
          </div>
          <span className={`admin-badge ${toolRouterStatus?.status === "ok" ? "success" : "danger"}`}>
            {toolRouterStatus?.status === "ok" ? "Tool Router Online" : "Tool Router Offline"}
          </span>
        </div>

        <div className="mt-4 space-y-3">
          <div className="admin-field">
            <label className="admin-field-label" htmlFor="terminal-paths">
              Allowed Paths
            </label>
            <textarea
              id="terminal-paths"
              className="admin-input min-h-56 resize-y"
              value={draft}
              onChange={(event) => setDraft(event.target.value)}
              placeholder={"D:\\AI\\Local\nD:\\Environment2"}
              spellCheck={false}
            />
          </div>

          <div className="admin-surface-muted rounded-3xl p-4 text-xs leading-6 text-slate-600">
            <div>Config: {settings?.config_path ?? "-"}</div>
            <div>Restart required: {settings?.restart_required ? "yes" : "no"}</div>
          </div>
        </div>
      </section>

      <section className="admin-surface rounded-3xl p-5">
        <div className="flex items-start justify-between gap-4">
          <div>
            <div className="text-sm font-semibold text-slate-900">Open WebUI Tool Server</div>
            <div className="mt-1 text-xs leading-6 text-slate-500">
              Open WebUI should continue using this OpenAPI terminal server. The same endpoint now supports both local and SSH execution.
            </div>
          </div>
          <span className={`admin-badge ${toolRouterStatus?.status === "ok" ? "success" : "danger"}`}>
            {toolRouterStatus?.status === "ok" ? "OpenAPI Ready" : "Tool Router Offline"}
          </span>
        </div>

        <div className="mt-4 grid gap-3 xl:grid-cols-2">
          <div className="admin-surface-muted rounded-3xl p-4">
            <div className="text-xs font-semibold uppercase tracking-[0.24em] text-slate-400">
              Tool Router
            </div>
            <div className="mt-2 font-mono text-sm text-slate-700">{toolServerInfo.baseUrl}</div>
            <button className="admin-button mt-3" type="button" onClick={() => copyText(toolServerInfo.baseUrl, "Tool Router URL")}>
              Copy Address
            </button>
          </div>

          <div className="admin-surface-muted rounded-3xl p-4">
            <div className="text-xs font-semibold uppercase tracking-[0.24em] text-slate-400">
              OpenAPI Spec
            </div>
            <div className="mt-2 font-mono text-sm text-slate-700">{toolServerInfo.openapiUrl}</div>
            <button className="admin-button mt-3" type="button" onClick={() => copyText(toolServerInfo.openapiUrl, "OpenAPI URL")}>
              Copy OpenAPI
            </button>
          </div>

          <div className="admin-surface-muted rounded-3xl p-4 xl:col-span-2">
            <div className="text-xs font-semibold uppercase tracking-[0.24em] text-slate-400">
              Terminal Endpoint
            </div>
            <div className="mt-2 font-mono text-sm text-slate-700">{toolServerInfo.terminalUrl}</div>
            <div className="mt-3 text-xs leading-6 text-slate-500">
              `POST /api/tools/terminal` accepts local terminal fields and SSH terminal fields on the same endpoint.
            </div>
          </div>
        </div>
      </section>

      <section className="admin-surface rounded-3xl p-5">
        <div className="flex items-start justify-between gap-4">
          <div>
            <div className="text-sm font-semibold text-slate-900">SSH Hosts</div>
            <div className="mt-1 text-xs leading-6 text-slate-500">
              Configure reusable remote execution targets. Secrets are not shown again after save; leaving them blank keeps the saved value.
            </div>
          </div>
          <div className="flex gap-2">
            <button className="admin-button" type="button" onClick={addSSHHost}>
              Add Host
            </button>
            <button className="admin-button" type="button" disabled={!selectedHostId} onClick={removeSelectedHost}>
              Remove
            </button>
            <button
              className="admin-button"
              type="button"
              disabled={sshHostActionState === "pending"}
              onClick={saveSSHHosts}
            >
              Save Hosts
            </button>
          </div>
        </div>

        <div className="mt-4 grid gap-4 xl:grid-cols-[280px_minmax(0,1fr)]">
          <div className="space-y-3">
            {sshHosts.length === 0 ? (
              <div className="admin-surface-muted rounded-3xl p-4 text-sm text-slate-500">
                No SSH hosts yet.
              </div>
            ) : (
              sshHosts.map((host) => (
                <button
                  key={host.id}
                  type="button"
                  onClick={() => {
                    setSelectedHostId(host.id);
                    setSshTestResult(null);
                  }}
                  className={`admin-surface-muted w-full rounded-3xl p-4 text-left transition ${
                    selectedHostId === host.id ? "shadow-[0_18px_40px_rgba(15,23,42,0.14)]" : ""
                  }`}
                >
                  <div className="flex items-center justify-between gap-3">
                    <div className="text-sm font-semibold text-slate-900">{host.label}</div>
                    <span className={`admin-badge ${host.enabled ? "success" : "danger"}`}>
                      {host.enabled ? "Enabled" : "Disabled"}
                    </span>
                  </div>
                  <div className="mt-2 text-xs leading-6 text-slate-500">
                    {host.username}@{host.host}:{host.port}
                  </div>
                  <div className="mt-1 text-xs leading-6 text-slate-500">
                    {host.id} · {host.remote_shell_default}
                  </div>
                </button>
              ))
            )}
          </div>

          <div className="space-y-4">
            {selectedHost ? (
              <>
                <div className="grid gap-3 md:grid-cols-2">
                  <div className="admin-field">
                    <label className="admin-field-label">Host ID</label>
                    <input
                      className="admin-input"
                      value={selectedHost.id}
                      onChange={(event) => {
                        const nextId = event.target.value;
                        setSelectedHostId(nextId);
                        updateSelectedHost((host) => ({ ...host, id: nextId }));
                      }}
                    />
                  </div>
                  <div className="admin-field">
                    <label className="admin-field-label">Label</label>
                    <input
                      className="admin-input"
                      value={selectedHost.label}
                      onChange={(event) =>
                        updateSelectedHost((host) => ({ ...host, label: event.target.value }))
                      }
                    />
                  </div>
                  <div className="admin-field">
                    <label className="admin-field-label">Host</label>
                    <input
                      className="admin-input"
                      value={selectedHost.host}
                      onChange={(event) =>
                        updateSelectedHost((host) => ({ ...host, host: event.target.value }))
                      }
                    />
                  </div>
                  <div className="admin-field">
                    <label className="admin-field-label">Port</label>
                    <input
                      className="admin-input"
                      type="number"
                      value={selectedHost.port}
                      onChange={(event) =>
                        updateSelectedHost((host) => ({
                          ...host,
                          port: Number(event.target.value || 22),
                        }))
                      }
                    />
                  </div>
                  <div className="admin-field">
                    <label className="admin-field-label">Username</label>
                    <input
                      className="admin-input"
                      value={selectedHost.username}
                      onChange={(event) =>
                        updateSelectedHost((host) => ({ ...host, username: event.target.value }))
                      }
                    />
                  </div>
                  <div className="admin-field">
                    <label className="admin-field-label">Remote Shell</label>
                    <select
                      className="admin-input"
                      value={selectedHost.remote_shell_default}
                      onChange={(event) =>
                        updateSelectedHost((host) => ({
                          ...host,
                          remote_shell_default: event.target.value as "bash" | "powershell",
                        }))
                      }
                    >
                      <option value="bash">bash</option>
                      <option value="powershell">powershell</option>
                    </select>
                  </div>
                  <div className="admin-field">
                    <label className="admin-field-label">Auth Type</label>
                    <select
                      className="admin-input"
                      value={selectedHost.auth_type}
                      onChange={(event) =>
                        updateSelectedHost((host) => ({
                          ...host,
                          auth_type: event.target.value as "password" | "private_key",
                        }))
                      }
                    >
                      <option value="password">password</option>
                      <option value="private_key">private_key</option>
                    </select>
                  </div>
                  <div className="admin-field">
                    <label className="admin-field-label">Enabled</label>
                    <select
                      className="admin-input"
                      value={selectedHost.enabled ? "true" : "false"}
                      onChange={(event) =>
                        updateSelectedHost((host) => ({
                          ...host,
                          enabled: event.target.value === "true",
                        }))
                      }
                    >
                      <option value="true">Enabled</option>
                      <option value="false">Disabled</option>
                    </select>
                  </div>
                </div>

                {selectedHost.auth_type === "password" ? (
                  <div className="admin-field">
                    <label className="admin-field-label">
                      Password {selectedHost.has_password ? "(leave blank to keep saved value)" : ""}
                    </label>
                    <input
                      className="admin-input"
                      type="password"
                      value={selectedHost.password ?? ""}
                      onChange={(event) =>
                        updateSelectedHost((host) => ({ ...host, password: event.target.value }))
                      }
                    />
                  </div>
                ) : (
                  <>
                    <div className="admin-field">
                      <label className="admin-field-label">
                        Private Key {selectedHost.has_private_key ? "(leave blank to keep saved value)" : ""}
                      </label>
                      <textarea
                        className="admin-input min-h-40 resize-y font-mono text-xs"
                        value={selectedHost.private_key ?? ""}
                        onChange={(event) =>
                          updateSelectedHost((host) => ({ ...host, private_key: event.target.value }))
                        }
                        spellCheck={false}
                      />
                    </div>
                    <div className="admin-field">
                      <label className="admin-field-label">Passphrase</label>
                      <input
                        className="admin-input"
                        type="password"
                        value={selectedHost.passphrase ?? ""}
                        onChange={(event) =>
                          updateSelectedHost((host) => ({ ...host, passphrase: event.target.value }))
                        }
                      />
                    </div>
                  </>
                )}

                <div className="grid gap-3 md:grid-cols-2">
                  <div className="admin-field">
                    <label className="admin-field-label">Allowed Paths</label>
                    <textarea
                      className="admin-input min-h-32 resize-y"
                      value={selectedHost.allowed_paths.join("\n")}
                      onChange={(event) =>
                        updateSelectedHost((host) => ({
                          ...host,
                          allowed_paths: event.target.value
                            .split(/\r?\n/)
                            .map((item) => item.trim())
                            .filter(Boolean),
                        }))
                      }
                      spellCheck={false}
                    />
                  </div>
                  <div className="admin-field">
                    <label className="admin-field-label">Default Workdir</label>
                    <input
                      className="admin-input"
                      value={selectedHost.default_workdir ?? ""}
                      onChange={(event) =>
                        updateSelectedHost((host) => ({
                          ...host,
                          default_workdir: event.target.value,
                        }))
                      }
                    />
                  </div>
                </div>

                <div className="admin-surface-muted rounded-3xl p-4 text-xs leading-6 text-slate-600">
                  <div>Host key status: {selectedHost.host_key_status || "unknown"}</div>
                  <div>Host key fingerprint: {selectedHost.host_key_fingerprint || "-"}</div>
                  <div>Config: {sshConfigPath || "-"}</div>
                </div>

                <div className="flex flex-wrap gap-2">
                  <button
                    className="admin-button"
                    type="button"
                    disabled={sshHostActionState === "pending"}
                    onClick={testSelectedSSHHost}
                  >
                    Test Connection
                  </button>
                  <button
                    className="admin-button"
                    type="button"
                    disabled={
                      sshHostActionState === "pending" ||
                      sshTestResult?.host_key_status !== "unknown" ||
                      !sshTestResult?.host_key_fingerprint
                    }
                    onClick={confirmSelectedSSHHostKey}
                  >
                    Confirm Host Key
                  </button>
                </div>

                {sshTestResult ? (
                  <div className="admin-surface-muted rounded-3xl p-4 text-sm text-slate-700">
                    <div className="font-semibold text-slate-900">{sshTestResult.summary}</div>
                    <div className="mt-2 text-xs leading-6 text-slate-500">
                      Status: {sshTestResult.host_key_status || "-"}
                    </div>
                    <div className="text-xs leading-6 text-slate-500">
                      Fingerprint: {sshTestResult.host_key_fingerprint || "-"}
                    </div>
                    {sshTestResult.error ? (
                      <div className="mt-2 text-xs leading-6 text-rose-600">
                        {sshTestResult.error.code}: {sshTestResult.error.message}
                      </div>
                    ) : null}
                  </div>
                ) : null}
              </>
            ) : (
              <div className="admin-surface-muted rounded-3xl p-6 text-sm text-slate-500">
                Select an SSH host on the left, or create a new one.
              </div>
            )}
          </div>
        </div>
      </section>

      <section className="admin-surface rounded-3xl p-5">
        <div className="flex items-start justify-between gap-4">
          <div>
            <div className="text-sm font-semibold text-slate-900">User Default SSH Host</div>
            <div className="mt-1 text-xs leading-6 text-slate-500">
              Bind each Open WebUI account email to its default SSH target. The WebUI patch will inject the mapped host automatically.
            </div>
          </div>
          <div className="flex gap-2">
            <button className="admin-button" type="button" onClick={addBinding}>
              Add Binding
            </button>
            <button
              className="admin-button"
              type="button"
              disabled={sshBindingActionState === "pending"}
              onClick={saveSSHBindings}
            >
              Save Bindings
            </button>
          </div>
        </div>

        <div className="mt-4 space-y-3">
          {sshBindings.length === 0 ? (
            <div className="admin-surface-muted rounded-3xl p-4 text-sm text-slate-500">
              No user bindings yet.
            </div>
          ) : null}

          {sshBindings.map((binding, index) => (
            <div key={`${binding.user_email}-${index}`} className="admin-surface-muted rounded-3xl p-4">
              <div className="grid gap-3 lg:grid-cols-[minmax(0,1fr)_240px_auto]">
                <div className="admin-field">
                  <label className="admin-field-label">User Email</label>
                  <input
                    className="admin-input"
                    value={binding.user_email}
                    onChange={(event) => updateBinding(index, { user_email: event.target.value })}
                  />
                </div>
                <div className="admin-field">
                  <label className="admin-field-label">Default Host</label>
                  <select
                    className="admin-input"
                    value={binding.default_host_id}
                    onChange={(event) =>
                      updateBinding(index, { default_host_id: event.target.value })
                    }
                  >
                    <option value="">Select host</option>
                    {sshHosts.map((host) => (
                      <option key={host.id} value={host.id}>
                        {host.label} ({host.id})
                      </option>
                    ))}
                  </select>
                </div>
                <div className="flex items-end">
                  <button className="admin-button" type="button" onClick={() => removeBinding(index)}>
                    Remove
                  </button>
                </div>
              </div>
            </div>
          ))}

          <div className="admin-surface-muted rounded-3xl p-4 text-xs leading-6 text-slate-600">
            <div>Config: {sshBindingConfigPath || "-"}</div>
            <div>Each Open WebUI user should have its own account email for this mapping to stay stable.</div>
          </div>
        </div>
      </section>

      <section className="admin-surface rounded-3xl p-5">
        <div className="text-sm font-semibold text-slate-900">Service Status Summary</div>
        <div className="mt-4 grid gap-3 md:grid-cols-2 xl:grid-cols-3">
          {(services?.services ?? []).map((service) => (
            <div key={service.name} className="admin-surface-muted rounded-3xl p-4">
              <div className="flex items-center justify-between gap-3">
                <div className="text-sm font-semibold text-slate-900">{service.name}</div>
                <span className={`admin-badge ${service.status === "ok" ? "success" : "danger"}`}>
                  {service.status}
                </span>
              </div>
              <div className="mt-2 text-xs leading-6 text-slate-500">{service.address || "-"}</div>
              <div className="mt-1 text-xs leading-6 text-slate-500">{service.message || "-"}</div>
            </div>
          ))}
        </div>
      </section>
    </div>
  );
}
