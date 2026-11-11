import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { AutomationHostsSection } from "@/components/automationHost/AutomationHostsSection";
import { resetViewer } from "@/test/viewerActions";

vi.mock("@/api/client", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/api/client")>()),
  api: { get: vi.fn() },
}));
vi.mock("@/hooks/useActionState", async () => (await import("@/test/viewerActions")).mockedActionState);
vi.mock("@/hooks/AccessHooks", () => ({ useIsInstanceOwner: () => false }));

const host = (name: string, managers: unknown[]) => ({
  id: `h-${name}`,
  name,
  machine: "prod",
  os: "linux",
  arch: "amd64",
  version: "v0.3.0",
  connected: true,
  last_seen: "2026-08-12T12:00:00Z",
  managers,
});

const hosts = [
  host("jobs", [
    { workspace_id: "ws-1", manager: "op", state: "ready" },
    { workspace_id: "ws-2", manager: "op", state: "token rejected" },
    { workspace_id: "ws-3", manager: "op", state: "unreachable" },
  ]),
  host("bare", []),
];

describe("AutomationHostsSection", () => {
  beforeEach(() => {
    resetViewer();
    vi.mocked(api.get).mockReset().mockImplementation(((url: string) => {
      if (url === "/api/automation-hosts") return Promise.resolve({ data: hosts });
      if (url === "/api/machines") return Promise.resolve({ data: [] });
      if (url === "/api/workspaces") {
        return Promise.resolve({ data: [{ id: "ws-1", name: "Acme renamed", slug: "acme" }, { id: "ws-2", name: "Globex", slug: "globex" }] });
      }
      return Promise.reject(new Error(`unexpected GET ${url}`));
    }) as typeof api.get);
  });

  it("lists each workspace token a host reported under the workspace's current name, with its manager and how it fared, and none for a host with no token", async () => {
    render(
      <QueryClientProvider client={new QueryClient()}>
        <AutomationHostsSection />
      </QueryClientProvider>,
    );

    const managers = within(await screen.findByRole("list", { name: "Secrets managers" }));
    await managers.findByText("Acme renamed");
    const items = managers.getAllByRole("listitem").map((item) => item.textContent);

    expect(items).toEqual(["Acme renamed1Password", "Globex1Password token rejected", "Another workspace1Password unreachable"]);
    expect(screen.getAllByRole("list", { name: "Secrets managers" })).toHaveLength(1);
  });
});
