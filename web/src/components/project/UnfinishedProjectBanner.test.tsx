import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { UnfinishedProjectBanner } from "@/components/project/UnfinishedProjectBanner";
import { useUnfinishedProjectStore } from "@/stores/unfinishedProjectStore";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import type { Project, RepoRef } from "@/models/Project";

const mocks = vi.hoisted(() => ({ get: vi.fn() }));
vi.mock("@/api/client", () => ({ api: mocks, errorMessage: vi.fn() }));

const project: Project = { id: "p-1", name: "Backend", prefix: "BE", position: 0, icon: "", tests_location: "", created_at: "", updated_at: "" };

const repo = (role: RepoRef["role"]): RepoRef => ({ owner: "acme", name: role, full_name: `acme/${role}`, connector_id: "github", role });

const serve = (repos: RepoRef[]) =>
  mocks.get.mockImplementation(async (url: string) => ({ data: url === "/api/projects" ? [project] : repos }));

const renderBanner = () =>
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <UnfinishedProjectBanner projectId="p-1" />
      </MemoryRouter>
    </QueryClientProvider>,
  );

beforeEach(() => {
  mocks.get.mockReset();
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  useUnfinishedProjectStore.getState().start("p-1");
});

describe("UnfinishedProjectBanner", () => {
  it("leads back to the repository step until dismissed, while only a tests repository is attached", async () => {
    serve([repo("tests")]);
    const user = userEvent.setup();
    renderBanner();

    expect(await screen.findByRole("link", { name: "Continue setup" })).toHaveAttribute(
      "href",
      "/acme/wizard/project/repository?project=p-1",
    );
    await user.click(screen.getByRole("button", { name: "Dismiss" }));

    expect(screen.queryByRole("link", { name: "Continue setup" })).not.toBeInTheDocument();
  });

  it("stays away once the project has a repository to deploy", async () => {
    serve([repo("app")]);
    renderBanner();

    await vi.waitFor(() => expect(mocks.get).toHaveBeenCalledWith("/api/projects/p-1/repos"));
    expect(screen.queryByRole("link", { name: "Continue setup" })).not.toBeInTheDocument();
  });
});
