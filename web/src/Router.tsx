import { lazy, Suspense, useMemo } from "react";
import { Navigate, Outlet, createBrowserRouter, RouterProvider, type RouteObject } from "react-router";

import { Layout } from "@/Layout";
import { OnboardingGate } from "@/components/auth/OnboardingGate";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { useBootstrapStatus } from "@/hooks/AuthHooks";
import { AutomationPage } from "@/pages/AutomationPage";
import { AutomationsPage } from "@/pages/AutomationsPage";
import { BoardPage } from "@/pages/BoardPage";
import { ChatPage } from "@/pages/ChatPage";
import { DeployPage } from "@/pages/DeployPage";
import { DocsPage } from "@/pages/DocsPage";
import { DnsOnboardingPage } from "@/pages/DnsOnboardingPage";
import { ErrorPage } from "@/pages/ErrorPage";
import { FirstLoginWizardPage } from "@/pages/FirstLoginWizardPage";
import { HomePage } from "@/pages/HomePage";
import { InboxPage } from "@/pages/InboxPage";
import { InvitePreviewPage } from "@/pages/InvitePreviewPage";
import { InstanceBootstrapPage } from "@/pages/InstanceBootstrapPage";
import { LoginPage } from "@/pages/LoginPage";
import { MembersPage } from "@/pages/MembersPage";
import { MemoriesPage } from "@/pages/MemoriesPage";
import { OwnerWizardPage } from "@/pages/OwnerWizardPage";
import { ProjectSettingsPage } from "@/pages/ProjectSettingsPage";
import { ProjectWizardImportPage } from "@/pages/ProjectWizardImportPage";
import { ProjectWizardPage } from "@/pages/ProjectWizardPage";
import { RunnersPage } from "@/pages/RunnersPage";
import { ServicePage } from "@/pages/ServicePage";
import { SettingsPage } from "@/pages/SettingsPage";
import { StackPage } from "@/pages/StackPage";
import { TicketPage } from "@/pages/TicketPage";
import { useSessionStore } from "@/stores/sessionStore";

// Lazy-loaded: together their deps would double the main bundle, and most sessions never visit either route.
const DocPage = lazy(() => import("@/pages/DocPage").then((m) => ({ default: m.DocPage })));
const MemoryPage = lazy(() => import("@/pages/MemoryPage").then((m) => ({ default: m.MemoryPage })));
const InterviewPage = lazy(() => import("@/pages/InterviewPage").then((m) => ({ default: m.InterviewPage })));
const TopologyPage = lazy(() =>
  import("@/pages/TopologyPage").then((m) => ({ default: m.TopologyPage })),
);

const buildRoutes = (loggedIn: boolean): RouteObject[] => [
  {
    element: <Layout />,
    // Catches a lazy route's dynamic import throw (deploy while a tab is open) instead of the default error screen.
    errorElement: <ErrorPage />,
    children: [
      { path: "/login", element: <LoginPage /> },
      { path: "/invite", element: <InvitePreviewPage /> },
      // Re-entry to bootstrap for a wrong GitHub App; the API only accepts it while no user exists.
      { path: "/setup", element: <InstanceBootstrapPage /> },
      ...(loggedIn
        ? [
            { path: "/wizard/onboarding/owner", element: <OwnerWizardPage /> },
            { path: "/wizard/onboarding/profile", element: <FirstLoginWizardPage /> },
            // Old onboarding paths redirect: onboarding lives under /wizard/… now.
            { path: "/onboarding/owner", element: <Navigate to="/wizard/onboarding/owner" replace /> },
            { path: "/onboarding/profile", element: <Navigate to="/wizard/onboarding/profile" replace /> },
            { path: "/onboarding/dns", element: <Navigate to="/wizard/onboarding/dns" replace /> },
            {
              element: (
                <OnboardingGate>
                  <Outlet />
                </OnboardingGate>
              ),
              children: [
                { path: "/", element: <HomePage /> },
                { path: "/inbox", element: <InboxPage /> },
                { path: "/chat", element: <ChatPage /> },
                { path: "/chat/:conversationId", element: <ChatPage /> },
                { path: "/wizard/onboarding/dns", element: <DnsOnboardingPage /> },
                { path: "/wizard/project/import", element: <ProjectWizardImportPage /> },
                { path: "/wizard/project/:step", element: <ProjectWizardPage /> },
                {
                  path: "/topology",
                  element: (
                    <Suspense fallback={<LoadingDisplay />}>
                      <TopologyPage />
                    </Suspense>
                  ),
                },
                { path: "/docs", element: <DocsPage /> },
                {
                  path: "/docs/:projectToken/:docId",
                  element: (
                    <Suspense fallback={<LoadingDisplay />}>
                      <DocPage />
                    </Suspense>
                  ),
                },
                {
                  // Legacy shape for old links/mention chips with only a doc id; DocPage fetches by id either way.
                  path: "/docs/:docId",
                  element: (
                    <Suspense fallback={<LoadingDisplay />}>
                      <DocPage />
                    </Suspense>
                  ),
                },
                { path: "/memories", element: <MemoriesPage /> },
                {
                  path: "/memories/:projectToken/:memoryId",
                  element: (
                    <Suspense fallback={<LoadingDisplay />}>
                      <MemoryPage />
                    </Suspense>
                  ),
                },
                {
                  // Legacy shape mirroring docs' bare-id route.
                  path: "/memories/:memoryId",
                  element: (
                    <Suspense fallback={<LoadingDisplay />}>
                      <MemoryPage />
                    </Suspense>
                  ),
                },
                { path: "/board", element: <BoardPage /> },
                { path: "/board/:projectId", element: <BoardPage /> },
                { path: "/tickets/:ticketId", element: <TicketPage /> },
                { path: "/runners", element: <RunnersPage /> },
                { path: "/automations", element: <AutomationsPage /> },
                { path: "/automations/:id", element: <AutomationPage /> },
                { path: "/services/:serviceId", element: <ServicePage /> },
                { path: "/stacks/:stackId", element: <StackPage /> },
                { path: "/stacks/:stackId/deploys/:deployId", element: <DeployPage /> },
                { path: "/members", element: <MembersPage /> },
                { path: "/settings", element: <SettingsPage /> },
                { path: "/projects/:projectId/settings", element: <ProjectSettingsPage /> },
                {
                  path: "/projects/:projectId/interview",
                  element: (
                    <Suspense fallback={<LoadingDisplay />}>
                      <InterviewPage />
                    </Suspense>
                  ),
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
  const router = useMemo(() => createBrowserRouter(buildRoutes(loggedIn)), [loggedIn]);

  // Rendered here, not as a route: once bootstrap-status reports configured, no URL can reach this page again.
  const { data: bootstrapStatus, isPending } = useBootstrapStatus();
  const needsBootstrap = bootstrapStatus !== undefined && !bootstrapStatus.configured;

  // key: remount RouterProvider on the auth flip, or the old router's stale subscription desyncs UI state.
  return (
    <>
      {isPending && <LoadingDisplay />}
      {needsBootstrap && <InstanceBootstrapPage />}
      {!isPending && !needsBootstrap && <RouterProvider key={loggedIn ? "in" : "out"} router={router} />}
    </>
  );
};
