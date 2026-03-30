import { useEffect, useState } from "react";

import { useAdminScope } from "../app/AdminScopeContext";
import { PageHeader } from "../components/PageHeader";
import { StatusCard } from "../components/StatusCard";
import { apiGet, apiPost } from "../lib/api";
import { formatJson, formatTime } from "../lib/format";
import type {
  HostIPsResponse,
  OverviewResponse,
  SelfCheckResponse,
  ServiceStatus,
  StartAllResponse,
  UserWorkspaceResponse,
} from "../lib/types";

type ActionMap = Record<string, "start" | "stop" | "">;

export function DashboardPage() {
  const { selectedUserEmail } = useAdminScope();
  const [data, setData] = useState<OverviewResponse | null>(null);
  const [workspace, setWorkspace] = useState<UserWorkspaceResponse | null>(null);
  const [error, setError] = useState("");
  const [shareData, setShareData] = useState<HostIPsResponse | null>(null);
  const [shareLoading, setShareLoading] = useState(false);
  const [shareError, setShareError] = useState("");
  const [copied, setCopied] = useState("");
  const [actions, setActions] = useState<ActionMap>({});
  const [stackAction, setStackAction] = useState<"" | "start-all" | "self-check">("");
  const [stackNotice, setStackNotice] = useState("");
  const [selfCheck, setSelfCheck] = useState<SelfCheckResponse | null>(null);

  async function loadOverview() {
    try {
      const response = await apiGet<OverviewResponse>("/internal/admin/overview");
      setData(response);
      setError("");
    } catch (err) {
      setError((err as Error).message);
    }
  }

  useEffect(() => {
    loadOverview();
  }, []);

  useEffect(() => {
    if (!selectedUserEmail) {
      setWorkspace(null);
      return;
    }
    apiGet<UserWorkspaceResponse>(
      `/internal/admin/users/workspace?user_email=${encodeURIComponent(selectedUserEmail)}`,
    )
      .then((response) => setWorkspace(response))
      .catch(() => setWorkspace(null));
  }, [selectedUserEmail]);

  async function fetchShareLinks() {
    setShareLoading(true);
    setShareError("");
    try {
      const resp = await apiGet<HostIPsResponse>("/internal/admin/host-ips");
      setShareData(resp);
    } catch (err) {
      setShareError((err as Error).message);
    } finally {
      setShareLoading(false);
    }
  }

  async function runServiceAction(service: ServiceStatus, action: "start" | "stop") {
    setActions((prev) => ({ ...prev, [service.name]: action }));
    setError("");
    try {
      await apiPost(`/internal/admin/services/${action}`, { name: service.name });
      await refreshServices();
    } catch (err) {
      setError((err as Error).message);
      await loadOverview();
    } finally {
      setActions((prev) => ({ ...prev, [service.name]: "" }));
    }
  }

  async function runStartAll() {
    setStackAction("start-all");
    setError("");
    setStackNotice("");
    try {
      const response = await apiPost<StartAllResponse>("/internal/admin/start-all");
      const failed = response.results.filter((item) => !item.ok);
      setStackNotice(
        failed.length === 0
          ? "全部启动完成，服务已进入健康检查。"
          : `启动完成，但有 ${failed.length} 个服务未通过检查。`,
      );
      await refreshServices();
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setStackAction("");
    }
  }

  async function runSelfCheck() {
    setStackAction("self-check");
    setError("");
    setStackNotice("");
    try {
      const response = await apiGet<SelfCheckResponse>("/internal/admin/self-check");
      setSelfCheck(response);
      setStackNotice(response.ok ? "全链路自检通过。" : "全链路自检发现失败项。");
      await loadOverview();
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setStackAction("");
    }
  }

  async function refreshServices() {
    for (let attempt = 0; attempt < 8; attempt += 1) {
      await delay(attempt === 0 ? 500 : 1500);
      try {
        const response = await apiGet<OverviewResponse>("/internal/admin/overview");
        setData(response);
        setError("");
        return;
      } catch {
        // retry until services finish booting
      }
    }
    await loadOverview();
  }

  function copyURL(url: string) {
    navigator.clipboard.writeText(url).then(() => {
      setCopied(url);
      setTimeout(() => setCopied(""), 2000);
    });
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="总览"
        description={`系统运行态与当前用户工作区摘要。当前视角：${selectedUserEmail || "全部用户 / All Users"}`}
        action={
          <div className="flex items-center gap-2">
            <button
              className="admin-button"
              disabled={stackAction !== ""}
              onClick={runStartAll}
              type="button"
            >
              {stackAction === "start-all" ? "启动中..." : "全部启动"}
            </button>
            <button
              className="admin-button"
              disabled={stackAction !== ""}
              onClick={runSelfCheck}
              type="button"
            >
              {stackAction === "self-check" ? "检查中..." : "全链路自检"}
            </button>
          </div>
        }
      />

      {error ? (
        <div className="admin-surface rounded-3xl bg-rose-50 p-4 text-sm text-rose-700">
          {error}
        </div>
      ) : null}

      {stackNotice ? (
        <div className="admin-surface rounded-3xl bg-emerald-50 p-4 text-sm text-emerald-700">
          {stackNotice}
        </div>
      ) : null}

      <section className="grid gap-4 md:grid-cols-2 2xl:grid-cols-3">
        {data?.services.map((service) => {
          const pending = actions[service.name] ?? "";
          return (
            <StatusCard
              key={service.name}
              title={service.name}
              status={service.status}
              subtitle={service.address}
              meta={service.message ?? service.control.unsupported_reason}
              actions={
                <>
                  <button
                    className="admin-button"
                    disabled={!service.control.can_start || service.status === "ok" || pending !== ""}
                    onClick={() => runServiceAction(service, "start")}
                    title={service.control.unsupported_reason}
                    type="button"
                  >
                    {pending === "start" ? "启动中..." : "启动"}
                  </button>
                  <button
                    className="admin-button danger"
                    disabled={!service.control.can_stop || service.status !== "ok" || pending !== ""}
                    onClick={() => runServiceAction(service, "stop")}
                    title={service.control.unsupported_reason}
                    type="button"
                  >
                    {pending === "stop" ? "停止中..." : "停止"}
                  </button>
                </>
              }
            />
          );
        })}
      </section>

      {selfCheck ? (
        <section className="admin-surface rounded-3xl p-5">
          <div className="flex items-center justify-between gap-4">
            <div>
              <div className="text-sm font-semibold text-slate-900">全链路自检结果</div>
              <div className="mt-1 text-xs text-slate-500">
                覆盖 Gateway、模型、聊天、Terminal、MCP 与可选 SSH 路径。
              </div>
            </div>
            <span className={`admin-badge ${selfCheck.ok ? "ok" : "down"}`}>
              {selfCheck.ok ? "ok" : "failed"}
            </span>
          </div>
          <div className="mt-4 space-y-3">
            {selfCheck.checks.map((item) => (
              <div key={item.id} className="admin-surface-muted rounded-3xl p-4">
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <div className="text-sm font-medium text-slate-900">{item.label}</div>
                    <div className="mt-1 text-xs text-slate-500">{item.message}</div>
                  </div>
                  <span
                    className={`admin-badge ${
                      item.status === "ok" ? "ok" : item.status === "skipped" ? "" : "down"
                    }`}
                  >
                    {item.status}
                  </span>
                </div>
                {item.details ? (
                  <pre className="mt-3 whitespace-pre-wrap break-words text-xs leading-6 text-slate-600">
                    {formatJson(item.details)}
                  </pre>
                ) : null}
              </div>
            ))}
          </div>
        </section>
      ) : null}

      {selectedUserEmail && workspace ? (
        <section className="admin-surface rounded-3xl p-5">
          <div className="text-sm font-semibold text-slate-900">当前用户摘要</div>
          <div className="mt-4 grid gap-4 md:grid-cols-3">
            <Metric title="用户" value={workspace.workspace.user_email} />
            <Metric title="默认 SSH" value={workspace.workspace.default_ssh_host_id || "未设置"} />
            <Metric
              title="可访问目录"
              value={String(workspace.workspace.terminal_allowed_paths.length)}
            />
          </div>
        </section>
      ) : null}

      <section className="admin-surface rounded-3xl p-5">
        <div className="flex items-center justify-between gap-4">
          <div>
            <div className="text-sm font-semibold text-slate-900">队友访问地址</div>
            <div className="mt-1 text-xs text-slate-500">
              读取当前机器的局域网 IP，生成 Open WebUI 可访问地址。
            </div>
          </div>
          <button className="admin-button" disabled={shareLoading} onClick={fetchShareLinks} type="button">
            {shareLoading ? "读取中..." : "获取地址"}
          </button>
        </div>

        {shareError ? (
          <div className="mt-4 admin-surface rounded-3xl bg-rose-50 p-3 text-sm text-rose-700">
            {shareError}
          </div>
        ) : null}

        {shareData && shareData.share_urls.length > 0 ? (
          <div className="mt-4 space-y-2">
            {shareData.share_urls.map((url) => (
              <div
                key={url}
                className="admin-surface-muted flex items-center justify-between gap-3 rounded-3xl px-4 py-3"
              >
                <span className="font-mono text-sm text-slate-800">{url}</span>
                <button className="admin-button" onClick={() => copyURL(url)} type="button">
                  {copied === url ? "已复制" : "复制"}
                </button>
              </div>
            ))}
          </div>
        ) : shareData && shareData.share_urls.length === 0 ? (
          <div className="mt-4 text-sm text-slate-500">未找到可用的局域网 IP。</div>
        ) : null}
      </section>

      <section className="grid gap-4 xl:grid-cols-[1.2fr_1fr]">
        <div className="admin-surface rounded-3xl p-5">
          <div className="text-sm font-semibold text-slate-900">当前活动模型</div>
          <pre className="admin-surface-muted mt-4 whitespace-pre-wrap break-words rounded-3xl p-4 text-xs leading-6 text-slate-700">
            {formatJson(data?.active_model ?? {})}
          </pre>
        </div>
        <div className="admin-surface rounded-3xl p-5">
          <div className="text-sm font-semibold text-slate-900">最近失败</div>
          <div className="mt-4 space-y-3">
            {(data?.recent_failures ?? []).slice(0, 5).map((entry, index) => (
              <div key={index} className="admin-surface-muted rounded-3xl p-3">
                <div className="text-xs text-slate-500">{formatTime(entry.time)}</div>
                <div className="mt-1 text-sm text-slate-800">
                  {String(entry.reason ?? entry.message ?? "unknown error")}
                </div>
              </div>
            ))}
          </div>
        </div>
      </section>
    </div>
  );
}

function Metric({ title, value }: { title: string; value: string }) {
  return (
    <div className="admin-surface-muted rounded-3xl p-4">
      <div className="text-xs uppercase tracking-[0.16em] text-slate-400">{title}</div>
      <div className="mt-2 break-all text-sm text-slate-900">{value}</div>
    </div>
  );
}

function delay(ms: number) {
  return new Promise((resolve) => window.setTimeout(resolve, ms));
}
