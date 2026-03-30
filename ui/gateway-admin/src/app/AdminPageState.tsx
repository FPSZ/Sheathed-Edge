import { createContext, useContext, type ReactNode } from "react";

export type LogSource = "sessions" | "tools";
export type LogRange = "1h" | "24h" | "7d" | "all";

export type SelectedLog = {
  source: LogSource;
  item: Record<string, unknown>;
} | null;

export type AdminPageStateValue = {
  logSource: LogSource;
  setLogSource: (value: LogSource) => void;
  logRange: LogRange;
  setLogRange: (value: LogRange) => void;
  logFailureOnly: boolean;
  setLogFailureOnly: (value: boolean) => void;
  selectedLog: SelectedLog;
  setSelectedLog: (value: SelectedLog) => void;
};

const AdminPageStateContext = createContext<AdminPageStateValue | null>(null);

export function AdminPageStateProvider({
  value,
  children,
}: {
  value: AdminPageStateValue;
  children: ReactNode;
}) {
  return <AdminPageStateContext.Provider value={value}>{children}</AdminPageStateContext.Provider>;
}

export function useAdminPageState() {
  const value = useContext(AdminPageStateContext);
  if (!value) {
    throw new Error("useAdminPageState must be used inside AdminPageStateProvider");
  }
  return value;
}
