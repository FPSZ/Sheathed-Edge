import { type ReactNode, useEffect, useMemo, useState } from "react";

import { PageHeader } from "../components/PageHeader";
import { apiGet, apiPost } from "../lib/api";
import type { AgentLayerPreset, AgentLayersResponse } from "../lib/types";

type ActionState = "idle" | "pending";

function newPresetDraft(): AgentLayerPreset {
  const suffix = Math.random().toString(36).slice(2, 8);
  return {
    id: `preset-${suffix}`,
    label: `New Preset ${suffix}`,
    enable_agent_router: true,
    enable_reverse_skills: false,
    enable_pwn_skills: false,
    enable_web_skills: false,
    enable_awdp_red: false,
    enable_awdp_blue: false,
  };
}

export function AgentLayersPage() {
  const [data, setData] = useState<AgentLayersResponse | null>(null);
  const [presets, setPresets] = useState<AgentLayerPreset[]>([]);
  const [defaultPresetID, setDefaultPresetID] = useState("");
  const [editingID, setEditingID] = useState("");
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [actionState, setActionState] = useState<ActionState>("idle");

  async function load() {
    try {
      const response = await apiGet<AgentLayersResponse>("/internal/admin/agent-layers");
      setData(response);
      setPresets(response.presets);
      setDefaultPresetID(response.default_preset_id ?? "");
      setEditingID((current) => {
        if (current && response.presets.some((item) => item.id === current)) {
          return current;
        }
        return response.presets[0]?.id ?? "";
      });
      setError("");
    } catch (err) {
      setError((err as Error).message);
    }
  }

  useEffect(() => {
    void load();
  }, []);

  const editingPreset = useMemo(
    () => presets.find((item) => item.id === editingID) ?? null,
    [editingID, presets],
  );

  useEffect(() => {
    if (!editingPreset) {
      return;
    }
    window.sessionStorage.setItem("admin-agent-layers-selected-preset", JSON.stringify(editingPreset));
    window.dispatchEvent(new CustomEvent("admin-agent-layers-preset-changed"));
  }, [editingPreset]);

  function updatePreset(presetID: string, patch: Partial<AgentLayerPreset>) {
    setPresets((current) => current.map((item) => (item.id === presetID ? { ...item, ...patch } : item)));
  }

  function addPreset() {
    const draft = newPresetDraft();
    setPresets((current) => [...current, draft]);
    setEditingID(draft.id);
    setNotice("");
  }

  function removePreset() {
    if (!editingID) {
      return;
    }
    setPresets((current) => current.filter((item) => item.id !== editingID));
    setEditingID((current) => {
      if (current !== editingID) {
        return current;
      }
      const next = presets.find((item) => item.id !== editingID);
      return next?.id ?? "";
    });
    if (defaultPresetID === editingID) {
      setDefaultPresetID("");
    }
  }

  async function save() {
    setActionState("pending");
    setError("");
    setNotice("");
    try {
      const response = await apiPost<AgentLayersResponse>("/internal/admin/agent-layers", {
        default_preset_id: defaultPresetID,
        presets,
      });
      setData(response);
      setPresets(response.presets);
      setDefaultPresetID(response.default_preset_id ?? "");
      setNotice("Agent / Skills 预设已保存，后续会话可以复用这些组合。");
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setActionState("idle");
    }
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="Agent Layers"
        description="管理 Router、Reverse、Pwn、Web、AWDP Red、AWDP Blue 六层组合，并预览最终会挂载的 prompt、skills、tool scope 与 retrieval roots。Binary Core 会在 Reverse/Pwn 启用时自动注入，AWDP Core 会在 Red/Blue 启用时自动注入。"
        action={
          <div className="flex shrink-0 flex-nowrap items-center gap-2">
            <button className="admin-button" type="button" onClick={addPreset}>
              新建预设
            </button>
            <button className="admin-button" disabled={actionState === "pending"} type="button" onClick={save}>
              保存预设
            </button>
          </div>
        }
      />

      {error ? <div className="admin-surface rounded-3xl bg-rose-50 p-4 text-sm text-rose-700">{error}</div> : null}
      {notice ? <div className="admin-surface rounded-3xl bg-emerald-50 p-4 text-sm text-emerald-700">{notice}</div> : null}

      <div className="grid gap-4 xl:grid-cols-[340px_minmax(0,1fr)]">
        <section className="admin-surface rounded-3xl p-5">
          <div className="text-xs uppercase tracking-[0.18em] text-slate-400">Presets</div>
          <div className="mt-2 text-sm text-slate-500">{data?.config_path ?? "-"}</div>

          <div className="mt-4 space-y-2">
            {presets.map((preset) => {
              const isSelected = preset.id === editingID;
              const isDefault = preset.id === defaultPresetID;
              return (
                <button
                  key={preset.id}
                  className={[
                    "w-full rounded-2xl border px-3 py-3 text-left transition",
                    isSelected
                      ? "border-slate-900 bg-white text-slate-950"
                      : "border-transparent bg-[var(--panel-muted)] text-slate-700 hover:bg-white",
                  ].join(" ")}
                  type="button"
                  onClick={() => setEditingID(preset.id)}
                >
                  <div className="flex items-center justify-between gap-3">
                    <div className="font-medium">{preset.label}</div>
                    {isDefault ? <span className="admin-badge success">默认</span> : null}
                  </div>
                  <div className="mt-2 text-xs text-slate-500">{preset.id}</div>
                </button>
              );
            })}
          </div>
        </section>

        <section className="admin-surface rounded-3xl p-5">
          {editingPreset ? (
            <div className="space-y-5">
              <div className="grid gap-4 md:grid-cols-2">
                <Field label="Preset ID">
                  <input
                    className="admin-input"
                    value={editingPreset.id}
                    onChange={(event) => updatePreset(editingPreset.id, { id: event.target.value })}
                  />
                </Field>
                <Field label="Label">
                  <input
                    className="admin-input"
                    value={editingPreset.label}
                    onChange={(event) => updatePreset(editingPreset.id, { label: event.target.value })}
                  />
                </Field>
              </div>

              <div className="grid gap-3 grid-cols-1">
                <ToggleCard
                  checked={editingPreset.enable_agent_router}
                  description="共享路由入口，负责 task_family、competition_mode、phase 与 primary_skill 的统一约束。"
                  label="Agent Router"
                  onChange={(checked) => updatePreset(editingPreset.id, { enable_agent_router: checked })}
                />
                <ToggleCard
                  checked={editingPreset.enable_reverse_skills}
                  description="启用 reverse 家族的关键函数恢复、校验追踪、算法恢复与 flag 提取路线。"
                  label="Reverse Skills"
                  onChange={(checked) => updatePreset(editingPreset.id, { enable_reverse_skills: checked })}
                />
                <ToggleCard
                  checked={editingPreset.enable_pwn_skills}
                  description="启用 pwn 家族的保护判断、原语判断、利用路径与 patch / regression 约束。"
                  label="Pwn Skills"
                  onChange={(checked) => updatePreset(editingPreset.id, { enable_pwn_skills: checked })}
                />
                <ToggleCard
                  checked={editingPreset.enable_web_skills}
                  description="启用 web 家族的枚举、sink 判断、payload 迭代与回归约束。"
                  label="Web Skills"
                  onChange={(checked) => updatePreset(editingPreset.id, { enable_web_skills: checked })}
                />
                <ToggleCard
                  checked={editingPreset.enable_awdp_red}
                  description="启用 AWDP 红队工作流：快攻、复用、收 flag 与提交前后检查。"
                  label="AWDP Red"
                  onChange={(checked) => updatePreset(editingPreset.id, { enable_awdp_red: checked })}
                />
                <ToggleCard
                  checked={editingPreset.enable_awdp_blue}
                  description="启用 AWDP 蓝队工作流：风险分诊、最小修补、checker-safe 与回归。"
                  label="AWDP Blue"
                  onChange={(checked) => updatePreset(editingPreset.id, { enable_awdp_blue: checked })}
                />
                <ToggleCard
                  checked={defaultPresetID === editingPreset.id}
                  description="作为默认预设。Binary Core 与 AWDP Core 不单独暴露，会在相关层启用时自动挂载。"
                  label="Default Preset"
                  onChange={(checked) => setDefaultPresetID(checked ? editingPreset.id : "")}
                />
              </div>

              <div className="flex items-center gap-2">
                <button className="admin-button danger" type="button" onClick={removePreset}>
                  删除预设
                </button>
              </div>
            </div>
          ) : (
            <div className="text-sm text-slate-500">请选择或新建一个预设。</div>
          )}
        </section>
      </div>
    </div>
  );
}

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <label className="space-y-2">
      <div className="text-xs uppercase tracking-[0.16em] text-slate-400">{label}</div>
      {children}
    </label>
  );
}

function ToggleCard({
  label,
  description,
  checked,
  onChange,
}: {
  label: string;
  description: string;
  checked: boolean;
  onChange: (checked: boolean) => void;
}) {
  return (
    <div className="admin-surface-muted rounded-2xl p-4">
      <div className="flex items-start justify-between gap-4">
        <div className="min-w-0 flex-1">
          <div className="text-base font-medium text-slate-900">{label}</div>
          <div className="mt-1 text-sm leading-6 text-slate-500">{description}</div>
        </div>
        <label className="admin-switch" aria-label={label}>
          <input checked={checked} type="checkbox" onChange={(event) => onChange(event.target.checked)} />
          <span className="admin-switch-track" />
        </label>
      </div>
    </div>
  );
}
