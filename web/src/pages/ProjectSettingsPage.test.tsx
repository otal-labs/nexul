import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ContextAwareConfirmation } from "react-confirm";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ProjectSettingsPage } from "@/pages/ProjectSettingsPage";
import { ticketCounts } from "@/test/ticketCounts";
import { api } from "@/api/client";
import { useWorkspaceStore } from "@/stores/workspaceStore";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn(),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const project = { id: "p-1", name: "Backend", prefix: "BE", position: 0, setup: { finished: true, steps: {} }, created_at: "", updated_at: "" };

const categories = [{ id: "c-1", project_id: "p-1", name: "Sprint 1", position: 0, created_at: "", updated_at: "" }];

const renderPage = (initialEntry = "/acme/projects/p-1/settings") => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <ContextAwareConfirmation.ConfirmationRoot />
      <MemoryRouter initialEntries={[initialEntry]}>
        <Routes>
          <Route path="/acme/projects/:projectId/settings/:section?" element={<ProjectSettingsPage />} />
          <Route path="/" element={<div>home</div>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

const owner = { role_name: "Owner", roles: [{ id: "role-owner", name: "Owner", is_owner: true }], permissions: [] };

beforeEach(() => {
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  vi.mocked(api.get).mockReset();
  vi.mocked(api.patch).mockReset();
  vi.mocked(api.delete).mockReset();
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/workspaces/ws-1/me") return { data: owner };
    if (url === "/api/permissions/explain") return { data: { effective: [] } };
    if (url === "/api/projects") return { data: [project] };
    if (url === "/api/projects/p-1") return { data: project };
    if (url === "/api/projects/p-1/impact") return { data: { tickets: 0, repos: 0, services: 0 } };
    if (url === "/api/categories") return { data: categories };
    if (url === "/api/projects/p-1/ticket-counts") return { data: ticketCounts() };
    return { data: [] };
  });
});

describe("ProjectSettingsPage", () => {
  const renderMissingProject = (permissions: string[]) => {
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
    vi.mocked(api.get).mockImplementation(async (url: string) => {
      if (url === "/api/workspaces/ws-1/me") return { data: { role_name: "Member", permissions } };
      return { data: [] };
    });
    renderPage("/acme/projects/NOPE/settings");
  };

  it("offers the board from a missing project to a member who can read tickets", async () => {
    renderMissingProject(["tickets:read"]);
    expect(await screen.findByRole("link", { name: "Go to your board" })).toHaveAttribute("href", "/acme/board");
  });

  it("does not offer the board from a missing project without tickets:read", async () => {
    renderMissingProject(["projects:read"]);
    expect(await screen.findByText("Project not found")).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Go to your board" })).not.toBeInTheDocument();
  });

  it("shows an error state", async () => {
    vi.mocked(api.get).mockRejectedValue(new Error("boom"));
    renderPage();
    expect(await screen.findByText("Something went wrong")).toBeInTheDocument();
  });

  it("names the project in the breadcrumb and renders the full section nav", async () => {
    renderPage();

    expect(await screen.findByRole("link", { name: "Backend" })).toHaveAttribute("href", "/acme/board/BE");
    expect(screen.getByText("Settings")).toBeInTheDocument();
    for (const name of ["Statuses", "Categories", "Ticket types", "Labels", "Repositories", "Services", "Danger zone"]) {
      expect(screen.getByRole("link", { name })).toBeInTheDocument();
    }
    expect(screen.queryByRole("link", { name: "Board" })).not.toBeInTheDocument();
  });

  it("titles the page by the open section", async () => {
    renderPage("/acme/projects/p-1/settings/labels");

    expect(await screen.findByRole("heading", { level: 1, name: "Labels" })).toBeInTheDocument();
  });

  it("lands an old board path on General, the way any unknown section does", async () => {
    renderPage("/acme/projects/p-1/settings/board");

    expect(await screen.findByRole("heading", { level: 1, name: "General" })).toBeInTheDocument();
  });

  it("defaults to the General section", async () => {
    renderPage();

    expect(await screen.findByText(/It can't change/)).toBeInTheDocument();
    expect(screen.getByRole("heading", { level: 1, name: "General" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "General" })).toHaveAttribute("aria-current", "page");
  });

  it("switches sections through the URL section param when a nav link is clicked", async () => {
    const user = userEvent.setup();
    renderPage();

    await screen.findByText(/It can't change/);
    await user.click(screen.getByRole("link", { name: "Categories" }));

    expect(await screen.findByText("Sprint 1")).toBeInTheDocument();
    expect(screen.queryByText(/It can't change/)).not.toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Categories" })).toHaveAttribute("aria-current", "page");
  });

  it("deep-links straight into a non-default section", async () => {
    renderPage("/acme/projects/p-1/settings/categories");

    expect(await screen.findByText("Sprint 1")).toBeInTheDocument();
    expect(screen.getByRole("heading", { level: 1, name: "Categories" })).toBeInTheDocument();
  });

  it("General renames the project inline", async () => {
    const user = userEvent.setup();
    vi.mocked(api.patch).mockResolvedValue({ data: { ...project, name: "Platform" } });
    renderPage();

    await user.click(await screen.findByRole("button", { name: "Rename project" }));
    const input = screen.getByLabelText("Project name");
    await user.clear(input);
    await user.type(input, "Platform");
    await user.keyboard("{Enter}");
    expect(api.patch).toHaveBeenCalledWith("/api/projects/p-1", { name: "Platform" });
  });

  it("locks Remove project with the rule's sentence when a rule on this project denies it", async () => {
    const sentence = "Denied on project Backend for role Member";
    const defaultGet = vi.mocked(api.get).getMockImplementation()!;
    vi.mocked(api.get).mockImplementation(async (url: string) => {
      if (url === "/api/workspaces/ws-1/me") {
        return { data: { role_name: "Member", permissions: [], projects: [{ project_id: "p-1", actions: ["projects:delete"] }], show_locked_actions: true } };
      }
      if (url === "/api/permissions/explain") return { data: { effective: [{ action: "projects:delete", allowed: false, source: { sentence } }] } };
      return defaultGet(url);
    });
    renderPage("/acme/projects/p-1/settings/danger");

    await vi.waitFor(() => expect(screen.getByRole("button", { name: "Remove project" })).toHaveAttribute("aria-disabled", "true"));
    expect(screen.getByRole("button", { name: "Remove project" })).toHaveAccessibleDescription(sentence);
  });

  it("blocks removal of a project with affected work from the danger zone", async () => {
    const user = userEvent.setup();
    const defaultGet = vi.mocked(api.get).getMockImplementation()!;
    vi.mocked(api.get).mockImplementation(async (url: string) => {
      if (url === "/api/projects/p-1/impact") return { data: { tickets: 3, repos: 1, services: 0 } };
      return defaultGet(url);
    });
    renderPage("/acme/projects/p-1/settings/danger");

    await user.click(await screen.findByRole("button", { name: "Remove project" }));
    const dialog = await screen.findByRole("dialog");
    expect(within(dialog).getByText(/still has work in it/i)).toBeInTheDocument();
    expect(within(dialog).getByText(/3 tickets/)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Got it" }));

    expect(api.delete).not.toHaveBeenCalled();
  });

  it("removes an empty project from the danger zone after confirming", async () => {
    const user = userEvent.setup();
    vi.mocked(api.delete).mockResolvedValue({ data: undefined });
    renderPage("/acme/projects/p-1/settings/danger");

    await user.click(await screen.findByRole("button", { name: "Remove project" }));
    const dialog = await screen.findByRole("dialog");
    expect(within(dialog).getByText(/can't be undone/i)).toBeInTheDocument();
    await user.click(within(dialog).getByRole("button", { name: "Remove project" }));
    expect(api.delete).toHaveBeenCalledWith("/api/projects/p-1");
  });
});
