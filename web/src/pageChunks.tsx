import { isValidElement, lazy, type ComponentType, type ReactNode } from "react";
import type { QueryClient } from "@tanstack/react-query";
import { matchRoutes, type RouteObject } from "react-router";

import { whenIdle } from "@/lib/idle";
import { loadTicketForm } from "@/utils/loadTicketForm";

// Every page is its own chunk: a cold start loads the shell and the page it opens on, the rest once the first screen settles.
const page = <M, C extends ComponentType<Record<string, never>>>(load: () => Promise<M>, pick: (m: M) => C) => {
  let loaded: C | undefined;
  const preload = () => load().then((m) => (loaded = pick(m)));
  // A page already loaded resolves synchronously, so it renders in the same pass instead of behind a fallback React holds for 300ms.
  const resolved = (component: C) => ({ then: (fulfil: (m: { default: C }) => void) => fulfil({ default: component }) });
  const Page = lazy(() => (loaded ? (resolved(loaded) as unknown as Promise<{ default: C }>) : preload().then((component) => ({ default: component }))));
  return Object.assign(Page, { preload });
};

export const AutomationPage = page(() => import("@/pages/AutomationPage"), (m) => m.AutomationPage);
export const AutomationsPage = page(() => import("@/pages/AutomationsPage"), (m) => m.AutomationsPage);
export const BoardPage = page(() => import("@/pages/BoardPage"), (m) => m.BoardPage);
export const ChatPage = page(() => import("@/pages/ChatPage"), (m) => m.ChatPage);
export const ConfigurationPage = page(() => import("@/pages/ConfigurationPage"), (m) => m.ConfigurationPage);
export const DeployPage = page(() => import("@/pages/DeployPage"), (m) => m.DeployPage);
export const DnsOnboardingPage = page(() => import("@/pages/DnsOnboardingPage"), (m) => m.DnsOnboardingPage);
export const DocsPage = page(() => import("@/pages/DocsPage"), (m) => m.DocsPage);
export const FirstLoginWizardPage = page(() => import("@/pages/FirstLoginWizardPage"), (m) => m.FirstLoginWizardPage);
export const HomePage = page(() => import("@/pages/HomePage"), (m) => m.HomePage);
export const InboxPage = page(() => import("@/pages/InboxPage"), (m) => m.InboxPage);
export const InterviewPage = page(() => import("@/pages/InterviewPage"), (m) => m.InterviewPage);
export const InvitePreviewPage = page(() => import("@/pages/InvitePreviewPage"), (m) => m.InvitePreviewPage);
export const LoginPage = page(() => import("@/pages/LoginPage"), (m) => m.LoginPage);
export const MemoriesPage = page(() => import("@/pages/MemoriesPage"), (m) => m.MemoriesPage);
export const OwnerWizardPage = page(() => import("@/pages/OwnerWizardPage"), (m) => m.OwnerWizardPage);
export const ProjectSettingsPage = page(() => import("@/pages/ProjectSettingsPage"), (m) => m.ProjectSettingsPage);
export const ProjectWizardImportPage = page(() => import("@/pages/ProjectWizardImportPage"), (m) => m.ProjectWizardImportPage);
export const ProjectWizardPage = page(() => import("@/pages/ProjectWizardPage"), (m) => m.ProjectWizardPage);
export const RunnersPage = page(() => import("@/pages/RunnersPage"), (m) => m.RunnersPage);
export const SetupPage = page(() => import("@/pages/SetupPage"), (m) => m.SetupPage);
export const StackPage = page(() => import("@/pages/StackPage"), (m) => m.StackPage);
export const TicketPage = page(() => import("@/pages/TicketPage"), (m) => m.TicketPage);
export const TopologyPage = page(() => import("@/pages/TopologyPage"), (m) => m.TopologyPage);
export const WorkspaceEntryPage = page(() => import("@/pages/WorkspaceEntryPage"), (m) => m.WorkspaceEntryPage);
export const YourSettingsPage = page(() => import("@/pages/YourSettingsPage"), (m) => m.YourSettingsPage);

// Topology and Interview are left to load on a visit: most sessions never open them, and their chunks are the heaviest.
const warmed: (() => Promise<unknown>)[] = [
  AutomationPage.preload,
  AutomationsPage.preload,
  BoardPage.preload,
  ChatPage.preload,
  ConfigurationPage.preload,
  DeployPage.preload,
  DnsOnboardingPage.preload,
  DocsPage.preload,
  FirstLoginWizardPage.preload,
  HomePage.preload,
  InboxPage.preload,
  InvitePreviewPage.preload,
  LoginPage.preload,
  MemoriesPage.preload,
  OwnerWizardPage.preload,
  ProjectSettingsPage.preload,
  ProjectWizardImportPage.preload,
  ProjectWizardPage.preload,
  RunnersPage.preload,
  SetupPage.preload,
  StackPage.preload,
  TicketPage.preload,
  WorkspaceEntryPage.preload,
  YourSettingsPage.preload,
  loadTicketForm,
];

const preloadElement = (node: ReactNode): Promise<unknown>[] => {
  if (!isValidElement<{ children?: ReactNode }>(node)) return [];
  const own = (node.type as { preload?: () => Promise<unknown> }).preload?.();
  return [...(own ? [own] : []), ...preloadElement(node.props.children)];
};

// Settles once every page the URL opens on has loaded, failed loads included: the route's error screen reports those.
export const preloadMatchedPages = (routes: RouteObject[]) =>
  Promise.allSettled((matchRoutes(routes, window.location) ?? []).flatMap((match) => preloadElement(match.route.element)));

const SETTLE_POLL_MS = 500;


// Once the first screen's requests have settled, one chunk per idle period, so a first visit to any page doesn't wait on the
// network and warming never competes with the page being opened.
export const warmPagesWhenIdle = (client: QueryClient) => {
  let cancelled = false;
  const queue = [...warmed];
  const next = () => {
    const load = queue.shift();
    if (cancelled || !load) return;
    void load().finally(() => whenIdle(next));
  };
  const start = () => {
    if (cancelled) return;
    if (client.isFetching() > 0) {
      setTimeout(start, SETTLE_POLL_MS);
      return;
    }
    whenIdle(next);
  };
  setTimeout(start, SETTLE_POLL_MS);
  return () => {
    cancelled = true;
  };
};
