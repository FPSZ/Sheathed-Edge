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
      <PageHeader title="系统设置" description="当前仅只读展示运行地址与服务信息摘要。" />
      {error ? <div className="rounded-2xl border border-rose-200 bg-rose-50 p-4 text-sm text-rose-700">{error}</div> : null}
      <section className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
        <div className="text-sm font-semibold text-slate-900">服务配置摘要</div>
        <pre className="mt-4 whitespace-pre-wrap break-words rounded-2xl bg-slate-50 p-4 text-xs leading-6 text-slate-700">
          {formatJson(services ?? {})}
        </pre>
      </section>
    </div>
  );
}
