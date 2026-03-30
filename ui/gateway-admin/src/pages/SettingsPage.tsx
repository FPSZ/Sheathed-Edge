import { useEffect, useMemo, useState } from "react";

import { useAdminScope } from "../app/AdminScopeContext";
import { PageHeader } from "../components/PageHeader";
import { apiGet, apiPost } from "../lib/api";
import type {
  ServicesResponse,
  SSHHostProfile,
  SSHHostsResponse,
  SSHHostTestResponse,
  TerminalPathsSettings,
  UserWorkspaceResponse,
} from "../lib/types";

type ActionState = "idle" | "pending";

function newHostDraft(): SSHHostProfile {
  return {
    id: `host-${Math.random().toString(36).slice(2, 8)}`,
    label: "新 SSH 主机",
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

export function SettingsPage() {
  const { selectedUserEmail, refreshUsers } = useAdminScope();
  const [services, setServices] = useState<ServicesResponse | null>(null);
  const [globalPaths, setGlobalPaths] = useState<TerminalPathsSettings | null>(null);
  const [globalPathDraft, setGlobalPathDraft] = useState("");
  const [workspace, setWorkspace] = useState<UserWorkspaceResponse | null>(null);
  const [workspacePathDraft, setWorkspacePathDraft] = useState("");
  const [sshHosts, setSshHosts] = useState<SSHHostProfile[]>([]);
  const [sshConfigPath, setSshConfigPath] = useState("");
  const [selectedHostId, setSelectedHostId] = useState("");
  const [sshTestResult, setSshTestResult] = useState<SSHHostTestResponse | null>(null);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [workspaceActionState, setWorkspaceActionState] = useState<ActionState>("idle");
  const [globalPathActionState, setGlobalPathActionState] = useState<ActionState>("idle");
  const [sshHostActionState, setSshHostActionState] = useState<ActionState>("idle");

  async function loadBase() {
    try {
      const [servicesResp, globalPathsResp, sshHostsResp] = await Promise.all([
        apiGet<ServicesResponse>("/internal/admin/services"),
        apiGet<TerminalPathsSettings>("/internal/admin/settings/terminal-paths"),
        apiGet<SSHHostsResponse>("/internal/admin/ssh/hosts"),
      ]);
      setServices(servicesResp);
      setGlobalPaths(globalPathsResp);
      setGlobalPathDraft(globalPathsResp.allowed_paths.join("\n"));
      setSshHosts(sshHostsResp.hosts);
      setSshConfigPath(sshHostsResp.config_path);
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

  async function loadWorkspace() {
    if (!selectedUserEmail) {
      setWorkspace(null);
      setWorkspacePathDraft("");
      return;
    }
    try {
      const response = await apiGet<UserWorkspaceResponse>(
        `/internal/admin/users/workspace?user_email=${encodeURIComponent(selectedUserEmail)}`,
      );
      setWorkspace(response);
      setWorkspacePathDraft(response.workspace.terminal_allowed_paths.join("\n"));
    } catch (err) {
      setError((err as Error).message);
    }
  }

  useEffect(() => {
    loadBase();
  }, []);

  useEffect(() => {
    loadWorkspace();
  }, [selectedUserEmail]);

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

  async function saveWorkspace() {
    if (!selectedUserEmail || !workspace) {
      return;
    }
    setWorkspaceActionState("pending");
    setError("");
    setNotice("");
    try {
      const response = await apiPost<UserWorkspaceResponse>("/internal/admin/users/workspace", {
        workspace: {
          ...workspace.workspace,
          user_email: selectedUserEmail,
          terminal_allowed_paths: workspacePathDraft
            .split(/\r?\n/)
            .map((item: string) => item.trim())
            .filter(Boolean),
        },
      });
      setWorkspace(response);
      setWorkspacePathDraft(response.workspace.terminal_allowed_paths.join("\n"));
      setNotice("用户工作区已保存。");
      await refreshUsers();
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setWorkspaceActionState("idle");
    }
  }

  async function saveGlobalPaths() {
    setGlobalPathActionState("pending");
    setError("");
    setNotice("");
    try {
      const allowedPaths = globalPathDraft
        .split(/\r?\n/)
        .map((item) => item.trim())
        .filter(Boolean);
      const response = await apiPost<TerminalPathsSettings>("/internal/admin/settings/terminal-paths", {
        allowed_paths: allowedPaths,
        restart_now: false,
      });
      setGlobalPaths(response);
      setGlobalPathDraft(response.allowed_paths.join("\n"));
      setNotice("系统路径边界已保存。");
      await loadWorkspace();
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setGlobalPathActionState("idle");
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
      setSshConfigPath(response.config_path);
      setSelectedHostId((current) => {
        if (current && response.hosts.some((host) => host.id === current)) {
          return current;
        }
        return response.hosts[0]?.id ?? "";
      });
      setNotice("全局 SSH 主机已保存。");
      await loadWorkspace();
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
      const response = await apiPost<SSHHostsResponse>("/internal/admin/ssh/hosts/confirm-host-key", {
        host_id: selectedHost.id,
        fingerprint: sshTestResult.host_key_fingerprint,
      });
      setSshHosts(response.hosts);
      setSshConfigPath(response.config_path);
      setNotice("SSH host key 已确认。");
      await testSelectedSSHHost();
    } catch (err) {
      setError((err as Error).message);
      setSshHostActionState("idle");
    }
  }

  async function copyText(value: string, label: string) {
    try {
      await navigator.clipboard.writeText(value);
      setNotice(`${label} 已复制。`);
      setError("");
    } catch (err) {
      setError((err as Error).message);
    }
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="设置"
        description="当前页分成两层：上面是用户工作区，下面是系统边界和全局 SSH 资源。"
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
            <div className="text-sm font-semibold text-slate-900">当前用户工作区</div>
            <div className="mt-1 text-xs leading-6 text-slate-500">
              作用于当前选中的 Open WebUI 用户。包含本地目录边界、默认工作目录、默认 SSH 目标。
            </div>
          </div>
          <button
            className="admin-button"
            type="button"
            disabled={!selectedUserEmail || workspaceActionState === "pending"}
            onClick={saveWorkspace}
          >
            保存用户工作区
          </button>
        </div>

        {!selectedUserEmail ? (
          <div className="mt-4 admin-surface-muted rounded-3xl p-5 text-sm text-slate-500">
            先在顶部选择一个具体用户，再编辑该用户的工作区。
          </div>
        ) : workspace ? (
          <div className="mt-4 grid gap-4 xl:grid-cols-[1.2fr_0.8fr]">
            <div className="space-y-4">
              <div className="admin-field">
                <label className="admin-field-label">用户邮箱</label>
                <input className="admin-input" value={workspace.workspace.user_email} disabled />
              </div>
              <div className="admin-field">
                <label className="admin-field-label">显示名称</label>
                <input
                  className="admin-input"
                  value={workspace.workspace.label}
                  onChange={(event) =>
                    setWorkspace((current) =>
                      current
                        ? {
                            ...current,
                            workspace: { ...current.workspace, label: event.target.value },
                          }
                        : current,
                    )
                  }
                />
              </div>
              <div className="admin-field">
                <label className="admin-field-label">可访问目录</label>
                <textarea
                  className="admin-input min-h-40 resize-y"
                  value={workspacePathDraft}
                  onChange={(event) => setWorkspacePathDraft(event.target.value)}
                  spellCheck={false}
                />
              </div>
            </div>

            <div className="space-y-4">
              <div className="admin-field">
                <label className="admin-field-label">默认本地工作目录</label>
                <input
                  className="admin-input"
                  value={workspace.workspace.default_local_workdir ?? ""}
                  onChange={(event) =>
                    setWorkspace((current) =>
                      current
                        ? {
                            ...current,
                            workspace: {
                              ...current.workspace,
                              default_local_workdir: event.target.value,
                            },
                          }
                        : current,
                    )
                  }
                />
              </div>
              <div className="admin-field">
                <label className="admin-field-label">默认 SSH 主机</label>
                <select
                  className="admin-input"
                  value={workspace.workspace.default_ssh_host_id ?? ""}
                  onChange={(event) =>
                    setWorkspace((current) =>
                      current
                        ? {
                            ...current,
                            workspace: {
                              ...current.workspace,
                              default_ssh_host_id: event.target.value,
                            },
                          }
                        : current,
                    )
                  }
                >
                  <option value="">不指定</option>
                  {sshHosts.map((host) => (
                    <option key={host.id} value={host.id}>
                      {host.label} ({host.id})
                    </option>
                  ))}
                </select>
              </div>
              <div className="admin-surface-muted rounded-3xl p-4 text-xs leading-6 text-slate-600">
                <div>配置文件：{workspace.config_path || "-"}</div>
                <div>旧绑定兼容文件：{workspace.legacy_bindings_path || "-"}</div>
                <div>系统边界变更后需要重启 Tool Router：{workspace.restart_required ? "是" : "否"}</div>
              </div>
            </div>
          </div>
        ) : null}
      </section>

      <section className="admin-surface rounded-3xl p-5">
        <div className="flex items-start justify-between gap-4">
          <div>
            <div className="text-sm font-semibold text-slate-900">系统边界与 Tool Server</div>
            <div className="mt-1 text-xs leading-6 text-slate-500">
              这是全局运行配置。用户工作区只能在这些边界内收窄，不能放宽。
            </div>
          </div>
          <button
            className="admin-button"
            disabled={globalPathActionState === "pending"}
            onClick={saveGlobalPaths}
          >
            保存系统边界
          </button>
        </div>

        <div className="mt-4 grid gap-4 xl:grid-cols-[1fr_1fr]">
          <div className="space-y-4">
            <div className="admin-field">
              <label className="admin-field-label">系统 allowed_paths</label>
              <textarea
                className="admin-input min-h-48 resize-y"
                value={globalPathDraft}
                onChange={(event) => setGlobalPathDraft(event.target.value)}
                placeholder={"D:\\AI\\Local\nD:\\Environment2"}
                spellCheck={false}
              />
            </div>
            <div className="admin-surface-muted rounded-3xl p-4 text-xs leading-6 text-slate-600">
              <div>配置文件：{globalPaths?.config_path ?? "-"}</div>
              <div>重启 Tool Router：{globalPaths?.restart_required ? "需要" : "不需要"}</div>
            </div>
          </div>

          <div className="space-y-3">
            <div className="admin-surface-muted rounded-3xl p-4">
              <div className="text-xs font-semibold uppercase tracking-[0.24em] text-slate-400">
                Tool Router
              </div>
              <div className="mt-2 font-mono text-sm text-slate-700">{toolServerInfo.baseUrl}</div>
              <button className="admin-button mt-3" type="button" onClick={() => copyText(toolServerInfo.baseUrl, "Tool Router 地址")}>
                复制地址
              </button>
            </div>
            <div className="admin-surface-muted rounded-3xl p-4">
              <div className="text-xs font-semibold uppercase tracking-[0.24em] text-slate-400">
                OpenAPI Spec
              </div>
              <div className="mt-2 font-mono text-sm text-slate-700">{toolServerInfo.openapiUrl}</div>
              <button className="admin-button mt-3" type="button" onClick={() => copyText(toolServerInfo.openapiUrl, "OpenAPI 地址")}>
                复制 OpenAPI
              </button>
            </div>
            <div className="admin-surface-muted rounded-3xl p-4 text-xs leading-6 text-slate-600">
              <div>Terminal Endpoint：{toolServerInfo.terminalUrl}</div>
              <div>状态：{toolRouterStatus?.status === "ok" ? "在线" : "离线"}</div>
            </div>
          </div>
        </div>
      </section>

      <section className="admin-surface rounded-3xl p-5">
        <div className="flex items-start justify-between gap-4">
          <div>
            <div className="text-sm font-semibold text-slate-900">全局 SSH 主机</div>
            <div className="mt-1 text-xs leading-6 text-slate-500">
              这里只维护系统级 SSH 主机资料。用户只在上面的工作区里选择默认主机。
            </div>
          </div>
          <div className="flex gap-2">
            <button className="admin-button" type="button" onClick={addSSHHost}>
              新建主机
            </button>
            <button className="admin-button" type="button" disabled={!selectedHostId} onClick={removeSelectedHost}>
              删除主机
            </button>
            <button
              className="admin-button"
              type="button"
              disabled={sshHostActionState === "pending"}
              onClick={saveSSHHosts}
            >
              保存主机
            </button>
          </div>
        </div>

        <div className="mt-4 grid gap-4 xl:grid-cols-[280px_minmax(0,1fr)]">
          <div className="space-y-3">
            {sshHosts.length === 0 ? (
              <div className="admin-surface-muted rounded-3xl p-4 text-sm text-slate-500">
                还没有 SSH 主机。
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
                      {host.enabled ? "启用" : "停用"}
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
                  <Field label="Host ID">
                    <input
                      className="admin-input"
                      value={selectedHost.id}
                      onChange={(event) => {
                        const nextId = event.target.value;
                        setSelectedHostId(nextId);
                        updateSelectedHost((host) => ({ ...host, id: nextId }));
                      }}
                    />
                  </Field>
                  <Field label="标签">
                    <input
                      className="admin-input"
                      value={selectedHost.label}
                      onChange={(event) =>
                        updateSelectedHost((host) => ({ ...host, label: event.target.value }))
                      }
                    />
                  </Field>
                  <Field label="Host">
                    <input
                      className="admin-input"
                      value={selectedHost.host}
                      onChange={(event) =>
                        updateSelectedHost((host) => ({ ...host, host: event.target.value }))
                      }
                    />
                  </Field>
                  <Field label="Port">
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
                  </Field>
                  <Field label="Username">
                    <input
                      className="admin-input"
                      value={selectedHost.username}
                      onChange={(event) =>
                        updateSelectedHost((host) => ({ ...host, username: event.target.value }))
                      }
                    />
                  </Field>
                  <Field label="Remote Shell">
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
                  </Field>
                  <Field label="认证方式">
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
                  </Field>
                  <Field label="状态">
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
                      <option value="true">启用</option>
                      <option value="false">停用</option>
                    </select>
                  </Field>
                </div>

                {selectedHost.auth_type === "password" ? (
                  <Field label={`密码${selectedHost.has_password ? "（留空保留原值）" : ""}`}>
                    <input
                      className="admin-input"
                      type="password"
                      value={selectedHost.password ?? ""}
                      onChange={(event) =>
                        updateSelectedHost((host) => ({ ...host, password: event.target.value }))
                      }
                    />
                  </Field>
                ) : (
                  <>
                    <Field label={`私钥${selectedHost.has_private_key ? "（留空保留原值）" : ""}`}>
                      <textarea
                        className="admin-input min-h-40 resize-y font-mono text-xs"
                        value={selectedHost.private_key ?? ""}
                        onChange={(event) =>
                          updateSelectedHost((host) => ({ ...host, private_key: event.target.value }))
                        }
                        spellCheck={false}
                      />
                    </Field>
                    <Field label="Passphrase">
                      <input
                        className="admin-input"
                        type="password"
                        value={selectedHost.passphrase ?? ""}
                        onChange={(event) =>
                          updateSelectedHost((host) => ({ ...host, passphrase: event.target.value }))
                        }
                      />
                    </Field>
                  </>
                )}

                <div className="grid gap-3 md:grid-cols-2">
                  <Field label="Allowed Paths">
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
                  </Field>
                  <Field label="Default Workdir">
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
                  </Field>
                </div>

                <div className="admin-surface-muted rounded-3xl p-4 text-xs leading-6 text-slate-600">
                  <div>Host key status: {selectedHost.host_key_status || "unknown"}</div>
                  <div>Host key fingerprint: {selectedHost.host_key_fingerprint || "-"}</div>
                  <div>配置文件：{sshConfigPath || "-"}</div>
                </div>

                <div className="flex flex-wrap gap-2">
                  <button
                    className="admin-button"
                    type="button"
                    disabled={sshHostActionState === "pending"}
                    onClick={testSelectedSSHHost}
                  >
                    测试连接
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
                    确认 Host Key
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
                选择左侧主机，或者新建一台。
              </div>
            )}
          </div>
        </div>
      </section>
    </div>
  );
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="admin-field">
      <label className="admin-field-label">{label}</label>
      {children}
    </div>
  );
}
