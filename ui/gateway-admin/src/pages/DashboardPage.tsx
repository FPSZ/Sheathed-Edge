import { useEffect, useState } from "react";

import { PageHeader } from "../components/PageHeader";
import { StatusCard } from "../components/StatusCard";
import { apiGet, apiPost } from "../lib/api";
import { formatJson, formatTime } from "../lib/format";
import type { HostIPsResponse, OverviewResponse, ServiceStatus } from "../lib/types";

type ActionMap = Record<string, "start" | "stop" | "">;

export function DashboardPage() {
  const [data, setData] = useState<OverviewResponse | null>(null);
  const [error, setError] = useState("");
  const [shareData, setShareData] = useState<HostIPsResponse | null>(null);
  const [shareLoading, setShareLoading] = useState(false);
  const [shareError, setShareError] = useState("");
  const [copied, setCopied] = useState("");
  const [actions, setActions] = useState<ActionMap>({});

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

  async function refreshServices() {
    for (let attempt = 0; attempt < 6; attempt += 1) {
      await delay(attempt === 0 ? 400 : 1200);
      try {
        const response = await apiGet<OverviewResponse>("/internal/admin/overview");
        setData(response);
        setError("");
        return;
      } catch {
        // keep retrying for slow starts
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
        description="系统可用性、当前模型状态、最近会话与失败摘要。"
      />

      {error ? (
        <div className="admin-surface rounded-3xl bg-rose-50 p-4 text-sm text-rose-700">
          {error}
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
                    disabled={!service.control.can_start || pending !== ""}
                    onClick={() => runServiceAction(service, "start")}
                    title={service.control.unsupported_reason}
                  >
                    {pending === "start" ? "启动中..." : "启动"}
                  </button>
                  <button
                    className="admin-button danger"
                    disabled={!service.control.can_stop || pending !== ""}
                    onClick={() => runServiceAction(service, "stop")}
                    title={service.control.unsupported_reason}
                  >
                    {pending === "stop" ? "停止中..." : "停止"}
                  </button>
                </>
              }
            />
          );
        })}
      </section>

      <section className="admin-surface rounded-3xl p-5">
        <div className="flex items-center justify-between gap-4">
          <div>
            <div className="text-sm font-semibold text-slate-900">队友访问链接</div>
            <div className="mt-1 text-xs text-slate-500">
              读取当前机器的局域网 IP，生成 Open WebUI 访问链接。
            </div>
          </div>
          <button className="admin-button" disabled={shareLoading} onClick={fetchShareLinks}>
            {shareLoading ? "读取中..." : "获取链接"}
          </button>
        </div>

        {shareError ? (
          <div className="admin-surface rounded-3xl bg-rose-50 p-3 text-sm text-rose-700">
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
                <button className="admin-button" onClick={() => copyURL(url)}>
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

function delay(ms: number) {
  return new Promise((resolve) => window.setTimeout(resolve, ms));
}
