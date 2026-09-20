import { useState } from "react";
import { Outlet, useLocation } from "react-router";

import { Sidebar } from "@/components/sidebar/Sidebar";
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
  // Wizards own the whole viewport; the sidebar's workspace nav has nothing to point at mid-wizard.
  const onboarding = useLocation().pathname.startsWith("/wizard/");

  return (
    <div className="flex min-h-screen">
      {!onboarding && (
        <Sidebar
          collapsed={collapsed}
          onToggleCollapse={() => setCollapsed((c) => !c)}
          isLoggedIn={isLoggedIn}
          unreadCount={unread?.count ?? 0}
        />
      )}
      <div className="flex min-w-0 flex-1 flex-col">
        <main className="flex-1">
          <Outlet />
        </main>
      </div>
    </div>
  );
};
