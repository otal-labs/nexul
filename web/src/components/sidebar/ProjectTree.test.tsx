import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { MemoryRouter, useLocation } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { ProjectTree } from "@/components/sidebar/ProjectTree";

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

const mockApi = () => {
  vi.mocked(api.get).mockImplementation(async (url: string, config?: unknown) => {
    const projectId = (config as { params?: { project_id?: string } } | undefined)?.params?.project_id ?? "";
    if (url === "/api/projects") return { data: projects };
    if (url === "/api/docs") return { data: docsByProject[projectId] ?? [] };
    return { data: [] };
  });
};

const LocationSpy = () => <div data-testid="location">{useLocation().pathname}</div>;

const renderTree = (collapsed = false) => {
  mockApi();
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <ContextAwareConfirmation.ConfirmationRoot />
        <ProjectTree collapsed={collapsed} />
        <LocationSpy />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  vi.mocked(api.post).mockReset();
});

describe("ProjectTree", () => {
  it("renders each project as a collapsed row, closed by default", async () => {
    renderTree();
    expect(await screen.findByRole("button", { name: /BE.*Backend/ })).toHaveAttribute("aria-expanded", "false");
    expect(screen.getByRole("button", { name: /FE.*Frontend/ })).toHaveAttribute("aria-expanded", "false");
  });

  it("expanding a project reveals Board, each doc title, and Settings — no intermediate Docs row", async () => {
    const user = userEvent.setup();
    renderTree();

    const toggle = await screen.findByRole("button", { name: /BE.*Backend/ });
    await user.click(toggle);
    expect(toggle).toHaveAttribute("aria-expanded", "true");

    // Both projects' links stay mounted (CSS-only open/close animation), so
    // scope by href like any other always-mounted collapsible panel.
    const boardLinks = screen.getAllByRole("link", { name: "Board" });
    expect(boardLinks.find((link) => link.getAttribute("href") === "/board/BE")).toBeTruthy();
    expect(screen.getByRole("link", { name: "Runbook" })).toHaveAttribute("href", "/docs/BE/d-1");
    const settingsLinks = screen.getAllByRole("link", { name: "Settings" });
    expect(settingsLinks.find((link) => link.getAttribute("href") === "/projects/BE/settings")).toBeTruthy();
    expect(screen.queryByText("Docs")).not.toBeInTheDocument();
  });

  it("collapses a project back on second click", async () => {
    const user = userEvent.setup();
    renderTree();

    const toggle = await screen.findByRole("button", { name: /BE.*Backend/ });
    await user.click(toggle);
    expect(toggle).toHaveAttribute("aria-expanded", "true");
    await user.click(toggle);
    expect(toggle).toHaveAttribute("aria-expanded", "false");
  });

  it("has no section header — no Work/Deploy/Manage label anywhere in the tree", async () => {
    renderTree();
    await screen.findByRole("button", { name: /BE.*Backend/ });
    expect(screen.queryByText(/^Work$/)).not.toBeInTheDocument();
    expect(screen.queryByText(/^Deploy$/)).not.toBeInTheDocument();
    expect(screen.queryByText(/^Manage$/)).not.toBeInTheDocument();
  });

  it("shows a persistent New project row that opens the project wizard", async () => {
    const user = userEvent.setup();
    renderTree();

    await user.click(await screen.findByRole("button", { name: "New project" }));
    expect(screen.getByTestId("location")).toHaveTextContent("/wizard/project/project");
  });

  it("hovering a project row reveals a + that creates a doc scoped to that project", async () => {
    const user = userEvent.setup();
    renderTree();

    await screen.findByRole("button", { name: /BE.*Backend/ });
    await user.click(screen.getByRole("button", { name: "New doc in Backend" }));
    expect(await screen.findByLabelText("Title")).toBeInTheDocument();
    expect(await screen.findByRole("combobox", { name: "Project" })).toHaveTextContent("Backend");
  });

  it("collapsed rail: project rows show the prefix and toggle the same expand state, every row centered", async () => {
    const user = userEvent.setup();
    renderTree(true);

    const toggle = await screen.findByRole("button", { name: "BE" });
    expect(toggle).toHaveAttribute("title", "Backend");
    expect(toggle.className).toMatch(/justify-center/);
    expect(screen.queryByRole("link", { name: /Board/ })).not.toBeInTheDocument();

    await user.click(toggle);
    const board = screen.getByRole("link", { name: "Board" });
    expect(board).toHaveAttribute("href", "/board/BE");
    expect(board.className).toMatch(/justify-center/);
    expect(screen.getByRole("link", { name: "Runbook" }).className).toMatch(/justify-center/);
    expect(screen.getByRole("link", { name: "Settings" }).className).toMatch(/justify-center/);
    expect(screen.queryByRole("button", { name: /New doc in/ })).not.toBeInTheDocument();
  });
});
