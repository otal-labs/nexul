import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ContextAwareConfirmation } from "react-confirm";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ProjectRepos } from "@/components/project/ProjectRepos";
import { api } from "@/api/client";
import type { RepoRef } from "@/models/Project";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn(),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const repos: RepoRef[] = [
  { owner: "onik97", name: "nexul", full_name: "onik97/nexul", connector_id: "github" },
];

const renderRepos = (list: RepoRef[] = repos) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  vi.mocked(api.get).mockResolvedValue({ data: list });
  return render(
    <QueryClientProvider client={client}>
      <ContextAwareConfirmation.ConfirmationRoot />
      <ProjectRepos projectId="p-1" />
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  vi.mocked(api.post).mockReset();
  vi.mocked(api.delete).mockReset();
});

describe("ProjectRepos", () => {
  it("lists repos and adds a new one through the dialog", async () => {
    const user = userEvent.setup();
    vi.mocked(api.post).mockResolvedValue({ data: undefined });
    renderRepos();

    expect(await screen.findByText("onik97/nexul")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Add repo" }));
    await user.type(screen.getByLabelText("Repository owner"), "acme");
    await user.type(screen.getByLabelText("Repository name"), "api");
    expect(screen.getByRole("combobox", { name: "Connector" })).toHaveTextContent("GitHub");
    await user.click(screen.getByRole("button", { name: "Add repository" }));
    expect(api.post).toHaveBeenCalledWith("/api/projects/p-1/repos", {
      owner: "acme",
      name: "api",
      connector_id: "github",
    });
  });

  it("shows an empty state when there are no repos", async () => {
    renderRepos([]);
    expect(await screen.findByText("No repositories associated yet.")).toBeInTheDocument();
  });

  it("does not show the empty state while repos are still loading", () => {
    vi.mocked(api.get).mockReturnValue(new Promise(() => {}));
    renderRepos([]);
    expect(screen.queryByText("No repositories associated yet.")).not.toBeInTheDocument();
  });

  it("does not submit empty repo fields", async () => {
    const user = userEvent.setup();
    renderRepos();

    await user.click(screen.getByRole("button", { name: "Add repo" }));
    await user.click(screen.getByRole("button", { name: "Add repository" }));
    expect(await screen.findByText("Owner is required")).toBeInTheDocument();
    expect(screen.getByText("Repo name is required")).toBeInTheDocument();
    expect(api.post).not.toHaveBeenCalled();
  });

  it("removes a repo", async () => {
    const user = userEvent.setup();
    vi.mocked(api.delete).mockResolvedValue({ data: undefined });
    renderRepos();

    await user.click(await screen.findByRole("button", { name: "Remove onik97/nexul" }));
    expect(api.delete).toHaveBeenCalledWith("/api/projects/repos/onik97/nexul");
  });
});
