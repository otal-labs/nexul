import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ContextAwareConfirmation } from "react-confirm";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ProjectSettingsPage } from "@/pages/ProjectSettingsPage";
import { api } from "@/api/client";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn(),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const project = { id: "p-1", name: "Backend", prefix: "BE", position: 0, created_at: "", updated_at: "" };

const categories = [{ id: "c-1", project_id: "p-1", name: "Sprint 1", position: 0, created_at: "", updated_at: "" }];

const renderPage = (initialEntry = "/projects/p-1/settings") => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <ContextAwareConfirmation.ConfirmationRoot />
      <MemoryRouter initialEntries={[initialEntry]}>
        <Routes>
          <Route path="/projects/:projectId/settings" element={<ProjectSettingsPage />} />
          <Route path="/" element={<div>home</div>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  vi.mocked(api.patch).mockReset();
  vi.mocked(api.delete).mockReset();
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/projects") return { data: [project] };
    if (url === "/api/projects/p-1") return { data: project };
    if (url === "/api/projects/p-1/impact") return { data: { tickets: 0, repos: 0, services: 0 } };
    if (url === "/api/categories") return { data: categories };
    return { data: [] };
  });
});

describe("ProjectSettingsPage", () => {
  it("shows an error state", async () => {
    vi.mocked(api.get).mockRejectedValue(new Error("boom"));
    renderPage();
    expect(await screen.findByText("Something went wrong")).toBeInTheDocument();
  });

  it("renders the page context line and the full section nav", async () => {
    renderPage();

    expect(await screen.findByRole("heading", { name: "Backend" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Categories" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Repositories" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Services" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Board" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "T3 pairing" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Danger zone" })).toBeInTheDocument();
  });

  it("defaults to the General section", async () => {
    renderPage();

    expect(await screen.findByRole("heading", { name: "General" })).toBeInTheDocument();
    expect(screen.getByText("Used in ticket IDs")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "General" })).toHaveAttribute("aria-current", "page");
  });

  it("switches sections through the URL section param when a nav link is clicked", async () => {
    const user = userEvent.setup();
    renderPage();

    await screen.findByRole("heading", { name: "General" });
    await user.click(screen.getByRole("link", { name: "Categories" }));

    expect(await screen.findByText("Sprint 1")).toBeInTheDocument();
    expect(screen.queryByText("Used in ticket IDs")).not.toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Categories" })).toHaveAttribute("aria-current", "page");
  });

  it("deep-links straight into a non-default section", async () => {
    renderPage("/projects/p-1/settings?section=categories");

    expect(await screen.findByText("Sprint 1")).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "General" })).not.toBeInTheDocument();
  });

  it("General renders the project name, prefix, and an icon control, and saves an icon choice", async () => {
    const user = userEvent.setup();
    vi.mocked(api.patch).mockResolvedValue({ data: { ...project, icon: "Rocket" } });
    renderPage();

    await screen.findByRole("heading", { name: "General" });
    expect(screen.getAllByText("Backend").length).toBeGreaterThan(0);
    expect(screen.getAllByText("BE").length).toBeGreaterThan(0);

    await user.click(screen.getByRole("button", { name: "Change project icon" }));
    await user.click(await screen.findByRole("button", { name: "Rocket" }));

    expect(api.patch).toHaveBeenCalledWith("/api/projects/p-1", { name: "Backend", icon: "Rocket" });
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
    expect(api.patch).toHaveBeenCalledWith("/api/projects/p-1", { name: "Platform", icon: "" });
  });

  it("blocks removal of a project with affected work from the danger zone", async () => {
    const user = userEvent.setup();
    vi.mocked(api.get).mockImplementation(async (url: string) => {
      if (url === "/api/projects") return { data: [project] };
      if (url === "/api/projects/p-1") return { data: project };
      if (url === "/api/projects/p-1/impact") return { data: { tickets: 3, repos: 1, services: 0 } };
      return { data: [] };
    });
    renderPage("/projects/p-1/settings?section=danger");

    await user.click(await screen.findByRole("button", { name: "Remove project" }));
    expect(await screen.findByText(/still has affected work/i)).toBeInTheDocument();
    expect(screen.getByText(/3 tickets/)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Got it" }));

    expect(api.delete).not.toHaveBeenCalled();
  });

  it("removes an empty project from the danger zone after confirming", async () => {
    const user = userEvent.setup();
    vi.mocked(api.delete).mockResolvedValue({ data: undefined });
    renderPage("/projects/p-1/settings?section=danger");

    await user.click(await screen.findByRole("button", { name: "Remove project" }));
    expect(await screen.findByText(/can be removed/i)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Remove" }));
    expect(api.delete).toHaveBeenCalledWith("/api/projects/p-1");
  });
});
