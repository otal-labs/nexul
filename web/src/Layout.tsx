import { Suspense, useRef } from "react";
import { Outlet, useLocation } from "react-router";

import { CommandPalette } from "@/components/command/CommandPalette";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { PhoneBanner } from "@/components/PhoneBanner";
import { ServerUpdatedBanner } from "@/components/ServerUpdatedBanner";
import { ViewAsBanner } from "@/components/ViewAsBanner";
import { Sidebar } from "@/components/sidebar/Sidebar";
import { VoiceCallAudio } from "@/components/voice/VoiceCallAudio";
import { useFetchUnreadCount } from "@/hooks/NotificationHooks";
import { useEnsureWorkspaceSelected } from "@/hooks/WorkspaceHooks";
import { useLiveEvents } from "@/hooks/useLiveEvents";
import { usePageEntrance } from "@/hooks/usePageEntrance";
import { useReportTimezone } from "@/hooks/useReportTimezone";
import { useUnreadTitle } from "@/hooks/useUnreadTitle";
import { buildLiveURL } from "@/lib/live";
import { useSessionStore } from "@/stores/sessionStore";
import { useSidebarStore } from "@/stores/sidebarStore";
import { useIsViewingAs } from "@/stores/viewAsStore";

export const Layout = () => {
  const collapsed = useSidebarStore((s) => s.collapsed);
  const toggleCollapsed = useSidebarStore((s) => s.toggleCollapsed);
  const isLoggedIn = useSessionStore((s) => s.isLoggedIn);
  const token = useSessionStore((s) => s.token);
  const viewing = useIsViewingAs();
  // The socket authenticates as the signed-in member and cannot carry the view headers, so a view leaves it closed.
  useLiveEvents(token && !viewing ? buildLiveURL(token) : null);
  const { data: unread } = useFetchUnreadCount(isLoggedIn);
  useEnsureWorkspaceSelected(isLoggedIn);
  useUnreadTitle(isLoggedIn);
  useReportTimezone(isLoggedIn);
  const { pathname } = useLocation();
  // Wizards own the whole app frame; the sidebar's nav has nothing to point at there.
  const onboarding = /^\/(?:[^/]+\/)?wizard\//.test(pathname);
  // Sign-in, the invitation and setup draw their own full-screen frame, signed in or not.
  const shell = isLoggedIn && !/^\/(?:login|invite|setup)$/.test(pathname);
  const frame = useRef<HTMLDivElement>(null);
  usePageEntrance(frame);

  // Outside the shell a page paints its own full-screen canvas; inside it, every page floats on the light field.
  return (
    <>
      {!shell && (
        <main className="min-h-screen">
          <Suspense fallback={<LoadingDisplay />}>
            <Outlet />
          </Suspense>
        </main>
      )}
      {shell && (
        <div className="flex h-dvh">
          <div className="light-field" aria-hidden />
          {!onboarding && (
            <Sidebar collapsed={collapsed} onToggleCollapse={toggleCollapsed} unreadCount={unread?.count ?? 0} />
          )}
          {!onboarding && <CommandPalette />}
          <div className="flex min-w-0 flex-1 flex-col">
            <ViewAsBanner />
            <ServerUpdatedBanner />
            <PhoneBanner />
            <main className="min-h-0 flex-1 p-2">
              <div ref={frame} className="app-frame">
                <Suspense fallback={<LoadingDisplay />}>
                  <Outlet />
                </Suspense>
              </div>
            </main>
          </div>
        </div>
      )}
      <VoiceCallAudio />
    </>
  );
};
