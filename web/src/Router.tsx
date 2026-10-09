import { Suspense, useEffect, useMemo, useState, type ReactNode } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { Outlet, createBrowserRouter, RouterProvider, type RouteObject } from "react-router";

import { Layout } from "@/Layout";
import { AreaGate } from "@/components/auth/AreaGate";
import { ProjectRevokedGate } from "@/components/project/ProjectRevokedGate";
import { OnboardingGate } from "@/components/auth/OnboardingGate";
import { WorkspaceScope } from "@/components/workspace/WorkspaceScope";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { useBootstrapStatus } from "@/hooks/AuthHooks";
import { TabSegmentContext } from "@/hooks/useTabPath";
import { ErrorPage } from "@/pages/ErrorPage";
import {
  AutomationPage,
  AutomationsPage,
  BoardPage,
  ChatPage,
  ConfigurationPage,
  DeployPage,
  DnsOnboardingPage,
  DocsPage,
  FirstLoginWizardPage,
  HomePage,
  InboxPage,
  InterviewPage,
  InvitePreviewPage,
  LoginPage,
  MemoriesPage,
  OwnerWizardPage,
  ProjectSettingsPage,
  ProjectWizardImportPage,
  ProjectWizardPage,
  RunnersPage,
  SetupPage,
  StackPage,
  TicketPage,
  TopologyPage,
  WorkspaceEntryPage,
  YourSettingsPage,
  preloadMatchedPages,
  warmPagesWhenIdle,
} from "@/pageChunks";
import { useSessionStore } from "@/stores/sessionStore";
import type { RouteAccess, RouteArea } from "@/models/Access";

const gate = (area: RouteArea): RouteAccess => ({ area });

// Each tab is a static route where `:tab` would swallow a sibling's `:id`; the bare path is wrapped alike so tab switches don't remount.
const staticTabs = (path: string, tabs: string[], area: RouteArea, element: ReactNode): RouteObject[] =>
  [undefined, ...tabs].map((tab) => ({
    path: tab ? `${path}/${tab}` : path,
    handle: gate(area),
    element: <TabSegmentContext value={tab}>{element}</TabSegmentContext>,
  }));

// Every page reached from a workspace's sidebar lives under its slug; personal and instance pages stay unprefixed.
const workspaceRoutes: RouteObject[] = [
  { index: true, element: <HomePage /> },
  { path: "inbox/:rowKey?/:tab?", element: <InboxPage /> },
  { path: "chat", element: <ChatPage /> },
  { path: "chat/:conversationId", element: <ChatPage /> },
  { path: "wizard/project/import", handle: gate("newProject"), element: <ProjectWizardImportPage /> },
  { path: "wizard/project/:step", handle: gate("newProject"), element: <ProjectWizardPage /> },
  {
    path: "topology",
    handle: gate("topology"),
    element: (
      <Suspense fallback={<LoadingDisplay />}>
        <TopologyPage />
      </Suspense>
    ),
  },
  { path: "docs", element: <DocsPage /> },
  { path: "docs/:projectToken/:docId", element: <DocsPage /> },
  // Mention chips carry only the doc id; the page moves them to the project's URL.
  { path: "docs/:docId", element: <DocsPage /> },
  { path: "memories", handle: gate("memories"), element: <MemoriesPage /> },
  { path: "memories/:projectToken/:memoryId/:tab?", handle: gate("memories"), element: <MemoriesPage /> },
  { path: "board", handle: gate("tickets"), element: <BoardPage /> },
  { path: "board/:projectId", handle: gate("tickets"), element: <BoardPage /> },
  { path: "tickets/:ticketId", handle: gate("tickets"), element: <TicketPage /> },
  { path: "runners", handle: gate("runners"), element: <RunnersPage /> },
  ...staticTabs("automations", ["hosts", "secrets"], "automations", <AutomationsPage />),
  { path: "automations/:id/:tab?", handle: gate("automations"), element: <AutomationPage /> },
  { path: "stacks/:stackId/logs/:service?", handle: gate("stacks"), element: <StackPage forcedSection="logs" /> },
  { path: "stacks/:stackId/:section?", handle: gate("stacks"), element: <StackPage /> },
  { path: "stacks/:stackId/deploys/:deployId", handle: gate("deploys"), element: <DeployPage /> },
  { path: "configuration/:section?", handle: gate("configuration"), element: <ConfigurationPage /> },
  { path: "projects/:projectId/settings/:section?/:tab?", handle: gate("projects"), element: <ProjectSettingsPage /> },
  {
    path: "projects/:projectId/interview/:tab?",
    handle: gate("memories"),
    element: (
      <Suspense fallback={<LoadingDisplay />}>
        <InterviewPage />
      </Suspense>
    ),
  },
];

const buildRoutes = (loggedIn: boolean): RouteObject[] => [
  {
    element: <Layout />,
    // Catches a lazy route's dynamic import throw (deploy while a tab is open) instead of the default error screen.
    // It replaces the Layout, so signed in it supplies the frame the screen expects to fill.
    errorElement: (
      <main className={loggedIn ? "h-dvh p-2" : undefined}>
        <ErrorPage />
      </main>
    ),
    children: [
      { path: "/login", element: <LoginPage /> },
      { path: "/invite", element: <InvitePreviewPage /> },
      // The handoff link's landing and re-entry for a wrong GitHub App; the API only accepts either while no user exists.
      { path: "/setup", element: <SetupPage /> },
      ...(loggedIn
        ? [
            { path: "/wizard/onboarding/owner", element: <OwnerWizardPage /> },
            { path: "/wizard/onboarding/profile", element: <FirstLoginWizardPage /> },
            {
              element: (
                <OnboardingGate>
                  <Outlet />
                </OnboardingGate>
              ),
              children: [
                { path: "/", element: <WorkspaceEntryPage /> },
                { path: "/wizard/onboarding/dns", element: <DnsOnboardingPage /> },
                { path: "/settings/:section?/:tab?", element: <YourSettingsPage /> },
                {
                  path: "/:workspace",
                  element: <WorkspaceScope />,
                  children: [
                    {
                      element: (
                        <ProjectRevokedGate>
                          <AreaGate>
                            <Outlet />
                          </AreaGate>
                        </ProjectRevokedGate>
                      ),
                      children: workspaceRoutes,
                    },
                  ],
                },
              ],
            },
          ]
        : [{ path: "/", element: <HomePage /> }]),
      { path: "*", element: <ErrorPage /> },
    ],
  },
];

export const AppRouter = () => {
  const loggedIn = useSessionStore((s) => s.isLoggedIn);
  const client = useQueryClient();
  const routes = useMemo(() => buildRoutes(loggedIn), [loggedIn]);
  const router = useMemo(() => createBrowserRouter(routes), [routes]);
  // The page the URL opens on loads alongside the bootstrap request, and the router waits for it, so the page renders in
  // the same pass as the shell instead of behind a fallback React holds for 300ms.
  const [pageLoaded, setPageLoaded] = useState(false);
  useEffect(() => {
    void preloadMatchedPages(routes).then(() => setPageLoaded(true));
  }, [routes]);
  // Signed out, the next page load is the sign-in redirect, so only a session warms the other pages.
  useEffect(() => (loggedIn ? warmPagesWhenIdle(client) : undefined), [client, loggedIn]);

  // Rendered here, not as a route: once bootstrap-status reports configured, no URL can reach this page again.
  const { data: bootstrapStatus, isPending } = useBootstrapStatus();
  const needsBootstrap = bootstrapStatus !== undefined && !bootstrapStatus.configured;

  // key: remount RouterProvider on the auth flip, or the old router's stale subscription desyncs UI state.
  const routerReady = !isPending && pageLoaded;
  return (
    <>
      {!routerReady && !needsBootstrap && <LoadingDisplay />}
      {needsBootstrap && (
        <Suspense fallback={<LoadingDisplay />}>
          <SetupPage />
        </Suspense>
      )}
      {routerReady && !needsBootstrap && <RouterProvider key={loggedIn ? "in" : "out"} router={router} />}
    </>
  );
};
