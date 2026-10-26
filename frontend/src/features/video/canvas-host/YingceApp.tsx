/**
 * 本文件拥有把影策页面嵌进 Yoyo 的记忆路由、标题和会话绑定。
 * 不拥有那些页面的业务。进入画布路由时会绑定会话。它自己不跑模型循环。
 */
import { lazy, Suspense, useEffect, useLayoutEffect, useMemo, type ReactNode } from "react";
import { createMemoryRouter, Outlet, RouterProvider, useLocation, useParams } from "react-router";
import { AppProviders } from "@yingce/components/layout/app-providers";
import { RequireAuth } from "@yingce/components/auth/require-auth";
import { CanvasRefreshShell } from "@yingce/pages/canvas/canvas-refresh-shell";
import RouteErrorPage from "@yingce/pages/route-error";
import UserLayout from "@yingce/layouts/user-layout";
import { useCanvasStore } from "@yingce/stores/canvas/use-canvas-store";
import { bootstrapAppearance } from "@yingce/services/appearance-bootstrap";
import { useCopy } from "../../../lib/i18n";
import { bindCanvasSession, setCanvasHostTitle } from "./session";
import { installCanvasHub } from "./hub";
import { installHostNavigation } from "./host-nav";
import { primeCanvasMediaBase } from "./media";
import { syncYoyoPopupRoot } from "./design-system/popup-container";
import "@yingce/styles/globals.css";
import "@yingce/styles/shared/model-picker.css";
import "@yingce/styles/shared/overlays.css";
import "@yingce/styles/shared/scrollbars.css";
import "./design-system/tokens.css";
import "./design-system/overlays.css";
import "./apple-skin.css";
import "./host-dark.css";
import "./design-system/studio-lock.css";

const CanvasPage = lazy(() => import("@yingce/pages/canvas"));
const CanvasProjectPage = lazy(() => import("@yingce/pages/canvas/project"));
const CreatePage = lazy(() => import("@yingce/pages/create"));
const ProjectsPage = lazy(() => import("@yingce/pages/projects"));
const ProjectDetailPage = lazy(() => import("@yingce/pages/projects/detail"));
const DramaAgentPage = lazy(() => import("./DramaAgentPage"));
const AssetsPage = lazy(() => import("@yingce/pages/assets"));
const SkillsPage = lazy(() => import("@yingce/pages/skills"));
const TasksPage = lazy(() => import("@yingce/pages/tasks"));
const PluginsPage = lazy(() => import("@yingce/pages/plugins"));
const EaglePage = lazy(() => import("@yingce/pages/plugins/eagle"));
const SettingsPage = lazy(() => import("@yingce/pages/settings"));

if (typeof window !== "undefined") {
  (window as Window & { __YOYO_CANVAS_HOST__?: boolean }).__YOYO_CANVAS_HOST__ = true;
}

/** 用刷新壳包住懒加载页面。node 放进 Suspense。fallback 有传入时传给 Suspense，否则用 CanvasRefreshShell。 */
function deferred(node: ReactNode, fallback?: ReactNode) {
  return <Suspense fallback={fallback || <CanvasRefreshShell />}>{node}</Suspense>;
}

/** 画布路由上绑定会话，并按路径改宿主标题。 */
function BindAndTitle() {
  const copy = useCopy();
  const params = useParams();
  const location = useLocation();
  const sessionId = typeof window !== "undefined" ? String((window as Window & { __YOYO_VIDEO_SESSION__?: string }).__YOYO_VIDEO_SESSION__ || "") : "";
  const projects = useCanvasStore(/** s 取出 projects。 */ (s) => s.projects);
  const id = params.id || params.projectId || "";

  useEffect(() => {
    if (id && sessionId && location.pathname.startsWith("/canvas/")) void bindCanvasSession(sessionId, id);
  }, [id, sessionId, location.pathname]);

  useEffect(() => {
    const match = projects.find(/** p 用 id 和当前路由参数比对。 */ (p) => p.id === id);
    const path = location.pathname;
    const title = match?.title
      || (path.startsWith("/projects") || path.startsWith("/drama") ? copy.video.drama
        : path.startsWith("/create") || path === "/" ? copy.video.shellCreate
          : path.startsWith("/assets") ? copy.video.shellAssets
            : path.startsWith("/plugins") ? copy.video.shellPlugins
              : path.startsWith("/tasks") ? copy.video.shellTasks
                : path.startsWith("/skills") ? copy.video.shellSkills
                  : copy.video.canvas);
    setCanvasHostTitle(title);
  }, [id, location.pathname, projects, copy.video.drama, copy.video.shellCreate, copy.video.shellAssets, copy.video.shellPlugins, copy.video.shellTasks, copy.video.shellSkills, copy.video.canvas]);

  return null;
}

/** 套上登录和工作区布局，并挂上标题绑定。 */
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

/** YingceApp 用记忆路由打开创作、短剧、画布和资料页，并接上画布事件与宿主跳转。initialPath 放进记忆路由的初始地址。 */
export default function YingceApp({ initialPath = "/create" }: { initialPath?: string }) {
  useLayoutEffect(() => {
    void bootstrapAppearance();
  }, []);

  useEffect(() => {
    /** 去掉岛上的明暗类，并同步浮层根。 */
    const syncTheme = () => {
      const island = document.querySelector(".yingce-island") as HTMLElement | null;
      island?.classList.remove("dark", "light");
      syncYoyoPopupRoot();
    };
    syncTheme();
    const observer = new MutationObserver(syncTheme);
    observer.observe(document.documentElement, { attributes: true, attributeFilter: ["class", "data-palette"] });
    void primeCanvasMediaBase();
    void import("@yingce/lib/plugins/builtin");
    const uninstallHub = installCanvasHub();
    const uninstallHostNav = installHostNavigation();
    return () => {
      observer.disconnect();
      uninstallHub?.();
      uninstallHostNav?.();
    };
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
              { path: "/drama", element: deferred(<DramaAgentPage />) },
              { path: "/canvas", element: deferred(<CanvasPage />) },
              { path: "/canvas/:id", element: deferred(<CanvasProjectPage />, <CanvasRefreshShell />) },
              { path: "/projects", element: deferred(<ProjectsPage />) },
              { path: "/projects/:projectId", element: deferred(<ProjectDetailPage />) },
              { path: "/projects/:projectId/:view", element: deferred(<ProjectDetailPage />) },
              { path: "/projects/:projectId/chapters/:chapterId", element: deferred(<ProjectDetailPage />) },
              { path: "/projects/:projectId/workflow/:unitId/:stage", element: deferred(<ProjectDetailPage />) },
              { path: "/assets", element: deferred(<AssetsPage />) },
              { path: "/skills", element: deferred(<SkillsPage />) },
              { path: "/tasks", element: deferred(<TasksPage />) },
              { path: "/plugins", element: deferred(<PluginsPage />) },
              { path: "/plugins/eagle", element: deferred(<EaglePage />) },
              { path: "/settings", element: deferred(<SettingsPage />) },
            ],
          },
        ],
        { initialEntries: [initialPath] },
      ),
    [initialPath],
  );

  return (
    <div className="yingce-island yc-root h-full min-h-0 overflow-hidden" data-testid="canvas-studio">
      <AppProviders>
        <RouterProvider router={router} />
      </AppProviders>
    </div>
  );
}
