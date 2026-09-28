import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { MemoryRouter, useLocation } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { ProjectSection } from "@/components/sidebar/ProjectSection";
import { useWorkspaceStore } from "@/stores/workspaceStore";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn(),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

// The real wrapper lazy-imports CreateDocForm (tiptap + yjs); under full-suite load that dynamic import can outlive findBy's timeout, so the test resolves it synchronously.
vi.mock("@/components/doc/LazyCreateDocForm", async () => ({
  LazyCreateDocForm: (await import("@/components/doc/CreateDocForm")).CreateDocForm,
}));

const projects = [
  { id: "p-1", name: "Backend", prefix: "BE", position: 0, created_at: "", updated_at: "" },
  { id: "p-2", name: "Frontend", prefix: "FE", position: 1, created_at: "", updated_at: "" },
];

const docsByProject: Record<string, { id: string; project_id: string; title: string }[]> = {
  "p-1": [{ id: "d-1", project_id: "p-1", title: "Runbook" }],
  "p-2": [],
};

const mockApi = (list: unknown[] = projects) => {
  vi.mocked(api.get).mockImplementation(async (url: string, config?: unknown) => {
    const projectId = (config as { params?: { project_id?: string } } | undefined)?.params?.project_id ?? "";
    if (url === "/api/projects") return { data: list };
    if (url === "/api/docs") return { data: docsByProject[projectId] ?? [] };
    return { data: [] };
  });
};

const LocationSpy = () => <div data-testid="location">{useLocation().pathname}</div>;

const renderSection = ({ path = "/inbox", collapsed = false, list = projects } = {}) => {
  mockApi(list);
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[path]}>
        <ContextAwareConfirmation.ConfirmationRoot />
        <ProjectSection collapsed={collapsed} />
        <LocationSpy />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  useWorkspaceStore.setState({ selectedProjectId: "" });
});

describe("ProjectSection", () => {
  it("lists one project's pages once, not a copy per project", async () => {
    renderSection();

    expect(await screen.findByRole("button", { name: /BE.*Backend/ })).toBeInTheDocument();
    expect(screen.getAllByRole("link", { name: "Settings" })).toHaveLength(1);
    expect(screen.getByRole("link", { name: "Board" })).toHaveAttribute("href", "/board/BE");
    expect(screen.getByRole("link", { name: "Interview" })).toHaveAttribute("href", "/projects/BE/interview");
    expect(await screen.findByRole("link", { name: "Runbook" })).toHaveAttribute("href", "/docs/BE/d-1");
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
    await user.click(await screen.findByRole("button", { name: "New project" }));

    expect(screen.getByTestId("location")).toHaveTextContent("/wizard/project/project");
  });

  it("with no projects, offers New project instead of a switcher", async () => {
    const user = userEvent.setup();
    renderSection({ list: [] });

    await user.click(await screen.findByRole("button", { name: "New project" }));
    expect(screen.getByTestId("location")).toHaveTextContent("/wizard/project/project");
    expect(screen.queryByRole("link", { name: "Board" })).not.toBeInTheDocument();
  });

  it("the + beside the switcher creates a doc in the current project", async () => {
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: "New doc in Backend" }));
    expect(await screen.findByLabelText("Title")).toBeInTheDocument();
    expect(await screen.findByRole("combobox", { name: "Project" })).toHaveTextContent("Backend");
  });

  it("collapsed rail: the switcher shows the prefix and every page is an icon row", async () => {
    renderSection({ collapsed: true });

    const trigger = await screen.findByRole("button", { name: "BE" });
    expect(trigger).toHaveAttribute("title", "Backend");
    expect(screen.getByRole("link", { name: "Board" })).toHaveAttribute("href", "/board/BE");
    expect(screen.queryByText("Project")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /New doc in/ })).not.toBeInTheDocument();
  });
});
