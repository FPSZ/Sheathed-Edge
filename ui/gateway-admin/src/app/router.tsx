import { createBrowserRouter } from "react-router-dom";

import { AppShell } from "./AppShell";
import { DashboardPage } from "../pages/DashboardPage";
import { LogsPage } from "../pages/LogsPage";
import { MCPPage } from "../pages/MCPPage";
import { ModesPage } from "../pages/ModesPage";
import { ModelsPage } from "../pages/ModelsPage";
import { SettingsPage } from "../pages/SettingsPage";

export const router = createBrowserRouter(
  [
    {
      path: "/admin",
      element: <AppShell />,
      children: [
        { index: true, element: <DashboardPage /> },
        { path: "models", element: <ModelsPage /> },
        { path: "modes", element: <ModesPage /> },
        { path: "mcp", element: <MCPPage /> },
        { path: "logs", element: <LogsPage /> },
        { path: "settings", element: <SettingsPage /> },
      ],
    },
  ],
  {
    basename: "/",
  },
);
