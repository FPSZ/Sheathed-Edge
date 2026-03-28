import { useEffect, useState } from "react";

import { PageHeader } from "../components/PageHeader";
import { apiGet, apiPost } from "../lib/api";
import type { ModelsResponse } from "../lib/types";

type ActionState = "idle" | "pending";

export function ModelsPage() {
  const [data, setData] = useState<ModelsResponse | null>(null);
  const [error, setError] = useState("");
  const [actionState, setActionState] = useState<ActionState>("idle");

  async function load() {
    try {
      const response = await apiGet<ModelsResponse>("/internal/admin/models");
      setData(response);
      setError("");
    } catch (err) {
      setError((err as Error).message);
    }
  }

  useEffect(() => {
    load();
  }, []);

  async function runAction(path: string, body?: Record<string, unknown>) {
    setActionState("pending");
    try {
      await apiPost(path, body ?? {});
      await load();
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setActionState("idle");
    }
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="模型控制"
        description="管理当前活动 profile，并执行 llama-server 启停、重启与切换。"
        action={
          <div className="flex gap-2">
            <button className="admin-button" disabled={actionState === "pending"} onClick={() => runAction("/internal/admin/llama/start")}>启动</button>
            <button className="admin-button" disabled={actionState === "pending"} onClick={() => runAction("/internal/admin/llama/restart")}>重启</button>
            <button className="admin-button danger" disabled={actionState === "pending"} onClick={() => runAction("/internal/admin/llama/stop")}>停止</button>
          </div>
        }
      />

      {error ? <div className="rounded-2xl border border-rose-200 bg-rose-50 p-4 text-sm text-rose-700">{error}</div> : null}

      <section className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
        <div className="text-sm font-semibold text-slate-900">当前状态</div>
        <dl className="mt-4 grid gap-4 md:grid-cols-2 xl:grid-cols-4">
          <Meta label="活动 Profile" value={data?.active_profile_id ?? "-"} />
          <Meta label="运行状态" value={data?.active_model.running ? "running" : "stopped"} />
          <Meta label="PID" value={data?.active_model.pid ? String(data.active_model.pid) : "-"} />
          <Meta label="模型路径" value={data?.active_model.model_path ?? "-"} />
        </dl>
      </section>

      <section className="grid gap-4 xl:grid-cols-2">
        {(data?.profiles ?? []).map((profile) => (
          <div key={profile.id} className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
            <div className="flex items-start justify-between gap-3">
              <div>
                <div className="text-base font-semibold text-slate-950">{profile.label}</div>
                <div className="mt-1 text-xs text-slate-500">{profile.id}</div>
              </div>
              <span className={`rounded-full px-2.5 py-1 text-xs ${profile.enabled ? "bg-emerald-50 text-emerald-700" : "bg-slate-100 text-slate-500"}`}>
                {profile.enabled ? "enabled" : "disabled"}
              </span>
            </div>
            <dl className="mt-4 space-y-2 text-sm text-slate-600">
              <MetaLine label="量化" value={profile.quant} />
              <MetaLine label="ctx" value={String(profile.ctx_size)} />
              <MetaLine label="threads" value={String(profile.threads)} />
              <MetaLine label="parallel" value={String(profile.parallel)} />
            </dl>
            <div className="mt-4">
              <button
                className="admin-button"
                disabled={actionState === "pending" || !profile.enabled}
                onClick={() => runAction("/internal/admin/models/switch", { profile_id: profile.id })}
              >
                切换到该 Profile
              </button>
            </div>
          </div>
        ))}
      </section>
    </div>
  );
}

function Meta({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-2xl border border-slate-200 bg-slate-50 p-4">
      <dt className="text-xs uppercase tracking-[0.18em] text-slate-400">{label}</dt>
      <dd className="mt-2 break-all text-sm text-slate-800">{value}</dd>
    </div>
  );
}

function MetaLine({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between gap-3">
      <span className="text-slate-500">{label}</span>
      <span className="break-all text-right text-slate-800">{value}</span>
    </div>
  );
}
