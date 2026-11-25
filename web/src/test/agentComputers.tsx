import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render } from "@testing-library/react";
import { ContextAwareConfirmation } from "react-confirm";
import { createMemoryRouter, RouterProvider } from "react-router";

import { AgentPage } from "@/pages/AgentPage";

// A computer added with its app, online; overrides reshape it.
export const computer = (overrides: Record<string, unknown> = {}) => ({
  id: "c1",
  name: "Home",
  server_url: "http://c1.nexul-computer.invalid",
  runner: { connected: true, last_seen: "2026-10-10T12:00:00Z", reconnecting: false },
  token_expires_at: "2099-01-01T00:00:00Z",
  kind: "t3code",
  harness_version: "0.0.34",
  position: 0,
  runs_at_once: 1,
  running: 0,
  created_at: "2026-08-01T00:00:00Z",
  updated_at: "2026-08-01T00:00:00Z",
  ...overrides,
});

// A computer paired by URL before the Nexul app.
export const pairedBefore = (overrides: Record<string, unknown> = {}) => computer({ server_url: "https://home.example.com", runner: undefined, ...overrides });

export const unconfirmed = { computer_id: "c1", confirmed_at: null, providers: [], skipped_providers: [], turns: [] };

// The Agent page at a path, in its own client so a test can push a refresh at it; the router says where a click went.
export const renderAgent = (path: string) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const router = createMemoryRouter(
    [
      { path: "/acme/agent", element: <AgentPage /> },
      { path: "/acme/agent/:kind/:id", element: <AgentPage /> },
    ],
    { initialEntries: [path] },
  );
  render(
    <QueryClientProvider client={client}>
      <ContextAwareConfirmation.ConfirmationRoot />
      <RouterProvider router={router} />
    </QueryClientProvider>,
  );
  return { client, router };
};
