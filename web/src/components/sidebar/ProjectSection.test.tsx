import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, useLocation } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { ProjectSection } from "@/components/sidebar/ProjectSection";
import { useWorkspaceStore } from "@/stores/workspaceStore";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn(),
}));

const projects = [
  { id: "p-1", name: "Backend", prefix: "BE", position: 0, created_at: "", updated_at: "" },
  { id: "p-2", name: "Frontend", prefix: "FE", position: 1, created_at: "", updated_at: "" },
];

const ownerPermissions = ["docs:read", "docs:write", "memories:read", "projects:read", "projects:write", "tickets:read"];

const mockApi = (list: unknown[] = projects, permissions: string[] = ownerPermissions) => {
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/projects") return { data: list };
    if (url === "/api/workspaces/ws-1/me") return { data: { role_name: "Member", permissions } };
    return { data: [] };
  });
};

const LocationSpy = () => <div data-testid="location">{useLocation().pathname}</div>;

const renderSection = ({ path = "/inbox", collapsed = false, list = projects, permissions = ownerPermissions } = {}) => {
  mockApi(list, permissions);
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[path]}>
        <ProjectSection collapsed={collapsed} />
        <LocationSpy />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1", selectedProjectId: "" });
});

describe("ProjectSection", () => {
  it("lists one project's pages once, not a copy per project", async () => {
    renderSection();

    expect(await screen.findByRole("button", { name: /BE.*Backend/ })).toBeInTheDocument();
    expect(screen.getAllByRole("link", { name: "Settings" })).toHaveLength(1);
    expect(screen.getByRole("link", { name: "Board" })).toHaveAttribute("href", "/board/BE");
    expect(screen.getByRole("link", { name: "Interview" })).toHaveAttribute("href", "/projects/BE/interview");
    expect(screen.getByRole("link", { name: "Docs" })).toHaveAttribute("href", "/docs");
    expect(screen.getByRole("link", { name: "Memories" })).toHaveAttribute("href", "/memories");
    expect(screen.queryByRole("link", { name: "Runbook" })).not.toBeInTheDocument();
    expect(screen.queryByText("Frontend")).not.toBeInTheDocument();
  });

  it("follows the project in the URL and remembers it", async () => {
    renderSection({ path: "/projects/FE/settings" });

    expect(await screen.findByRole("button", { name: /FE.*Frontend/ })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Settings" })).toHaveAttribute("href", "/projects/FE/settings");
    expect(useWorkspaceStore.getState().selectedProjectId).toBe("p-2");
  });

  it("falls back to the last project visited off project pages", async () => {
    useWorkspaceStore.setState({ selectedProjectId: "p-2" });
    renderSection({ path: "/runners" });

    expect(await screen.findByRole("button", { name: /FE.*Frontend/ })).toBeInTheDocument();
  });

  it("switching project keeps you on the same project page", async () => {
    const user = userEvent.setup();
    renderSection({ path: "/projects/BE/settings" });

    await user.click(await screen.findByRole("button", { name: /BE.*Backend/ }));
    await user.click(await screen.findByRole("button", { name: /FE.*Frontend/ }));

    expect(screen.getByTestId("location")).toHaveTextContent("/projects/FE/settings");
    expect(useWorkspaceStore.getState().selectedProjectId).toBe("p-2");
  });

  it("switching from a non-project page lands on the new project's board", async () => {
    const user = userEvent.setup();
    renderSection({ path: "/runners" });

    await user.click(await screen.findByRole("button", { name: /BE.*Backend/ }));
    await user.click(await screen.findByRole("button", { name: /FE.*Frontend/ }));

    expect(screen.getByTestId("location")).toHaveTextContent("/board/FE");
  });

  it("the switcher's New project opens the project wizard", async () => {
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: /BE.*Backend/ }));
    await user.click(await screen.findByRole("button", { name: "New Project" }));

    expect(screen.getByTestId("location")).toHaveTextContent("/wizard/project/project");
  });

  it("with no projects, offers New project instead of a switcher", async () => {
    const user = userEvent.setup();
    renderSection({ list: [] });

    await user.click(await screen.findByRole("button", { name: "New project" }));
    expect(screen.getByTestId("location")).toHaveTextContent("/wizard/project/project");
    expect(screen.queryByRole("link", { name: "Board" })).not.toBeInTheDocument();
  });

  it("the menu lists every project and marks only the current one", async () => {
    const user = userEvent.setup();
    renderSection({ path: "/projects/FE/settings" });

    await user.click(await screen.findByRole("button", { name: /FE.*Frontend/ }));

    expect(await screen.findByRole("button", { name: /BE.*Backend/ })).not.toHaveAttribute("aria-current");
    const items = screen.getAllByRole("button", { current: true });
    expect(items).toHaveLength(1);
    expect(items[0]).toHaveTextContent("Frontend");
  });

  it("has no + beside the switcher; New Project only appears inside the menu", async () => {
    const user = userEvent.setup();
    renderSection();

    const trigger = await screen.findByRole("button", { name: /BE.*Backend/ });
    expect(screen.queryByRole("button", { name: /new/i })).not.toBeInTheDocument();

    await user.click(trigger);
    expect(await screen.findByRole("button", { name: "New Project" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /new doc/i })).not.toBeInTheDocument();
  });

  it("a viewer who may read but not create gets no New Project", async () => {
    const user = userEvent.setup();
    renderSection({ permissions: ["projects:read", "tickets:read"] });

    await user.click(await screen.findByRole("button", { name: /BE.*Backend/ }));
    expect(await screen.findByRole("button", { name: /FE.*Frontend/ })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "New Project" })).not.toBeInTheDocument();
  });

  it("collapsed rail: the switcher shows the prefix and every page is an icon row", async () => {
    renderSection({ collapsed: true });

    const trigger = await screen.findByRole("button", { name: "BE" });
    expect(trigger).toHaveAttribute("title", "Backend");
    expect(screen.getByRole("link", { name: "Board" })).toHaveAttribute("href", "/board/BE");
    expect(screen.queryByText("Project")).not.toBeInTheDocument();
  });
});
