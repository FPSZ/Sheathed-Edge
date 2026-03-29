import { useEffect, useState } from "react";

import { PageHeader } from "../components/PageHeader";
import { apiGet } from "../lib/api";
import { formatJson } from "../lib/format";
import type { ServicesResponse } from "../lib/types";

export function SettingsPage() {
  const [services, setServices] = useState<ServicesResponse | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    apiGet<ServicesResponse>("/internal/admin/services")
      .then(setServices)
      .catch((err: Error) => setError(err.message));
  }, []);

  return (
    <div className="space-y-6">
      <PageHeader title="系统设置" description="只读展示当前运行地址与服务配置摘要。" />
      {error ? (
        <div className="admin-surface rounded-3xl bg-rose-50 p-4 text-sm text-rose-700">
          {error}
        </div>
      ) : null}
      <section className="admin-surface rounded-3xl p-5">
        <div className="text-sm font-semibold text-slate-900">服务配置摘要</div>
        <pre className="admin-surface-muted mt-4 whitespace-pre-wrap break-words rounded-3xl p-4 text-xs leading-6 text-slate-700">
          {formatJson(services ?? {})}
        </pre>
      </section>
    </div>
  );
}
