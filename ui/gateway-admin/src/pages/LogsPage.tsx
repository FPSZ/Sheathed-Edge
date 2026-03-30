import { useEffect, useMemo, useState } from "react";

import { useAdminPageState } from "../app/AdminPageState";
import { PageHeader } from "../components/PageHeader";
import { apiGet } from "../lib/api";
import { formatJson, formatTime } from "../lib/format";
import type { LogListResponse } from "../lib/types";

export function LogsPage() {
  const [sessions, setSessions] = useState<Record<string, unknown>[]>([]);
  const [tools, setTools] = useState<Record<string, unknown>[]>([]);
  const [error, setError] = useState("");
  const {
    logSource,
    logRange,
    logFailureOnly,
    selectedLog,
    setSelectedLog,
  } = useAdminPageState();

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

  const activeItems = useMemo(() => {
    const items = logSource === "sessions" ? sessions : tools;
    return items.filter((item) => {
      if (logFailureOnly && !isFailureLike(item)) {
        return false;
      }
      return isWithinRange(item.time, logRange);
    });
  }, [logFailureOnly, logRange, logSource, sessions, tools]);

  useEffect(() => {
    if (activeItems.length === 0) {
      setSelectedLog(null);
      return;
    }

    if (
      !selectedLog ||
      selectedLog.source !== logSource ||
      !activeItems.includes(selectedLog.item)
    ) {
      setSelectedLog({ source: logSource, item: activeItems[0] });
    }
  }, [activeItems, logSource, selectedLog, setSelectedLog]);

  return (
    <div className="space-y-6">
      <PageHeader
        title="Logs"
        description="Browse session and tool logs with a focused list on the left and a detail preview in the utility rail."
      />

      {error ? (
        <div className="admin-surface rounded-3xl bg-rose-50 p-4 text-sm text-rose-700">
          {error}
        </div>
      ) : null}

      <section className="admin-surface rounded-3xl p-5">
        <div className="flex items-start justify-between gap-4">
          <div>
            <div className="text-sm font-semibold text-slate-900">
              {logSource === "sessions" ? "Session log stream" : "Tool log stream"}
            </div>
            <div className="mt-1 text-xs text-slate-500">
              Showing {activeItems.length} items in the current filter window.
            </div>
          </div>
          <div className="admin-badge muted">
            {logFailureOnly ? "Failure filter on" : "All severities"}
          </div>
        </div>

        <div className="mt-4 space-y-3">
          {activeItems.map((item, index) => {
            const isSelected = selectedLog?.item === item;
            return (
              <button
                key={buildLogKey(item, index)}
                type="button"
                className={`admin-log-item ${isSelected ? "active" : ""}`}
                onClick={() => setSelectedLog({ source: logSource, item })}
              >
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <div className="text-xs uppercase tracking-[0.16em] text-slate-400">
                      {formatTime(item.time)}
                    </div>
                    <div className="mt-1 text-sm font-medium text-slate-900">
                      {summarizeLogEntry(item)}
                    </div>
                  </div>
                  <span className={`admin-badge ${isFailureLike(item) ? "danger" : "muted"}`}>
                    {isFailureLike(item) ? "Attention" : "Normal"}
                  </span>
                </div>
                <pre className="mt-3 whitespace-pre-wrap break-words text-xs leading-6 text-slate-600">
                  {formatJson(item)}
                </pre>
              </button>
            );
          })}

          {activeItems.length === 0 ? (
            <div className="admin-surface-muted rounded-3xl p-4 text-sm text-slate-500">
              No log items match the current source and filter range.
            </div>
          ) : null}
        </div>
      </section>
    </div>
  );
}

function summarizeLogEntry(item: Record<string, unknown>) {
  const candidates = [item.summary, item.reason, item.message, item.tool, item.session_id];
  const hit = candidates.find((value) => typeof value === "string" && value.trim().length > 0);
  return typeof hit === "string" ? hit : "No summary available";
}

function isFailureLike(item: Record<string, unknown>) {
  const raw = JSON.stringify(item).toLowerCase();
  return raw.includes("error") || raw.includes("fail");
}

function isWithinRange(value: unknown, range: "1h" | "24h" | "7d" | "all") {
  if (range === "all") {
    return true;
  }
  const parsed = typeof value === "string" || typeof value === "number" ? Date.parse(String(value)) : NaN;
  if (Number.isNaN(parsed)) {
    return true;
  }
  const deltaMs = Date.now() - parsed;
  const limits = {
    "1h": 60 * 60 * 1000,
    "24h": 24 * 60 * 60 * 1000,
    "7d": 7 * 24 * 60 * 60 * 1000,
  };
  return deltaMs <= limits[range];
}

function buildLogKey(item: Record<string, unknown>, index: number) {
  const time = item.time;
  const summary = item.summary ?? item.reason ?? item.message ?? item.tool ?? item.session_id;
  return `${String(time ?? "no-time")}-${String(summary ?? index)}`;
}
