import { useEffect, useState } from "react";
import { Outlet, useLocation } from "react-router-dom";

import {
  AdminPageStateProvider,
  type LogRange,
  type LogSource,
  type SelectedLog,
} from "./AdminPageState";
import { DetailsDrawer } from "../components/DetailsDrawer";
import { Sidebar } from "../components/Sidebar";
import { TopBar } from "../components/TopBar";

const pageMeta: Record<string, { title: string; subtitle: string }> = {
  "/admin": {
    title: "Dashboard",
    subtitle: "System overview",
  },
  "/admin/models": {
    title: "Models",
    subtitle: "Llama runtime control",
  },
  "/admin/modes": {
    title: "Modes",
    subtitle: "AWDP capability layout",
  },
  "/admin/mcp": {
    title: "MCP Servers",
    subtitle: "External tool control plane",
  },
  "/admin/logs": {
    title: "Logs",
    subtitle: "Session and tool audit",
  },
  "/admin/settings": {
    title: "Settings",
    subtitle: "Runtime and terminal config",
  },
};

export function AppShell() {
  const location = useLocation();
  const meta = pageMeta[location.pathname] ?? pageMeta["/admin"];
  const [logSource, setLogSource] = useState<LogSource>("sessions");
  const [logRange, setLogRange] = useState<LogRange>("24h");
  const [logFailureOnly, setLogFailureOnly] = useState(false);
  const [selectedLog, setSelectedLog] = useState<SelectedLog>(null);

  useEffect(() => {
    if (location.pathname !== "/admin/logs") {
      setSelectedLog(null);
    }
  }, [location.pathname]);

  return (
    <AdminPageStateProvider
      value={{
        logSource,
        setLogSource,
        logRange,
        setLogRange,
        logFailureOnly,
        setLogFailureOnly,
        selectedLog,
        setSelectedLog,
      }}
    >
      <div className="flex h-screen overflow-hidden bg-[var(--app-bg)] text-slate-900">
        <Sidebar />
        <div className="flex min-w-0 flex-1 flex-col">
          <TopBar title={meta.title} subtitle={meta.subtitle} />
          <div className="flex min-h-0 flex-1">
            <main className="admin-scrollbar-hidden min-w-0 flex-1 overflow-auto px-6 py-6">
              <Outlet />
            </main>
            <DetailsDrawer title={meta.title} subtitle={meta.subtitle} />
          </div>
        </div>
      </div>
    </AdminPageStateProvider>
  );
}
