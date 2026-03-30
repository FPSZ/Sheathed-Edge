import { Activity, Bot, LayoutDashboard, Logs, PlugZap, Settings } from "lucide-react";
import { NavLink } from "react-router-dom";

const items = [
  { to: "/admin", label: "Dashboard", icon: LayoutDashboard, end: true },
  { to: "/admin/models", label: "Models", icon: Bot },
  { to: "/admin/modes", label: "Modes", icon: Activity },
  { to: "/admin/mcp", label: "MCP", icon: PlugZap },
  { to: "/admin/logs", label: "Logs", icon: Logs },
  { to: "/admin/settings", label: "Settings", icon: Settings },
];

export function Sidebar() {
  return (
    <aside className="flex h-full w-64 flex-col bg-[var(--panel-muted)] px-4 py-5 shadow-[inset_-1px_0_0_rgba(255,255,255,0.55)]">
      <div className="mb-8 px-2">
        <div className="text-xs uppercase tracking-[0.24em] text-slate-400">Sheathed Edge</div>
        <div className="mt-2 text-lg font-semibold text-slate-950">Gateway Admin</div>
      </div>
      <nav className="space-y-1">
        {items.map((item) => {
          const Icon = item.icon;
          return (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.end}
              className={({ isActive }) =>
                [
                  "flex items-center gap-3 rounded-2xl px-3 py-2.5 text-sm transition-all",
                  isActive
                    ? "bg-white text-slate-950 shadow-[0_1px_2px_rgba(15,23,42,0.04),0_10px_24px_rgba(15,23,42,0.06)]"
                    : "text-slate-600 hover:bg-white/70 hover:text-slate-900 hover:shadow-[0_1px_2px_rgba(15,23,42,0.03),0_6px_16px_rgba(15,23,42,0.04)]",
                ].join(" ")
              }
            >
              <Icon className="h-4 w-4" />
              <span>{item.label}</span>
            </NavLink>
          );
        })}
      </nav>
    </aside>
  );
}
