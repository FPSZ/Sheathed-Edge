import { useEffect, useState } from "react";

import { PageHeader } from "../components/PageHeader";
import { apiGet } from "../lib/api";
import { formatJson, formatTime } from "../lib/format";
import type { LogListResponse } from "../lib/types";

export function LogsPage() {
  const [sessions, setSessions] = useState<Record<string, unknown>[]>([]);
  const [tools, setTools] = useState<Record<string, unknown>[]>([]);
  const [error, setError] = useState("");

  useEffect(() => {
    Promise.all([
      apiGet<LogListResponse>("/internal/admin/logs/sessions"),
      apiGet<LogListResponse>("/internal/admin/logs/tools"),
    ])
      .then(([sessionData, toolData]) => {
        setSessions(sessionData.items);
        setTools(toolData.items);
      })
      .catch((err: Error) => setError(err.message));
  }, []);

  return (
    <div className="space-y-6">
      <PageHeader title="日志与审计" description="查看最近会话日志与工具调用摘要。" />
      {error ? <div className="admin-surface rounded-3xl bg-rose-50 p-4 text-sm text-rose-700">{error}</div> : null}

      <div className="grid gap-4 xl:grid-cols-2">
        <LogColumn title="会话日志" items={sessions} />
        <LogColumn title="工具日志" items={tools} />
      </div>
    </div>
  );
}

function LogColumn({ title, items }: { title: string; items: Record<string, unknown>[] }) {
  return (
    <section className="admin-surface rounded-3xl p-5">
      <div className="text-sm font-semibold text-slate-900">{title}</div>
      <div className="mt-4 space-y-3">
        {items.map((item, index) => (
          <div key={index} className="admin-surface-muted rounded-3xl p-4">
            <div className="text-xs text-slate-500">{formatTime(item.time)}</div>
            <pre className="mt-2 whitespace-pre-wrap break-words text-xs leading-6 text-slate-700">
              {formatJson(item)}
            </pre>
          </div>
        ))}
      </div>
    </section>
  );
}
