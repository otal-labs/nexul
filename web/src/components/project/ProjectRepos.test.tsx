import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ContextAwareConfirmation } from "react-confirm";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ProjectRepos } from "@/components/project/ProjectRepos";
import { api } from "@/api/client";
import type { IssueSync } from "@/models/IssueSync";
import type { RepoRef } from "@/models/Project";
import type { Installation } from "@/models/Repository";
import { withOwner } from "@/test/viewer";
import { useWorkspaceStore } from "@/stores/workspaceStore";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn(),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const repos: RepoRef[] = [
  { owner: "onik97", name: "nexul", full_name: "onik97/nexul", connector_id: "github", role: "app" },
  { owner: "onik97", name: "nexul-e2e", full_name: "onik97/nexul-e2e", connector_id: "github", role: "tests" },
];

const renderRepos = (list: RepoRef[] = repos, syncs: IssueSync[] = [], installations: Partial<Installation>[] = []) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  vi.mocked(api.get).mockImplementation(
    withOwner(async (url: string) => {
      if (url.endsWith("/issue-sync")) return { data: syncs };
      if (url === "/api/repositories/installations") return { data: { installations } };
      return { data: list };
    }),
  );
  return render(
    <QueryClientProvider client={client}>
      <ContextAwareConfirmation.ConfirmationRoot />
      <ProjectRepos projectId="p-1" />
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  vi.mocked(api.get).mockReset();
  vi.mocked(api.post).mockReset();
  vi.mocked(api.delete).mockReset();
  vi.mocked(api.put).mockReset();
});

const appSync: IssueSync = { project_id: "p-1", owner: "onik97", repo: "nexul", label: "nexul", mode: "label", enabled: true };

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

  it("marks the tests repository and only that one", async () => {
    renderRepos();
    expect(await screen.findByText("onik97/nexul-e2e")).toBeInTheDocument();
    expect(screen.getAllByText("Tests, never deployed")).toHaveLength(1);
  });

  it("shows an empty state when there are no repos", async () => {
    renderRepos([]);
    expect(await screen.findByText("No repositories yet. Add one to link its pull requests to tickets.")).toBeInTheDocument();
  });

  it("does not show the empty state while repos are still loading", () => {
    vi.mocked(api.get).mockReturnValue(new Promise(() => {}));
    renderRepos([]);
    expect(screen.queryByText("No repositories yet. Add one to link its pull requests to tickets.")).not.toBeInTheDocument();
  });

  it("does not submit empty repo fields", async () => {
    const user = userEvent.setup();
    renderRepos();

    await user.click(await screen.findByRole("button", { name: "Add repo" }));
    await user.click(screen.getByRole("button", { name: "Add repository" }));
    expect(await screen.findByText("Owner is required")).toBeInTheDocument();
    expect(screen.getByText("Repo name is required")).toBeInTheDocument();
    expect(api.post).not.toHaveBeenCalled();
    await user.click(screen.getByRole("button", { name: "Cancel" }));
  });

  it("removes a repo once confirmed", async () => {
    const user = userEvent.setup();
    vi.mocked(api.delete).mockResolvedValue({ data: undefined });
    renderRepos();

    await user.click(await screen.findByRole("button", { name: "Actions for onik97/nexul" }));
    await user.click(await screen.findByRole("button", { name: "Remove" }));
    expect(api.delete).not.toHaveBeenCalled();
    await user.click(await screen.findByRole("button", { name: "Remove repo" }));
    expect(api.delete).toHaveBeenCalledWith("/api/projects/repos/onik97/nexul");
  });

  it("switches a repository's issues on", async () => {
    const user = userEvent.setup();
    vi.mocked(api.put).mockResolvedValue({ data: appSync });
    renderRepos();

    const toggle = await screen.findByRole("switch", { name: "Issues of onik97/nexul" });
    expect(toggle).not.toBeChecked();
    await user.click(toggle);
    expect(api.put).toHaveBeenCalledWith("/api/projects/p-1/issue-sync", { owner: "onik97", repo: "nexul", enabled: true });
  });

  it("says when an installation can't read the issues a repository syncs", async () => {
    renderRepos(repos, [appSync], [{ account_login: "Onik97", missing_permissions: ["issues"] }]);
    expect(await screen.findByRole("switch", { name: "Issues of onik97/nexul" })).toBeChecked();
    expect(await screen.findByText("Issues permission missing")).toBeInTheDocument();
  });

  it("says nothing about the permission while no repository syncs its issues", async () => {
    renderRepos(repos, [], [{ account_login: "onik97", missing_permissions: ["issues"] }]);
    await screen.findByRole("switch", { name: "Issues of onik97/nexul" });
    expect(screen.queryByText("Issues permission missing")).not.toBeInTheDocument();
    expect(api.get).not.toHaveBeenCalledWith("/api/repositories/installations");
  });
});
