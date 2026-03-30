import { createContext, useContext, type ReactNode } from "react";

import type { UserSummary } from "../lib/types";

export type LogSource = "sessions" | "tools";
export type LogRange = "1h" | "24h" | "7d" | "all";

export type SelectedLog = {
  source: LogSource;
  item: Record<string, unknown>;
} | null;

export type AdminScopeValue = {
  users: UserSummary[];
  usersLoading: boolean;
  usersError: string;
  usersConfigPath: string;
  refreshUsers: () => Promise<void>;
  selectedUserEmail: string;
  setSelectedUserEmail: (value: string) => void;
  logSource: LogSource;
  setLogSource: (value: LogSource) => void;
  logRange: LogRange;
  setLogRange: (value: LogRange) => void;
  logFailureOnly: boolean;
  setLogFailureOnly: (value: boolean) => void;
  selectedLog: SelectedLog;
  setSelectedLog: (value: SelectedLog) => void;
};

const AdminScopeContext = createContext<AdminScopeValue | null>(null);

export function AdminScopeProvider({
  value,
  children,
}: {
  value: AdminScopeValue;
  children: ReactNode;
}) {
  return <AdminScopeContext.Provider value={value}>{children}</AdminScopeContext.Provider>;
}

export function useAdminScope() {
  const value = useContext(AdminScopeContext);
  if (!value) {
    throw new Error("useAdminScope must be used inside AdminScopeProvider");
  }
  return value;
}
