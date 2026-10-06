import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ProjectPeopleAccessSection } from "@/components/settings/ProjectPeopleAccessSection";
import type { Project } from "@/models/Project";
import { useWorkspaceStore } from "@/stores/workspaceStore";

const mocks = vi.hoisted(() => ({ get: vi.fn() }));
vi.mock("@/api/client", () => ({ api: mocks, errorMessage: () => "You can't manage members in OTAL" }));

const project = { id: "p-web", name: "Web", prefix: "WEB" } as Project;
const catalog = [
  { value: "tickets:read", label: "Read tickets", domain: "tickets", action: "read", area: "project" },
  { value: "tickets:write", label: "Create and update tickets", domain: "tickets", action: "write", area: "project" },
  { value: "docs:read", label: "Read docs", domain: "docs", action: "read", area: "project" },
  { value: "docs:write", label: "Create and update docs", domain: "docs", action: "write", area: "project" },
];

const renderSection = (access: Promise<unknown>) => {
  mocks.get.mockImplementation((url: string) => {
    if (url === "/api/projects/p-web/access") return access;
    if (url === "/api/permissions/catalog") return Promise.resolve({ data: { permissions: catalog } });
    if (url === "/api/workspaces/ws-1/people") return Promise.resolve({ data: { people: [{ user_id: "u-bob", login: "bob", display_name: "Bob", avatar_url: "" }] } });
    return Promise.reject(new Error(`unexpected GET ${url}`));
  });
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <ProjectPeopleAccessSection project={project} />
    </QueryClientProvider>,
  );
};

describe("ProjectPeopleAccessSection", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  });

  it("shows the refusal when the list can't be read", async () => {
    renderSection(Promise.reject(new Error("forbidden")));
    expect(await screen.findByText("You can't manage members in OTAL")).toBeInTheDocument();
  });

  it("says so when no Restricted member can open the project", async () => {
    renderSection(Promise.resolve({ data: { access: [] } }));
    expect(await screen.findByText("No restricted member can open Web.")).toBeInTheDocument();
    expect(screen.getByText("Change access from Team.")).toBeInTheDocument();
  });

  it("lists each Restricted member with a summary of what they hold", async () => {
    renderSection(Promise.resolve({ data: { access: [{ user_id: "u-bob", name: "Bob", actions: ["tickets:read", "tickets:write", "docs:read"] }] } }));
    expect(await screen.findByText("Bob")).toBeInTheDocument();
    expect(await screen.findByText("tickets Write · docs Read")).toBeInTheDocument();
  });
});
