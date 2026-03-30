import { useEffect, useMemo, useState } from "react";
import { ArrowRight, ExternalLink } from "lucide-react";
import { Link, useLocation } from "react-router-dom";

import { useAdminPageState } from "../app/AdminPageState";
import { apiGet } from "../lib/api";
import { formatTime } from "../lib/format";
import type {
  MCPOpenWebUIPreviewResponse,
  MCPRuntimeEntry,
  MCPServersResponse,
  ModelsResponse,
  ModesResponse,
  OverviewResponse,
  ServicesResponse,
  SSHBindingsResponse,
  SSHHostsResponse,
  TerminalPathsSettings,
} from "../lib/types";

type Props = {
  title: string;
  subtitle: string;
};

type DrawerRoute =
  | "/admin"
  | "/admin/models"
  | "/admin/modes"
  | "/admin/mcp"
  | "/admin/logs"
  | "/admin/settings";

export function DetailsDrawer({ title, subtitle }: Props) {
  const location = useLocation();
  const route = (location.pathname as DrawerRoute) || "/admin";
  const {
    logSource,
    setLogSource,
    logRange,
    setLogRange,
    logFailureOnly,
    setLogFailureOnly,
    selectedLog,
  } = useAdminPageState();
  const [overview, setOverview] = useState<OverviewResponse | null>(null);
  const [models, setModels] = useState<ModelsResponse | null>(null);
  const [modes, setModes] = useState<ModesResponse | null>(null);
  const [services, setServices] = useState<ServicesResponse | null>(null);
  const [terminalPaths, setTerminalPaths] = useState<TerminalPathsSettings | null>(null);
  const [sshHosts, setSshHosts] = useState<SSHHostsResponse | null>(null);
  const [sshBindings, setSshBindings] = useState<SSHBindingsResponse | null>(null);
  const [mcpServers, setMcpServers] = useState<MCPServersResponse | null>(null);
  const [mcpPreview, setMcpPreview] = useState<MCPOpenWebUIPreviewResponse | null>(null);

  useEffect(() => {
    let active = true;
    apiGet<OverviewResponse>("/internal/admin/overview")
      .then((response) => {
        if (active) {
          setOverview(response);
        }
      })
      .catch(() => {
        if (active) {
          setOverview(null);
        }
      });
    return () => {
      active = false;
    };
  }, [route]);

  useEffect(() => {
    let active = true;
    setModels(null);
    setModes(null);
    setServices(null);
    setTerminalPaths(null);
    setSshHosts(null);
    setSshBindings(null);
    setMcpServers(null);
    setMcpPreview(null);

    if (route === "/admin/models") {
      apiGet<ModelsResponse>("/internal/admin/models")
        .then((response) => {
          if (active) {
            setModels(response);
          }
        })
        .catch(() => {});
    }

    if (route === "/admin/modes") {
      apiGet<ModesResponse>("/internal/admin/modes")
        .then((response) => {
          if (active) {
            setModes(response);
          }
        })
        .catch(() => {});
    }

    if (route === "/admin/settings") {
      Promise.all([
        apiGet<ServicesResponse>("/internal/admin/services"),
        apiGet<TerminalPathsSettings>("/internal/admin/settings/terminal-paths"),
        apiGet<SSHHostsResponse>("/internal/admin/ssh/hosts"),
        apiGet<SSHBindingsResponse>("/internal/admin/ssh/bindings"),
      ])
        .then(([servicesResponse, pathResponse, sshHostsResponse, sshBindingsResponse]) => {
          if (!active) {
            return;
          }
          setServices(servicesResponse);
          setTerminalPaths(pathResponse);
          setSshHosts(sshHostsResponse);
          setSshBindings(sshBindingsResponse);
        })
        .catch(() => {});
    }

    if (route === "/admin/mcp") {
      Promise.all([
        apiGet<MCPServersResponse>("/internal/admin/mcp/servers"),
        apiGet<MCPOpenWebUIPreviewResponse>("/internal/admin/mcp/openwebui-preview"),
      ])
        .then(([serversResponse, previewResponse]) => {
          if (!active) {
            return;
          }
          setMcpServers(serversResponse);
          setMcpPreview(previewResponse);
        })
        .catch(() => {});
    }

    return () => {
      active = false;
    };
  }, [route]);

  const activeProfile = useMemo(() => {
    if (!models) {
      return null;
    }
    return (
      models.profiles.find((profile) => profile.id === models.active_profile_id) ??
      models.profiles.find((profile) => profile.enabled) ??
      models.profiles[0] ??
      null
    );
  }, [models]);

  const onlineServices = countOnlineServices(overview?.services ?? []);
  const latestFailure = (overview?.recent_failures ?? [])[0] ?? null;
  const quickJump = quickJumpForRoute(route);
  const toolRouterBaseUrl =
    services?.services.find((item) => item.name === "tool-router")?.address?.replace(/\/healthz$/i, "") ??
    "http://127.0.0.1:8091";

  return (
    <aside className="hidden h-full w-[22rem] min-w-0 overflow-x-hidden bg-[var(--panel-muted)] px-5 py-5 shadow-[inset_1px_0_0_rgba(255,255,255,0.55)] xl:flex xl:flex-col">
      <div className="min-w-0 pb-5">
        <div className="text-[11px] font-semibold uppercase tracking-[0.26em] text-slate-400">
          Utility Rail
        </div>
        <div className="mt-2 text-base font-semibold text-slate-950">{title}</div>
        <p className="mt-1 break-words text-sm leading-6 text-slate-500">{subtitle}</p>
      </div>

      <div className="admin-divider" />

      <div className="admin-scrollbar-hidden min-w-0 flex-1 overflow-auto overflow-x-hidden pt-5">
        <section className="admin-rail-summary min-w-0">
          <div className="admin-rail-summary-grid">
            <SummaryBlock label="Mode" value="awdp" />
            <SummaryBlock
              label="Model"
              value={overview?.active_model.label ?? overview?.active_model.profile_id ?? "Unknown"}
            />
            <SummaryBlock
              label="Online"
              value={`${onlineServices}/${(overview?.services ?? []).length || 4}`}
            />
          </div>
          <div className="mt-4 grid grid-cols-3 gap-3">
            <ServiceMark name="Gateway" status={findServiceStatus(overview?.services, "gateway")} />
            <ServiceMark name="Router" status={findServiceStatus(overview?.services, "tool-router")} />
            <ServiceMark name="WebUI" status={findServiceStatus(overview?.services, "open-webui")} />
          </div>
        </section>

        {route === "/admin" ? (
          <>
            <RailSection title="System Summary">
              <MetricList
                items={[
                  { label: "Profiles", value: String(overview?.available_profiles.length ?? 0) },
                  { label: "Recent Failure", value: latestFailure ? formatTime(latestFailure.time) : "None" },
                  {
                    label: "Active PID",
                    value: overview?.active_model.pid ? String(overview.active_model.pid) : "Stopped",
                  },
                ]}
              />
            </RailSection>

            <RailSection title="Recent Exceptions" footer={<QuickJumpLink to={quickJump.to} label={quickJump.label} />}>
              {(overview?.recent_failures ?? []).length > 0 ? (
                <PlainList
                  items={(overview?.recent_failures ?? []).slice(0, 3).map((entry, index) => ({
                    key: `failure-${index}`,
                    eyebrow: formatTime(entry.time),
                    title: summarizeLogEntry(entry),
                  }))}
                />
              ) : (
                <EmptyHint text="No recent failures in the overview feed." />
              )}
            </RailSection>
          </>
        ) : null}

        {route === "/admin/models" ? (
          <>
            <RailSection title="Run Budget">
              <MetricList
                items={[
                  { label: "Profile", value: activeProfile?.label ?? "Unknown" },
                  { label: "Total Context", value: formatTokens(activeProfile?.ctx_size) },
                  {
                    label: "Per Slot",
                    value: formatTokens(perSlotContext(activeProfile?.ctx_size, activeProfile?.parallel)),
                  },
                  { label: "Parallel", value: String(activeProfile?.parallel ?? 0) },
                  { label: "Threads", value: String(activeProfile?.threads ?? 0) },
                  { label: "GPU Layers", value: String(activeProfile?.n_gpu_layers ?? 0) },
                ]}
              />
            </RailSection>

            <RailSection title="Capacity Notes" footer={<QuickJumpLink to={quickJump.to} label={quickJump.label} />}>
              <NarrativeList
                items={[
                  {
                    title: "Per-user context",
                    body: `${formatTokens(perSlotContext(activeProfile?.ctx_size, activeProfile?.parallel))} each when all slots are in use.`,
                  },
                  {
                    title: "Usage profile",
                    body: describeParallelProfile(activeProfile?.parallel ?? 0),
                  },
                  {
                    title: "Apply rule",
                    body: "Context pool, parallel slots, and runtime flags take full effect after a llama restart.",
                  },
                ]}
              />
            </RailSection>
          </>
        ) : null}

        {route === "/admin/modes" ? (
          <>
            <RailSection title="Mode Relationship">
              <PlainList
                items={[
                  {
                    key: modes?.core.name ?? "core",
                    eyebrow: "core mode",
                    title: `${modes?.core.name ?? "awdp"} · ${modes?.core.tool_scope.length ?? 0} tools`,
                  },
                  ...(modes?.plugins ?? []).map((plugin) => ({
                    key: plugin.name,
                    eyebrow: "plugin",
                    title: `${plugin.name} · ${plugin.tool_scope.length} tools`,
                  })),
                ]}
                emptyText="Mode metadata will appear after the route loads."
              />
            </RailSection>

            <RailSection title="Current Rules" footer={<QuickJumpLink to={quickJump.to} label={quickJump.label} />}>
              <MetricList
                items={[
                  { label: "Core Prompts", value: String(modes?.core.prompt_files.length ?? 0) },
                  { label: "Retrieval Roots", value: String(modes?.core.retrieval_roots.length ?? 0) },
                  { label: "Plugin Count", value: String(modes?.plugins.length ?? 0) },
                ]}
              />
              <InlineTagRow items={(modes?.core.tool_scope ?? []).slice(0, 5)} />
            </RailSection>
          </>
        ) : null}

        {route === "/admin/logs" ? (
          <>
            <RailSection title="Quick Filters">
              <FilterRow
                label="Source"
                options={[
                  { label: "Sessions", value: "sessions" },
                  { label: "Tools", value: "tools" },
                ]}
                activeValue={logSource}
                onSelect={(value) => setLogSource(value as "sessions" | "tools")}
              />
              <FilterRow
                label="Range"
                options={[
                  { label: "1h", value: "1h" },
                  { label: "24h", value: "24h" },
                  { label: "7d", value: "7d" },
                  { label: "All", value: "all" },
                ]}
                activeValue={logRange}
                onSelect={(value) => setLogRange(value as "1h" | "24h" | "7d" | "all")}
              />
              <button
                type="button"
                className={`admin-filter-link ${logFailureOnly ? "active" : ""}`}
                onClick={() => setLogFailureOnly(!logFailureOnly)}
              >
                Failure first
              </button>
            </RailSection>

            <RailSection title="Selected Preview" footer={<QuickJumpLink to={quickJump.to} label={quickJump.label} />}>
              {selectedLog ? (
                <>
                  <MetricList
                    items={[
                      {
                        label: "Type",
                        value: selectedLog.source === "sessions" ? "Session log" : "Tool log",
                      },
                      { label: "Time", value: formatTime(selectedLog.item.time) },
                      { label: "Status", value: detectLogTone(selectedLog.item) },
                    ]}
                  />
                  <p className="mt-4 text-sm leading-7 text-slate-700">
                    {summarizeLogEntry(selectedLog.item)}
                  </p>
                </>
              ) : (
                <EmptyHint text="Pick a log entry in the main list to inspect it here." />
              )}
            </RailSection>
          </>
        ) : null}

        {route === "/admin/mcp" ? (
          <>
            <RailSection title="MCP Summary">
              <MetricList
                items={[
                  {
                    label: "Enabled",
                    value: String(
                      (mcpServers?.servers ?? []).filter((item) => item.profile.enabled).length,
                    ),
                  },
                  {
                    label: "Native",
                    value: String(
                      (mcpServers?.servers ?? []).filter(
                        (item) => item.profile.kind === "native_streamable_http",
                      ).length,
                    ),
                  },
                  {
                    label: "Bridged",
                    value: String(
                      (mcpServers?.servers ?? []).filter(
                        (item) => item.profile.kind !== "native_streamable_http",
                      ).length,
                    ),
                  },
                  {
                    label: "Preview",
                    value: String(mcpPreview?.connections.length ?? 0),
                  },
                ]}
              />
            </RailSection>

            <RailSection title="Latest Runtime" footer={<QuickJumpLink to={quickJump.to} label={quickJump.label} />}>
              {(mcpServers?.servers ?? []).length > 0 ? (
                <PlainList
                  items={(mcpServers?.servers ?? []).slice(0, 4).map((item) => ({
                    key: item.profile.id,
                    eyebrow: item.runtime_status.status,
                    title: `${item.profile.label} · ${runtimeCompact(item.runtime_status)}`,
                  }))}
                />
              ) : (
                <EmptyHint text="No MCP servers configured yet." />
              )}
            </RailSection>
          </>
        ) : null}

        {route === "/admin/settings" ? (
          <>
            <RailSection title="Effect Window">
              <PlainList
                items={[
                  { key: "ssh-save", eyebrow: "Instant", title: "SSH host save" },
                  { key: "binding-save", eyebrow: "Instant", title: "User binding save" },
                  { key: "paths", eyebrow: "Restart router", title: "Allowed paths" },
                  { key: "webui", eyebrow: "Restart WebUI", title: "WebUI terminal patch" },
                ]}
              />
            </RailSection>

            <RailSection title="Current Key Paths" footer={<QuickJumpLink to={quickJump.to} label={quickJump.label} />}>
              <MetricList
                items={[
                  { label: "OpenAPI", value: `${toolRouterBaseUrl}/openapi.json` },
                  { label: "Allowed Roots", value: String(terminalPaths?.allowed_paths.length ?? 0) },
                  { label: "SSH Hosts", value: String(sshHosts?.hosts.length ?? 0) },
                  { label: "User Bindings", value: String(sshBindings?.bindings.length ?? 0) },
                ]}
              />
              <a
                className="admin-inline-link mt-4"
                href={`${toolRouterBaseUrl}/openapi.json`}
                target="_blank"
                rel="noreferrer"
              >
                Open OpenAPI spec
                <ExternalLink className="h-3.5 w-3.5" />
              </a>
            </RailSection>
          </>
        ) : null}
      </div>
    </aside>
  );
}

function RailSection({
  title,
  children,
  footer,
}: {
  title: string;
  children: React.ReactNode;
  footer?: React.ReactNode;
}) {
  return (
    <section className="admin-rail-section">
      <div className="min-w-0 flex items-center justify-between gap-3">
        <h3 className="text-sm font-semibold text-slate-950">{title}</h3>
      </div>
      <div className="mt-4 min-w-0">{children}</div>
      {footer ? <div className="mt-5">{footer}</div> : null}
    </section>
  );
}

function SummaryBlock({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0">
      <div className="text-[11px] font-semibold uppercase tracking-[0.18em] text-slate-400">
        {label}
      </div>
      <div className="mt-2 break-words text-sm font-medium leading-6 text-slate-900">{value}</div>
    </div>
  );
}

function ServiceMark({ name, status }: { name: string; status: string }) {
  return (
    <div className="flex items-center gap-2 text-sm text-slate-700">
      <span className={`admin-service-dot ${status === "ok" ? "ok" : status === "unknown" ? "muted" : "down"}`} />
      <span>{name}</span>
    </div>
  );
}

function MetricList({ items }: { items: Array<{ label: string; value: string }> }) {
  return (
    <div className="space-y-3">
      {items.map((item) => (
        <div key={item.label} className="flex min-w-0 items-start justify-between gap-4">
          <span className="shrink-0 text-xs uppercase tracking-[0.16em] text-slate-400">{item.label}</span>
          <span className="min-w-0 max-w-[10.25rem] break-all text-right text-sm leading-6 text-slate-800">{item.value}</span>
        </div>
      ))}
    </div>
  );
}

function PlainList({
  items,
  emptyText,
}: {
  items: Array<{ key: string; eyebrow: string; title: string }>;
  emptyText?: string;
}) {
  if (items.length === 0) {
    return <EmptyHint text={emptyText ?? "No items available."} />;
  }

  return (
    <div className="space-y-4">
      {items.map((item, index) => (
        <div key={item.key} className={index > 0 ? "admin-rail-row admin-rail-row-bordered min-w-0" : "admin-rail-row min-w-0"}>
          <div className="text-[11px] font-semibold uppercase tracking-[0.18em] text-slate-400">
            {item.eyebrow}
          </div>
          <div className="mt-1 break-words text-sm leading-6 text-slate-800">{item.title}</div>
        </div>
      ))}
    </div>
  );
}

function NarrativeList({ items }: { items: Array<{ title: string; body: string }> }) {
  return (
    <div className="space-y-4">
      {items.map((item, index) => (
        <div key={item.title} className={index > 0 ? "admin-rail-row admin-rail-row-bordered min-w-0" : "admin-rail-row min-w-0"}>
          <div className="text-[11px] font-semibold uppercase tracking-[0.18em] text-slate-400">
            {item.title}
          </div>
          <p className="mt-1 break-words text-sm leading-7 text-slate-700">{item.body}</p>
        </div>
      ))}
    </div>
  );
}

function InlineTagRow({ items }: { items: string[] }) {
  if (items.length === 0) {
    return null;
  }

  return (
    <div className="mt-4 flex min-w-0 flex-wrap gap-x-3 gap-y-2 text-sm text-slate-600">
      {items.map((item) => (
        <span key={item} className="admin-inline-tag">
          {item}
        </span>
      ))}
    </div>
  );
}

function FilterRow({
  label,
  options,
  activeValue,
  onSelect,
}: {
  label: string;
  options: Array<{ label: string; value: string }>;
  activeValue: string;
  onSelect: (value: string) => void;
}) {
  return (
    <div className="mb-4">
      <div className="mb-2 text-[11px] font-semibold uppercase tracking-[0.18em] text-slate-400">
        {label}
      </div>
      <div className="flex min-w-0 flex-wrap gap-x-3 gap-y-2">
        {options.map((option) => (
          <button
            key={option.value}
            type="button"
            className={`admin-filter-link ${activeValue === option.value ? "active" : ""}`}
            onClick={() => onSelect(option.value)}
          >
            {option.label}
          </button>
        ))}
      </div>
    </div>
  );
}

function QuickJumpLink({ to, label }: { to: string; label: string }) {
  return (
    <Link to={to} className="admin-inline-link">
      {label}
      <ArrowRight className="h-3.5 w-3.5" />
    </Link>
  );
}

function EmptyHint({ text }: { text: string }) {
  return <p className="text-sm leading-7 text-slate-500">{text}</p>;
}

function countOnlineServices(services: Array<{ status: string }>) {
  return services.filter((service) => service.status === "ok").length;
}

function findServiceStatus(services: OverviewResponse["services"] | undefined, name: string) {
  return services?.find((service) => service.name === name)?.status ?? "unknown";
}

function summarizeLogEntry(item: Record<string, unknown>) {
  const candidates = [item.summary, item.reason, item.message, item.tool, item.session_id, item.request_id];
  const hit = candidates.find((value) => typeof value === "string" && value.trim().length > 0);
  return typeof hit === "string" ? hit : "No summary available";
}

function detectLogTone(item: Record<string, unknown>) {
  const text = JSON.stringify(item).toLowerCase();
  return text.includes("error") || text.includes("fail") ? "attention" : "normal";
}

function perSlotContext(ctxSize?: number, parallel?: number) {
  if (!ctxSize || !parallel) {
    return 0;
  }
  return Math.round(ctxSize / Math.max(1, parallel));
}

function formatTokens(value?: number) {
  if (!value) {
    return "0";
  }
  if (value >= 1024) {
    return `${Math.round(value / 1024)}k`;
  }
  return String(value);
}

function describeParallelProfile(parallel: number) {
  if (parallel <= 2) {
    return "Focused setup for longer context and heavier single-user work.";
  }
  if (parallel <= 4) {
    return "Balanced setup for small team use with decent per-user context.";
  }
  return "Throughput-first setup that favors more slots over deeper per-user context.";
}

function runtimeCompact(entry: MCPRuntimeEntry) {
  if (entry.last_error) {
    return entry.last_error;
  }
  if (entry.bridge_port) {
    return `${entry.kind} @ ${entry.bridge_port}`;
  }
  return entry.kind;
}

function quickJumpForRoute(route: DrawerRoute) {
  switch (route) {
    case "/admin/models":
      return { to: "/admin/settings", label: "Jump to Settings" };
    case "/admin/modes":
      return { to: "/admin", label: "Jump to Dashboard" };
    case "/admin/mcp":
      return { to: "/admin/settings", label: "Jump to Settings" };
    case "/admin/logs":
      return { to: "/admin", label: "Jump to Dashboard" };
    case "/admin/settings":
      return { to: "/admin/models", label: "Jump to Models" };
    default:
      return { to: "/admin/models", label: "Jump to Models" };
  }
}
