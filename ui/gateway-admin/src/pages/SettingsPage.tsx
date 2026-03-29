import { useEffect, useMemo, useState } from "react";

import { PageHeader } from "../components/PageHeader";
import { apiGet, apiPost } from "../lib/api";
import type { ServicesResponse, TerminalPathsSettings } from "../lib/types";

type ActionState = "idle" | "pending";

export function SettingsPage() {
  const [services, setServices] = useState<ServicesResponse | null>(null);
  const [settings, setSettings] = useState<TerminalPathsSettings | null>(null);
  const [draft, setDraft] = useState("");
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [actionState, setActionState] = useState<ActionState>("idle");

  async function load() {
    try {
      const [servicesResp, settingsResp] = await Promise.all([
        apiGet<ServicesResponse>("/internal/admin/services"),
        apiGet<TerminalPathsSettings>("/internal/admin/settings/terminal-paths"),
      ]);
      setServices(servicesResp);
      setSettings(settingsResp);
      setDraft(settingsResp.allowed_paths.join("\n"));
      setError("");
    } catch (err) {
      setError((err as Error).message);
    }
  }

  useEffect(() => {
    load();
  }, []);

  async function save() {
    setActionState("pending");
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
      setNotice("路径已保存。要立即生效，请在服务控制里重启 Tool Router。");
      const servicesResp = await apiGet<ServicesResponse>("/internal/admin/services");
      setServices(servicesResp);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setActionState("idle");
    }
  }

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
        title="系统设置"
        description="在这里配置 terminal 可访问的工作区和环境区。不同机器可保存不同路径。"
        action={
          <div className="flex shrink-0 flex-nowrap items-center gap-2">
            <button
              className="admin-button"
              disabled={actionState === "pending"}
              onClick={() => save()}
            >
              保存路径配置
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

      <section className="admin-surface rounded-3xl p-5">
        <div className="flex items-start justify-between gap-4">
          <div>
            <div className="text-sm font-semibold text-slate-900">Terminal 路径白名单</div>
            <div className="mt-1 text-xs leading-6 text-slate-500">
              每行一个绝对路径。AI 只能把 terminal 的 `workdir` 设在这些路径下面。
            </div>
          </div>
          <span className={`admin-badge ${toolRouterStatus?.status === "ok" ? "success" : "danger"}`}>
            {toolRouterStatus?.status === "ok" ? "Tool Router 在线" : "Tool Router 离线"}
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
            <div>配置文件：{settings?.config_path ?? "-"}</div>
            <div>说明：保存后会写回 Tool Router 配置；如果想立刻生效，请到服务控制里重启 Tool Router。</div>
          </div>
        </div>
      </section>

      <section className="admin-surface rounded-3xl p-5">
        <div className="flex items-start justify-between gap-4">
          <div>
            <div className="text-sm font-semibold text-slate-900">Open WebUI Tool Server</div>
            <div className="mt-1 text-xs leading-6 text-slate-500">
              在 Open WebUI 的 {"`Admin Settings -> External Tools`"} 中新增一个 OpenAPI 工具服务器，
              使用下面的 OpenAPI 地址。这里仅做展示与复制，不直接改 Open WebUI 配置。
            </div>
          </div>
          <span className={`admin-badge ${toolRouterStatus?.status === "ok" ? "success" : "danger"}`}>
            {toolRouterStatus?.status === "ok" ? "OpenAPI 可接入" : "Tool Router 离线"}
          </span>
        </div>

        <div className="mt-4 grid gap-3 xl:grid-cols-2">
          <div className="admin-surface-muted rounded-3xl p-4">
            <div className="text-xs font-semibold uppercase tracking-[0.24em] text-slate-400">
              Tool Router
            </div>
            <div className="mt-2 font-mono text-sm text-slate-700">{toolServerInfo.baseUrl}</div>
            <button
              className="admin-button mt-3"
              type="button"
              onClick={() => copyText(toolServerInfo.baseUrl, "Tool Router 地址")}
            >
              复制地址
            </button>
          </div>

          <div className="admin-surface-muted rounded-3xl p-4">
            <div className="text-xs font-semibold uppercase tracking-[0.24em] text-slate-400">
              OpenAPI Spec
            </div>
            <div className="mt-2 font-mono text-sm text-slate-700">{toolServerInfo.openapiUrl}</div>
            <button
              className="admin-button mt-3"
              type="button"
              onClick={() => copyText(toolServerInfo.openapiUrl, "OpenAPI 地址")}
            >
              复制 OpenAPI
            </button>
          </div>

          <div className="admin-surface-muted rounded-3xl p-4 xl:col-span-2">
            <div className="text-xs font-semibold uppercase tracking-[0.24em] text-slate-400">
              Terminal Endpoint
            </div>
            <div className="mt-2 font-mono text-sm text-slate-700">{toolServerInfo.terminalUrl}</div>
            <div className="mt-3 text-xs leading-6 text-slate-500">
              `POST /api/tools/terminal`
              接受 `command`、可选 `shell/workdir/timeout_ms`，返回扁平结构化执行结果。
              修改路径白名单后需要重启 Tool Router 才会生效。
            </div>
          </div>
        </div>
      </section>

      <section className="admin-surface rounded-3xl p-5">
        <div className="text-sm font-semibold text-slate-900">服务状态摘要</div>
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
