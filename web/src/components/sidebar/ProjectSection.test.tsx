import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, useLocation } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { ProjectSection } from "@/components/sidebar/ProjectSection";
import { projectFollower } from "@/hooks/ProjectHooks";
import type { ProjectSetup } from "@/models/Project";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { followFrame } from "@/test/followFrame";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn(),
}));

const setUp: ProjectSetup = { finished: true, steps: {} };

const projects = [
  { id: "p-1", name: "Backend", prefix: "BE", position: 0, workspace_id: "ws-1", setup: setUp, created_at: "", updated_at: "" },
  { id: "p-2", name: "Frontend", prefix: "FE", position: 1, workspace_id: "ws-1", setup: setUp, created_at: "", updated_at: "" },
];

// A project the wizard made on some other device: the server's record is all this browser has to go on.
const inSetup: (typeof projects)[number] = { ...projects[0]!, setup: { finished: false, steps: { project: "done", repository: "skipped" } } };

const ownerPermissions = ["docs:read", "docs:write", "memories:read", "projects:read", "projects:write", "tickets:read"];

interface MeExtra {
  restricted?: boolean;
  projects?: { project_id: string; actions: string[] }[];
}

const mockApi = (list: unknown[] = projects, permissions: string[] = ownerPermissions, me: MeExtra = {}) => {
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/projects") return { data: list };
    if (url === "/api/workspaces/ws-1/me") return { data: { role_name: "Member", permissions, ...me } };
    return { data: [] };
  });
};

const LocationSpy = () => <div data-testid="location">{useLocation().pathname}</div>;

const renderSection = ({ path = "/acme/inbox", collapsed = false, list = projects, permissions = ownerPermissions, me = {} as MeExtra } = {}) => {
  mockApi(list, permissions, me);
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const view = render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[path]}>
        <ProjectSection collapsed={collapsed} />
        <LocationSpy />
      </MemoryRouter>
    </QueryClientProvider>,
  );
  return { ...view, client };
};

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1", selectedProjectId: "" });
});

describe("ProjectSection", () => {
  it("Continue setup resumes Environment when a detected key still needs its value", async () => {
    const user = userEvent.setup();
    renderSection({ list: [{ ...projects[0]!, setup: { finished: false, steps: { project: "done", repository: "done", service: "done" }, stack_id: "stack-1", env_keys: ["PORT"] } }] });

    await user.click(await screen.findByRole("link", { name: "Continue setup" }));

    expect(screen.getByTestId("location")).toHaveTextContent("/acme/wizard/project/env");
  });

  it("lists one project's pages once, not a copy per project", async () => {
    renderSection();

    expect(await screen.findByRole("button", { name: /Backend/ })).toBeInTheDocument();
    expect(screen.getAllByRole("link", { name: "Settings" })).toHaveLength(1);
    expect(screen.getByRole("link", { name: "Board" })).toHaveAttribute("href", "/acme/board/BE");
    expect(screen.getByRole("link", { name: "Interview" })).toHaveAttribute("href", "/acme/projects/BE/interview");
    expect(screen.getByRole("link", { name: "Docs" })).toHaveAttribute("href", "/acme/docs");
    expect(screen.getByRole("link", { name: "Memories" })).toHaveAttribute("href", "/acme/memories");
    expect(screen.queryByRole("link", { name: "Runbook" })).not.toBeInTheDocument();
    expect(screen.queryByText("Frontend")).not.toBeInTheDocument();
  });

  it("while setup is open, offers only Continue setup, at the first step neither done nor skipped", async () => {
    renderSection({ list: [inSetup] });

    expect(await screen.findByRole("link", { name: "Continue setup" })).toHaveAttribute(
      "href",
      "/acme/wizard/project/service?project=p-1",
    );
    for (const page of ["Board", "Interview", "Docs", "Memories", "Settings"]) {
      expect(screen.queryByRole("link", { name: page })).not.toBeInTheDocument();
    }
  });

  it("tells a member who can't change the project that it is being set up, with nothing to press", async () => {
    renderSection({ list: [inSetup], permissions: ["projects:read", "tickets:read"] });

    expect(await screen.findByText("Being set up")).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Continue setup" })).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Board" })).not.toBeInTheDocument();
  });

  it("lists the project's pages once someone else finishes its setup", async () => {
    const { client } = renderSection({ list: [inSetup] });
    await screen.findByRole("link", { name: "Continue setup" });

    await act(() =>
      followFrame(projectFollower, "project.setup_changed", { project_id: "p-1", workspace_id: "ws-1", setup: setUp }, client),
    );

    expect(await screen.findByRole("link", { name: "Board" })).toHaveAttribute("href", "/acme/board/BE");
    expect(screen.queryByRole("link", { name: "Continue setup" })).not.toBeInTheDocument();
  });

  it("follows the project in the URL and remembers it", async () => {
    renderSection({ path: "/acme/projects/FE/settings" });

    expect(await screen.findByRole("button", { name: /Frontend/ })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Settings" })).toHaveAttribute("href", "/acme/projects/FE/settings");
    expect(useWorkspaceStore.getState().selectedProjectId).toBe("p-2");
  });

  it("falls back to the last project visited off project pages", async () => {
    useWorkspaceStore.setState({ selectedProjectId: "p-2" });
    renderSection({ path: "/acme/runners" });

    expect(await screen.findByRole("button", { name: /Frontend/ })).toBeInTheDocument();
  });

  it("switching project keeps you on the same project page", async () => {
    const user = userEvent.setup();
    renderSection({ path: "/acme/projects/BE/settings" });

    await user.click(await screen.findByRole("button", { name: /Backend/ }));
    await user.click(await screen.findByRole("button", { name: /Frontend/ }));

    expect(screen.getByTestId("location")).toHaveTextContent("/acme/projects/FE/settings");
    expect(useWorkspaceStore.getState().selectedProjectId).toBe("p-2");
  });

  it("switching from a non-project page lands on the new project's board", async () => {
    const user = userEvent.setup();
    renderSection({ path: "/acme/runners" });

    await user.click(await screen.findByRole("button", { name: /Backend/ }));
    await user.click(await screen.findByRole("button", { name: /Frontend/ }));

    expect(screen.getByTestId("location")).toHaveTextContent("/acme/board/FE");
  });

  it("the switcher's New project opens the project wizard", async () => {
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: /Backend/ }));
    await user.click(await screen.findByRole("button", { name: "New project" }));

    expect(screen.getByTestId("location")).toHaveTextContent("/acme/wizard/project/project");
  });

  it("with no projects, offers New project instead of a switcher", async () => {
    const user = userEvent.setup();
    renderSection({ list: [] });

    await user.click(await screen.findByRole("button", { name: "New project" }));
    expect(screen.getByTestId("location")).toHaveTextContent("/acme/wizard/project/project");
    expect(screen.queryByRole("link", { name: "Board" })).not.toBeInTheDocument();
  });

  it("the menu lists every project and marks only the current one", async () => {
    const user = userEvent.setup();
    renderSection({ path: "/acme/projects/FE/settings" });

    await user.click(await screen.findByRole("button", { name: /Frontend/ }));

    expect(await screen.findByRole("button", { name: /Backend/ })).not.toHaveAttribute("aria-current");
    const items = screen.getAllByRole("button", { current: true });
    expect(items).toHaveLength(1);
    expect(items[0]).toHaveTextContent("Frontend");
  });

  it("has no + beside the switcher; New project only appears inside the menu", async () => {
    const user = userEvent.setup();
    renderSection();

    const trigger = await screen.findByRole("button", { name: /Backend/ });
    expect(screen.queryByRole("button", { name: /new/i })).not.toBeInTheDocument();

    await user.click(trigger);
    expect(await screen.findByRole("button", { name: "New project" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /new doc/i })).not.toBeInTheDocument();
  });

  it("a viewer who may read but not create gets no New project", async () => {
    const user = userEvent.setup();
    renderSection({ permissions: ["projects:read", "tickets:read"] });

    await user.click(await screen.findByRole("button", { name: /Backend/ }));
    expect(await screen.findByRole("button", { name: /Frontend/ })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "New project" })).not.toBeInTheDocument();
  });

  it("collapsed rail: the switcher shows the prefix and every page is an icon row", async () => {
    renderSection({ collapsed: true });

    expect(await screen.findByRole("button", { name: "Backend" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Board" })).toHaveAttribute("href", "/acme/board/BE");
    expect(screen.queryByText("Project")).not.toBeInTheDocument();
  });

  it("shows a Restricted member the project section with what they hold on the project in view", async () => {
    const me = {
      restricted: true,
      projects: [
        { project_id: "p-1", actions: ["tickets:read", "tickets:write"] },
        { project_id: "p-2", actions: ["docs:read", "projects:read"] },
      ],
    };
    renderSection({ path: "/acme/board/BE", permissions: ["chat:read"], me });

    expect(await screen.findByRole("button", { name: /Backend/ })).toBeInTheDocument();
    expect(await screen.findByRole("link", { name: "Board" })).toHaveAttribute("href", "/acme/board/BE");
    expect(screen.queryByRole("link", { name: "Docs" })).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Settings" })).not.toBeInTheDocument();
  });
});
