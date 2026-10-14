import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { ProjectSettingsPage } from "@/pages/ProjectSettingsPage";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { asViewer, memberWho } from "@/test/viewer";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn(),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const project = { id: "p-1", name: "Backend", prefix: "BE", position: 0, workspace_id: "ws-1", setup: { finished: true, steps: {} }, created_at: "", updated_at: "" };
const category = { id: "c-1", project_id: "p-1", name: "Sprint 1", position: 0, color: "", created_at: "", updated_at: "" };
const status = { id: "open", name: "Open", position: 0, kind: "backlog", icon: "", created_at: "", updated_at: "" };
const ticketType = { id: "task", name: "task", position: 0, color: "", body_template: "", created_at: "", updated_at: "" };
const repo = { owner: "acme", name: "api", full_name: "acme/api", connector_id: "github", role: "app" };

// Everything a member needs to read these pages and nothing they write.
const READS = ["projects:read", "tickets:read", "stacks:read"];

const serve = (showLocked: boolean) => {
  vi.mocked(api.post).mockResolvedValue({ data: { bug: "orange" } });
  return vi.mocked(api.get).mockImplementation(
    asViewer(memberWho(READS, showLocked), (url: string) => {
      if (url === "/api/projects") return [project];
      if (url === "/api/projects/p-1") return project;
      if (url === "/api/categories") return [category];
      if (url === "/api/statuses") return [status];
      if (url === "/api/ticket-types") return [ticketType];
      if (url === "/api/tickets/labels") return ["bug"];
      if (url === "/api/projects/p-1/repos") return [repo];
      return [];
    }),
  );
};

const renderSettings = (path: string) =>
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter initialEntries={[`/acme/projects/p-1/settings/${path}`]}>
        <Routes>
          <Route path="/acme/projects/:projectId/settings/:section?/:tab?" element={<ProjectSettingsPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );

// Each page of project settings, text a reader still sees, every control that writes, and the row menus whose items do.
const pages = [
  { path: "general", anchor: "Backend", controls: ["Rename project", "Open the wizard"], menus: [] },
  { path: "categories", anchor: "Sprint 1", controls: ["New category", "Reorder Sprint 1"], menus: ["Actions for Sprint 1"] },
  { path: "repositories", anchor: "acme/api", controls: ["Add repo", "Remove acme/api"], menus: [] },
  { path: "services", anchor: "Attached, not deployed yet", controls: ["New service", "Deploy this repository"], menus: [] },
  { path: "board", anchor: "Open", controls: ["Add backlog status"], menus: ["Actions for Open"] },
  { path: "board/types", anchor: "task", controls: ["New ticket type", "Color for task"], menus: ["Actions for task"] },
  { path: "board/labels", anchor: "bug", controls: ["Color for label bug"], menus: [], dot: ".bg-orange-500" },
];

const control = (name: string) => screen.queryByRole("button", { name }) ?? screen.queryByRole("link", { name });

describe("project settings for a member who only reads", () => {
  beforeEach(() => {
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
    vi.mocked(api.get).mockReset();
  });

  it.each(pages)("$path shows what they may read and nothing that writes", async ({ path, anchor, controls, menus, dot }) => {
    serve(false);
    renderSettings(path);

    expect((await screen.findAllByText(anchor)).length).toBeGreaterThan(0);
    for (const name of [...controls, ...menus]) expect(control(name), name).not.toBeInTheDocument();
    if (dot) await vi.waitFor(() => expect(document.querySelector(dot)).toBeInTheDocument());
  });

  it.each(pages)("$path locks every control that writes when a role of theirs shows locked actions", async ({ path, anchor, controls, menus, dot }) => {
    serve(true);
    renderSettings(path);

    expect((await screen.findAllByText(anchor)).length).toBeGreaterThan(0);
    for (const name of controls) expect(await screen.findByRole("button", { name }), name).toHaveAttribute("aria-disabled", "true");
    for (const name of menus) expect(await screen.findByRole("button", { name }), name).not.toHaveAttribute("aria-disabled");
    if (dot) await vi.waitFor(() => expect(document.querySelector(dot)).toBeInTheDocument());
  });

  it("locks a row's menu items inside its menu", async () => {
    serve(true);
    renderSettings("categories");
    await userEvent.setup().click(await screen.findByRole("button", { name: "Actions for Sprint 1" }));

    for (const name of ["Edit", "Move up", "Delete"]) {
      expect(await screen.findByRole("button", { name }), name).toHaveAttribute("aria-disabled", "true");
    }
  });
});
