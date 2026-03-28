import { Outlet, useLocation } from "react-router-dom";

import { DetailsDrawer } from "../components/DetailsDrawer";
import { Sidebar } from "../components/Sidebar";
import { TopBar } from "../components/TopBar";

const pageMeta: Record<string, { title: string; subtitle: string; drawer: string }> = {
  "/admin": {
    title: "Dashboard",
    subtitle: "System overview",
    drawer: "总览页强调系统是否可用、当前模型状态、最近日志与失败摘要。",
  },
  "/admin/models": {
    title: "Models",
    subtitle: "Llama runtime control",
    drawer: "模型页优先展示当前活动 profile、启动状态与启停切换操作。",
  },
  "/admin/modes": {
    title: "Modes",
    subtitle: "AWDP capability layout",
    drawer: "模式页只做查看，不在线编辑 prompt、plugin 或 registry 配置。",
  },
  "/admin/logs": {
    title: "Logs",
    subtitle: "Session and tool audit",
    drawer: "日志页按会话日志与工具日志分栏，右侧抽屉用于查看失败细节。",
  },
  "/admin/settings": {
    title: "Settings",
    subtitle: "Read-only runtime config",
    drawer: "设置页当前只读，用来展示网关、Host Agent、Open WebUI 等地址与当前 profile 信息。",
  },
};

export function AppShell() {
  const location = useLocation();
  const meta = pageMeta[location.pathname] ?? pageMeta["/admin"];

  return (
    <div className="flex h-screen overflow-hidden bg-[var(--app-bg)] text-slate-900">
      <Sidebar />
      <div className="flex min-w-0 flex-1 flex-col">
        <TopBar title={meta.title} subtitle={meta.subtitle} />
        <div className="flex min-h-0 flex-1">
          <main className="min-w-0 flex-1 overflow-auto px-6 py-6">
            <Outlet />
          </main>
          <DetailsDrawer
            title={meta.title}
            description="设计语言参考 Memori-Vault，信息架构参考 AstrBot。"
            content={meta.drawer}
          />
        </div>
      </div>
    </div>
  );
}
