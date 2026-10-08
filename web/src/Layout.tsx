import { useState } from "react";
import { Outlet, useLocation } from "react-router";

import { PhoneBanner } from "@/components/PhoneBanner";
import { ServerUpdatedBanner } from "@/components/ServerUpdatedBanner";
import { Sidebar } from "@/components/sidebar/Sidebar";
import { VoiceCallAudio } from "@/components/voice/VoiceCallAudio";
import { useFetchUnreadCount } from "@/hooks/NotificationHooks";
import { useEnsureWorkspaceSelected } from "@/hooks/WorkspaceHooks";
import { useLiveEvents } from "@/hooks/useLiveEvents";
import { buildLiveURL } from "@/lib/live";
import { useSessionStore } from "@/stores/sessionStore";

export const Layout = () => {
  const [collapsed, setCollapsed] = useState(() => window.innerWidth < 768);
  const isLoggedIn = useSessionStore((s) => s.isLoggedIn);
  const token = useSessionStore((s) => s.token);
  useLiveEvents(token ? buildLiveURL(token) : null);
  const { data: unread } = useFetchUnreadCount(isLoggedIn);
  useEnsureWorkspaceSelected(isLoggedIn);
  // Wizards and the signed-out pages own the whole viewport; the sidebar's nav has nothing to point at there.
  const onboarding = /^\/(?:[^/]+\/)?wizard\//.test(useLocation().pathname);

  return (
    <div className="flex min-h-screen">
      {isLoggedIn && !onboarding && (
        <Sidebar collapsed={collapsed} onToggleCollapse={() => setCollapsed((c) => !c)} unreadCount={unread?.count ?? 0} />
      )}
      <div className="flex min-w-0 flex-1 flex-col">
        <ServerUpdatedBanner />
        <PhoneBanner />
        <main className="flex-1">
          <Outlet />
        </main>
      </div>
      <VoiceCallAudio />
    </div>
  );
};
