import { useEffect, useMemo, useState } from "react";

import { InlineSelect } from "../components/InlineSelect";
import { PageHeader } from "../components/PageHeader";
import { apiGet, apiPost } from "../lib/api";
import type { ModelProfile, ModelsResponse } from "../lib/types";

type ActionState = "idle" | "pending";
type DraftMap = Record<string, ModelProfile>;

export function ModelsPage() {
  const [data, setData] = useState<ModelsResponse | null>(null);
  const [drafts, setDrafts] = useState<DraftMap>({});
  const [error, setError] = useState("");
  const [actionState, setActionState] = useState<ActionState>("idle");

  async function load() {
    try {
      const response = await apiGet<ModelsResponse>("/internal/admin/models");
      setData(response);
      setDrafts(
        Object.fromEntries(response.profiles.map((profile) => [profile.id, { ...profile }])),
      );
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

  async function switchAndStart(profileId: string) {
    setActionState("pending");
    try {
      await apiPost("/internal/admin/models/switch", { profile_id: profileId });
      await apiPost("/internal/admin/llama/start", {});
      await load();
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setActionState("idle");
    }
  }

  async function saveProfile(profileId: string, applyNow: boolean) {
    const draft = drafts[profileId];
    if (!draft) {
      return;
    }

    setActionState("pending");
    try {
      await apiPost("/internal/admin/models/update", {
        profile: draft,
        apply_now: applyNow,
      });
      await load();
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setActionState("idle");
    }
  }

  function updateDraft(profileId: string, patch: Partial<ModelProfile>) {
    setDrafts((prev) => ({
      ...prev,
      [profileId]: {
        ...prev[profileId],
        ...patch,
      },
    }));
  }

  const activeProfile = useMemo(() => {
    return data?.profiles.find((profile) => profile.id === data.active_profile_id) ?? null;
  }, [data]);

  return (
    <div className="space-y-6">
      <PageHeader
        title="模型控制"
        description="管理当前活动模型、手动启停 llama-server，并直接在原位调整 profile 运行参数。"
        action={
          <div className="flex shrink-0 flex-nowrap items-center gap-2">
            <button
              className="admin-button"
              disabled={actionState === "pending"}
              onClick={() => runAction("/internal/admin/llama/start")}
            >
              启动当前模型
            </button>
            <button
              className="admin-button"
              disabled={actionState === "pending"}
              onClick={() => runAction("/internal/admin/llama/restart")}
            >
              重启
            </button>
            <button
              className="admin-button danger"
              disabled={actionState === "pending"}
              onClick={() => runAction("/internal/admin/llama/stop")}
            >
              停止
            </button>
          </div>
        }
      />

      {error ? (
        <div className="admin-surface rounded-3xl bg-rose-50 p-4 text-sm text-rose-700">
          {error}
        </div>
      ) : null}

      <section className="admin-surface rounded-3xl p-5">
        <div className="flex items-center justify-between gap-3">
          <div>
            <div className="text-sm font-semibold text-slate-900">当前状态</div>
            <div className="mt-1 text-xs text-slate-500">
              当前活动 profile 决定了 WebUI 默认命中的底层模型。
            </div>
          </div>
          <span className={`admin-badge ${data?.active_model.running ? "success" : "muted"}`}>
            {data?.active_model.running ? "运行中" : "已停止"}
          </span>
        </div>

        <dl className="mt-4 grid gap-4 md:grid-cols-2 xl:grid-cols-4">
          <Meta label="活动 Profile" value={data?.active_profile_id ?? "-"} />
          <Meta label="显示名称" value={activeProfile?.label ?? data?.active_model.label ?? "-"} />
          <Meta label="PID" value={data?.active_model.pid ? String(data.active_model.pid) : "-"} />
          <Meta
            label="模型路径"
            value={data?.active_model.model_path ?? activeProfile?.model_path ?? "-"}
          />
        </dl>
      </section>

      <section className="grid gap-4 xl:grid-cols-2">
        {(data?.profiles ?? []).map((profile) => {
          const isActive = profile.id === data?.active_profile_id;
          const isRunning = isActive && Boolean(data?.active_model.running);
          const draft = drafts[profile.id] ?? profile;

          return (
            <ModelCard
              key={profile.id}
              draft={draft}
              isActive={isActive}
              isRunning={isRunning}
              actionState={actionState}
              onChange={(patch) => updateDraft(profile.id, patch)}
              onSwitch={() => runAction("/internal/admin/models/switch", { profile_id: profile.id })}
              onSwitchAndStart={() => switchAndStart(profile.id)}
              onStop={() => runAction("/internal/admin/llama/stop")}
              onStart={() => runAction("/internal/admin/llama/start")}
              onSave={() => saveProfile(profile.id, false)}
              onSaveAndApply={() => saveProfile(profile.id, true)}
            />
          );
        })}
      </section>
    </div>
  );
}

type ModelCardProps = {
  draft: ModelProfile;
  isActive: boolean;
  isRunning: boolean;
  actionState: ActionState;
  onChange: (patch: Partial<ModelProfile>) => void;
  onSwitch: () => void;
  onSwitchAndStart: () => void;
  onStop: () => void;
  onStart: () => void;
  onSave: () => void;
  onSaveAndApply: () => void;
};

function ModelCard({
  draft,
  isActive,
  isRunning,
  actionState,
  onChange,
  onSwitch,
  onSwitchAndStart,
  onStop,
  onStart,
  onSave,
  onSaveAndApply,
}: ModelCardProps) {
  const disabled = actionState === "pending";

  return (
    <div
      className={`rounded-3xl p-5 ${
        isActive ? "admin-surface-strong bg-sky-50/50" : "admin-surface"
      }`}
    >
      <div className="flex items-start justify-between gap-3">
        <div>
          <div className="text-base font-semibold text-slate-950">{draft.label}</div>
          <div className="mt-1 text-xs text-slate-500">{draft.id}</div>
        </div>
        <div className="grid w-[13.5rem] grid-cols-2 gap-2">
          <Badge tone={draft.enabled ? "success" : "muted"} compact>
            {draft.enabled ? "可用" : "已禁用"}
          </Badge>
          <Badge tone="muted" compact>
            {draft.quant}
          </Badge>
          <Badge tone="info" compact hidden={!isActive}>
            当前活动
          </Badge>
          <Badge tone="success" compact hidden={!isRunning}>
            正在运行
          </Badge>
        </div>
      </div>

      <dl className="mt-4 space-y-2 text-sm text-slate-600">
        <EditableMetric
          label="上下文/人"
          value={Math.round(draft.ctx_size / Math.max(1, draft.parallel))}
          disabled={disabled}
          kind="select"
          options={CONTEXT_OPTIONS}
          onChange={(value) => onChange({ ctx_size: toNumber(value, draft.ctx_size) * draft.parallel })}
        />
        <EditableMetric
          label="线程"
          value={draft.threads}
          kind="number"
          disabled={disabled}
          onChange={(value) => onChange({ threads: toNumber(value, draft.threads) })}
        />
        <EditableMetric
          label="并发"
          value={draft.parallel}
          kind="number"
          disabled={disabled}
          onChange={(value) => onChange({ parallel: toNumber(value, draft.parallel) })}
        />
        <EditableMetric
          label="GPU Layers"
          value={draft.n_gpu_layers}
          kind="number"
          disabled={disabled}
          onChange={(value) => onChange({ n_gpu_layers: toNumber(value, draft.n_gpu_layers) })}
        />
        <ToggleMetric
          label="Flash Attn"
          value={draft.flash_attn}
          disabled={disabled}
          onToggle={() => onChange({ flash_attn: !draft.flash_attn })}
        />
        <ToggleMetric
          label="Enabled"
          value={draft.enabled}
          disabled={disabled}
          onToggle={() => onChange({ enabled: !draft.enabled })}
        />
      </dl>

      <div className="mt-5 flex flex-wrap gap-2">
        {isActive ? (
          <>
            <button className="admin-button" disabled={disabled || isRunning} onClick={onStart}>
              启动此模型
            </button>
            <button className="admin-button danger" disabled={disabled || !isRunning} onClick={onStop}>
              停止此模型
            </button>
          </>
        ) : (
          <>
            <button className="admin-button" disabled={disabled} onClick={onSwitch}>
              设为活动模型
            </button>
            <button className="admin-button" disabled={disabled || !draft.enabled} onClick={onSwitchAndStart}>
              切换并启动
            </button>
          </>
        )}
      </div>

      <div className="mt-3 flex flex-wrap gap-2">
        <button className="admin-button" disabled={disabled} onClick={onSave}>
          保存参数
        </button>
        <button
          className="admin-button"
          disabled={disabled || !(isActive && isRunning)}
          onClick={onSaveAndApply}
        >
          保存并重启应用
        </button>
      </div>
    </div>
  );
}

function EditableMetric({
  label,
  value,
  onChange,
  disabled,
  kind = "text",
  options = [],
}: {
  label: string;
  value: string | number;
  onChange: (value: string) => void;
  disabled: boolean;
  kind?: "text" | "number" | "select";
  options?: Array<{ label: string; value: number }>;
}) {
  return (
    <div className="flex items-center justify-between gap-3">
      <span className="text-slate-500">{label}</span>
      <div className="min-w-[7rem]">
        {kind === "select" ? (
          <InlineSelect
            value={value}
            options={options}
            disabled={disabled}
            onChange={(nextValue) => onChange(String(nextValue))}
          />
        ) : (
          <input
            className="admin-inline-input"
            type={kind}
            value={String(value)}
            disabled={disabled}
            onChange={(event) => onChange(event.target.value)}
          />
        )}
      </div>
    </div>
  );
}

function ToggleMetric({
  label,
  value,
  onToggle,
  disabled,
}: {
  label: string;
  value: boolean;
  onToggle: () => void;
  disabled: boolean;
}) {
  return (
    <div className="flex items-center justify-between gap-3">
      <span className="text-slate-500">{label}</span>
      <button className="admin-inline-toggle" disabled={disabled} onClick={onToggle}>
        {value ? "On" : "Off"}
      </button>
    </div>
  );
}

function Badge({
  children,
  tone,
  compact = false,
  hidden = false,
}: {
  children: string;
  tone: "success" | "info" | "muted";
  compact?: boolean;
  hidden?: boolean;
}) {
  return (
    <span
      className={`admin-badge ${tone} ${compact ? "admin-badge-grid" : ""} ${hidden ? "pointer-events-none opacity-0" : ""}`}
    >
      {children}
    </span>
  );
}

function Meta({ label, value }: { label: string; value: string }) {
  return (
    <div className="admin-surface-muted rounded-3xl p-4">
      <dt className="text-xs uppercase tracking-[0.18em] text-slate-400">{label}</dt>
      <dd className="mt-2 break-all text-sm text-slate-800">{value}</dd>
    </div>
  );
}

function toNumber(value: string, fallback: number) {
  const parsed = Number.parseInt(value, 10);
  return Number.isNaN(parsed) ? fallback : parsed;
}

const CONTEXT_OPTIONS = [
  { label: "16k", value: 16 * 1024 },
  { label: "32k", value: 32 * 1024 },
  { label: "48k", value: 48 * 1024 },
  { label: "64k", value: 64 * 1024 },
  { label: "128k", value: 128 * 1024 },
  { label: "256k", value: 256 * 1024 },
];
