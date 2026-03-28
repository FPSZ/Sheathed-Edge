import { useEffect, useState } from "react";

import { PageHeader } from "../components/PageHeader";
import { StatusCard } from "../components/StatusCard";
import { apiGet } from "../lib/api";
import { formatJson, formatTime } from "../lib/format";
import type { HostIPsResponse, OverviewResponse } from "../lib/types";

export function DashboardPage() {
  const [data, setData] = useState<OverviewResponse | null>(null);
  const [error, setError] = useState<string>("");
  const [shareData, setShareData] = useState<HostIPsResponse | null>(null);
  const [shareLoading, setShareLoading] = useState(false);
  const [shareError, setShareError] = useState("");
  const [copied, setCopied] = useState<string>("");

  useEffect(() => {
    apiGet<OverviewResponse>("/internal/admin/overview")
      .then(setData)
      .catch((err: Error) => setError(err.message));
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

      {error ? <div className="rounded-2xl border border-rose-200 bg-rose-50 p-4 text-sm text-rose-700">{error}</div> : null}

      <section className="grid gap-4 md:grid-cols-2 2xl:grid-cols-3">
        {data?.services.map((service) => (
          <StatusCard
            key={service.name}
            title={service.name}
            status={service.status}
            subtitle={service.address}
            meta={service.message}
          />
        ))}
      </section>

      <section className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
        <div className="flex items-center justify-between gap-4">
          <div>
            <div className="text-sm font-semibold text-slate-900">队友访问链接</div>
            <div className="mt-1 text-xs text-slate-500">读取当前机器的局域网 IP，生成 Open WebUI 访问链接。</div>
          </div>
          <button
            className="admin-button"
            disabled={shareLoading}
            onClick={fetchShareLinks}
          >
            {shareLoading ? "读取中..." : "获取链接"}
          </button>
        </div>

        {shareError ? (
          <div className="mt-4 rounded-2xl border border-rose-200 bg-rose-50 p-3 text-sm text-rose-700">{shareError}</div>
        ) : null}

        {shareData && shareData.share_urls.length > 0 ? (
          <div className="mt-4 space-y-2">
            {shareData.share_urls.map((url) => (
              <div
                key={url}
                className="flex items-center justify-between gap-3 rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3"
              >
                <span className="font-mono text-sm text-slate-800">{url}</span>
                <button
                  className="admin-button"
                  onClick={() => copyURL(url)}
                >
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
        <div className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
          <div className="text-sm font-semibold text-slate-900">当前活动模型</div>
          <pre className="mt-4 whitespace-pre-wrap break-words rounded-2xl bg-slate-50 p-4 text-xs leading-6 text-slate-700">
            {formatJson(data?.active_model ?? {})}
          </pre>
        </div>
        <div className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
          <div className="text-sm font-semibold text-slate-900">最近失败</div>
          <div className="mt-4 space-y-3">
            {(data?.recent_failures ?? []).slice(0, 5).map((entry, index) => (
              <div key={index} className="rounded-2xl border border-slate-200 bg-slate-50 p-3">
                <div className="text-xs text-slate-500">{formatTime(entry.time)}</div>
                <div className="mt-1 text-sm text-slate-800">{String(entry.reason ?? entry.message ?? "unknown error")}</div>
              </div>
            ))}
          </div>
        </div>
      </section>
    </div>
  );
}
