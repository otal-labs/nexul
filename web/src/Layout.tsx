import { Suspense, useRef, useState } from "react";
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
import { useUnreadTitle } from "@/hooks/useUnreadTitle";
import { buildLiveURL } from "@/lib/live";
import { useSessionStore } from "@/stores/sessionStore";
import { useIsViewingAs } from "@/stores/viewAsStore";

export const Layout = () => {
  // At 768px an open sidebar leaves the page too little width, so it starts as the icon rail below 1024px.
  const [collapsed, setCollapsed] = useState(() => window.innerWidth < 1024);
  const isLoggedIn = useSessionStore((s) => s.isLoggedIn);
  const token = useSessionStore((s) => s.token);
  const viewing = useIsViewingAs();
  // The socket authenticates as the signed-in member and cannot carry the view headers, so a view leaves it closed.
  useLiveEvents(token && !viewing ? buildLiveURL(token) : null);
  const { data: unread } = useFetchUnreadCount(isLoggedIn);
  useEnsureWorkspaceSelected(isLoggedIn);
  useUnreadTitle(isLoggedIn);
  // Wizards, the invite page and the signed-out pages own the whole viewport; the sidebar's nav has nothing to point at there.
  const onboarding = /^\/(?:(?:[^/]+\/)?wizard\/|invite$)/.test(useLocation().pathname);
  const frame = useRef<HTMLDivElement>(null);
  usePageEntrance(frame);

  // Signed out, the hero pages paint their own full-screen canvas; signed in, every page floats on the light field.
  return (
    <>
      {!isLoggedIn && (
        <main className="min-h-screen">
          <Suspense fallback={<LoadingDisplay />}>
            <Outlet />
          </Suspense>
        </main>
      )}
      {isLoggedIn && (
        <div className="flex h-dvh">
          <div className="light-field" aria-hidden />
          {!onboarding && (
            <Sidebar collapsed={collapsed} onToggleCollapse={() => setCollapsed((c) => !c)} unreadCount={unread?.count ?? 0} />
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
