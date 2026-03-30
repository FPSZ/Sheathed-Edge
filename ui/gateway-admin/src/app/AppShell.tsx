import { useEffect, useState } from "react";
import { Outlet, useLocation } from "react-router-dom";

import {
  AdminScopeProvider,
  type LogRange,
  type LogSource,
  type SelectedLog,
} from "./AdminScopeContext";
import { DetailsDrawer } from "../components/DetailsDrawer";
import { Sidebar } from "../components/Sidebar";
import { TopBar } from "../components/TopBar";
import { apiGet } from "../lib/api";
import type { UsersResponse } from "../lib/types";

const pageMeta: Record<string, { title: string; subtitle: string }> = {
  "/admin": {
    title: "总览",
    subtitle: "系统概况",
  },
  "/admin/models": {
    title: "模型",
    subtitle: "Llama 运行控制",
  },
  "/admin/modes": {
    title: "模式",
    subtitle: "AWDP 能力布局",
  },
  "/admin/mcp": {
    title: "MCP 服务",
    subtitle: "外部工具控制面",
  },
  "/admin/logs": {
    title: "日志",
    subtitle: "会话与工具审计",
  },
  "/admin/settings": {
    title: "设置",
    subtitle: "用户工作区与运行配置",
  },
};

export function AppShell() {
  const location = useLocation();
  const meta = pageMeta[location.pathname] ?? pageMeta["/admin"];
  const [logSource, setLogSource] = useState<LogSource>("sessions");
  const [logRange, setLogRange] = useState<LogRange>("24h");
  const [logFailureOnly, setLogFailureOnly] = useState(false);
  const [selectedLog, setSelectedLog] = useState<SelectedLog>(null);
  const [users, setUsers] = useState<UsersResponse["users"]>([]);
  const [usersLoading, setUsersLoading] = useState(false);
  const [usersError, setUsersError] = useState("");
  const [usersConfigPath, setUsersConfigPath] = useState("");
  const [selectedUserEmail, setSelectedUserEmail] = useState("");

  useEffect(() => {
    if (location.pathname !== "/admin/logs") {
      setSelectedLog(null);
    }
  }, [location.pathname]);

  async function refreshUsers() {
    setUsersLoading(true);
    try {
      const response = await apiGet<UsersResponse>("/internal/admin/users");
      setUsers(response.users);
      setUsersConfigPath(response.config_path);
      setUsersError("");
      setSelectedUserEmail((current) => {
        if (!current) {
          return "";
        }
        return response.users.some((item) => item.user_email === current) ? current : "";
      });
    } catch (err) {
      setUsersError((err as Error).message);
    } finally {
      setUsersLoading(false);
    }
  }

  useEffect(() => {
    refreshUsers();
  }, []);

  return (
    <AdminScopeProvider
      value={{
        users,
        usersLoading,
        usersError,
        usersConfigPath,
        refreshUsers,
        selectedUserEmail,
        setSelectedUserEmail,
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
    </AdminScopeProvider>
  );
}
