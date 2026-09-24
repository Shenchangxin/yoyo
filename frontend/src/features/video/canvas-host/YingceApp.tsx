import { lazy, Suspense, useEffect, useLayoutEffect, useMemo, type ReactNode } from "react";
import { createMemoryRouter, Outlet, RouterProvider, useLocation, useParams } from "react-router";
import { AppProviders } from "@yingce/components/layout/app-providers";
import { RequireAuth } from "@yingce/components/auth/require-auth";
import { CanvasRefreshShell } from "@yingce/pages/canvas/canvas-refresh-shell";
import RouteErrorPage from "@yingce/pages/route-error";
import UserLayout from "@yingce/layouts/user-layout";
import { useCanvasStore } from "@yingce/stores/canvas/use-canvas-store";
import { useCanvasThemeStore } from "@yingce/stores/canvas/use-canvas-theme-store";
import { bootstrapAppearance } from "@yingce/services/appearance-bootstrap";
import "@yingce/lib/plugins/builtin";
import { bindCanvasSession, setCanvasHostTitle } from "./session";
import { installCanvasHub } from "./hub";
import { primeCanvasMediaBase } from "./media";
import "@yingce/styles/globals.css";
import "@yingce/styles/shared/model-picker.css";
import "@yingce/styles/shared/overlays.css";
import "@yingce/styles/shared/scrollbars.css";

const CanvasPage = lazy(() => import("@yingce/pages/canvas"));
const CanvasProjectPage = lazy(() => import("@yingce/pages/canvas/project"));
const CreatePage = lazy(() => import("@yingce/pages/create"));
const ProjectsPage = lazy(() => import("@yingce/pages/projects"));
const ProjectDetailPage = lazy(() => import("@yingce/pages/projects/detail"));
const AssetsPage = lazy(() => import("@yingce/pages/assets"));
const TasksPage = lazy(() => import("@yingce/pages/tasks"));
const PluginsPage = lazy(() => import("@yingce/pages/plugins"));
const EaglePage = lazy(() => import("@yingce/pages/plugins/eagle"));
const SettingsPage = lazy(() => import("@yingce/pages/settings"));

if (typeof window !== "undefined") {
  (window as Window & { __YOYO_CANVAS_HOST__?: boolean }).__YOYO_CANVAS_HOST__ = true;
}

function deferred(node: ReactNode, fallback?: ReactNode) {
  return <Suspense fallback={fallback || <CanvasRefreshShell />}>{node}</Suspense>;
}

function BindAndTitle() {
  const params = useParams();
  const location = useLocation();
  const sessionId = typeof window !== "undefined" ? String((window as Window & { __YOYO_VIDEO_SESSION__?: string }).__YOYO_VIDEO_SESSION__ || "") : "";
  const projects = useCanvasStore((s) => s.projects);
  const id = params.id || params.projectId || "";

  useEffect(() => {
    if (id && sessionId) void bindCanvasSession(sessionId, id);
  }, [id, sessionId]);

  useEffect(() => {
    const match = projects.find((p) => p.id === id);
    const title = match?.title || (location.pathname.startsWith("/projects") ? "Chapter workflow" : location.pathname.startsWith("/assets") ? "Assets" : location.pathname.startsWith("/plugins") ? "Plugins" : location.pathname.startsWith("/tasks") ? "Tasks" : "Infinite canvas");
    setCanvasHostTitle(title);
  }, [id, location.pathname, projects]);

  return null;
}

function WorkspaceLayout() {
  return (
    <RequireAuth>
      <UserLayout>
        <BindAndTitle />
        <Outlet />
      </UserLayout>
    </RequireAuth>
  );
}

export default function YingceApp() {
  useLayoutEffect(() => {
    void bootstrapAppearance();
  }, []);

  useEffect(() => {
    const dark = document.documentElement.classList.contains("dark");
    useCanvasThemeStore.getState().setTheme(dark ? "dark" : "light");
    void primeCanvasMediaBase();
    return installCanvasHub();
  }, []);

  const router = useMemo(
    () =>
      createMemoryRouter(
        [
          {
            element: <WorkspaceLayout />,
            errorElement: <RouteErrorPage />,
            children: [
              { path: "/", element: deferred(<CreatePage />) },
              { path: "/create", element: deferred(<CreatePage />) },
              { path: "/canvas", element: deferred(<CanvasPage />) },
              { path: "/canvas/:id", element: deferred(<CanvasProjectPage />, <CanvasRefreshShell />) },
              { path: "/projects", element: deferred(<ProjectsPage />) },
              { path: "/projects/:projectId", element: deferred(<ProjectDetailPage />) },
              { path: "/projects/:projectId/:view", element: deferred(<ProjectDetailPage />) },
              { path: "/projects/:projectId/chapters/:chapterId", element: deferred(<ProjectDetailPage />) },
              { path: "/projects/:projectId/workflow/:unitId/:stage", element: deferred(<ProjectDetailPage />) },
              { path: "/assets", element: deferred(<AssetsPage />) },
              { path: "/tasks", element: deferred(<TasksPage />) },
              { path: "/plugins", element: deferred(<PluginsPage />) },
              { path: "/plugins/eagle", element: deferred(<EaglePage />) },
              { path: "/settings", element: deferred(<SettingsPage />) },
            ],
          },
        ],
        { initialEntries: ["/canvas?mode=new"] },
      ),
    [],
  );

  return (
    <div className="yingce-island yc-root h-full min-h-0 overflow-hidden" data-testid="canvas-studio">
      <AppProviders>
        <RouterProvider router={router} />
      </AppProviders>
    </div>
  );
}
