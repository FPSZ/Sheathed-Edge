import { useEffect, useState } from "react";

import { PageHeader } from "../components/PageHeader";
import { apiGet } from "../lib/api";
import type { ModeDefinition, ModesResponse } from "../lib/types";

export function ModesPage() {
  const [data, setData] = useState<ModesResponse | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    apiGet<ModesResponse>("/internal/admin/modes")
      .then(setData)
      .catch((err: Error) => setError(err.message));
  }, []);

  return (
    <div className="space-y-6">
      <PageHeader
        title="模式与插件"
        description="查看 awdp 主模式与 web / pwn 插件的结构化定义。"
      />
      {error ? <div className="rounded-2xl border border-rose-200 bg-rose-50 p-4 text-sm text-rose-700">{error}</div> : null}

      {data ? (
        <div className="grid gap-4 xl:grid-cols-3">
          <ModeCard title="Core Mode" mode={data.core} />
          {data.plugins.map((plugin) => (
            <ModeCard key={plugin.name} title="Plugin" mode={plugin} />
          ))}
        </div>
      ) : null}
    </div>
  );
}

function ModeCard({ title, mode }: { title: string; mode: ModeDefinition }) {
  return (
    <section className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
      <div className="text-xs uppercase tracking-[0.18em] text-slate-400">{title}</div>
      <div className="mt-2 text-lg font-semibold text-slate-950">{mode.name}</div>
      <div className="mt-1 text-sm text-slate-500">{mode.type}</div>

      <Block label="Prompt Files" items={mode.prompt_files} />
      <Block label="Tool Scope" items={mode.tool_scope} />
      <Block label="Retrieval Roots" items={mode.retrieval_roots} />
      <Block label="Eval Tags" items={mode.eval_tags} />
    </section>
  );
}

function Block({ label, items }: { label: string; items: string[] }) {
  return (
    <div className="mt-4">
      <div className="mb-2 text-xs uppercase tracking-[0.16em] text-slate-400">{label}</div>
      <div className="space-y-2">
        {items.map((item) => (
          <div key={item} className="rounded-xl border border-slate-200 bg-slate-50 px-3 py-2 text-xs text-slate-700">
            {item}
          </div>
        ))}
      </div>
    </div>
  );
}
