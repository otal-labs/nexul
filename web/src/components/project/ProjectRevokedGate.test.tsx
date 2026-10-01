import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ProjectRevokedGate } from "@/components/project/ProjectRevokedGate";
import { useWorkspaceStore } from "@/stores/workspaceStore";

const mocks = vi.hoisted(() => ({ get: vi.fn() }));
vi.mock("@/api/client", () => ({ api: mocks, errorMessage: () => "" }));

const web = { id: "p-web", name: "Web", prefix: "WEB" };

const renderGate = (path: string) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[path]}>
        <ProjectRevokedGate>
          <p>Page body</p>
        </ProjectRevokedGate>
      </MemoryRouter>
    </QueryClientProvider>,
  );
  return client;
};

describe("ProjectRevokedGate", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1", selectedWorkspaceSlug: "otal" });
  });

  it.each(["/otal/board/WEB", "/otal/tickets/WEB-12", "/otal/projects/WEB/settings"])(
    "turns %s into the revoked state when its project drops out of the list while open",
    async (path) => {
      mocks.get.mockResolvedValue({ data: [web] });
      const client = renderGate(path);
      await vi.waitFor(() => expect(client.getQueryData(["getProjects", "ws-1"])).toEqual([web]));
      expect(screen.getByText("Page body")).toBeInTheDocument();

      mocks.get.mockResolvedValue({ data: [] });
      await act(() => client.invalidateQueries());

      expect(await screen.findByText("You no longer have access to this project")).toBeInTheDocument();
      expect(screen.getByText("Someone changed your access. Projects you can still open are in the sidebar.")).toBeInTheDocument();
      expect(screen.getByRole("link", { name: "Go to Home" })).toHaveAttribute("href", "/otal");
      expect(screen.queryByText("Page body")).not.toBeInTheDocument();
    },
  );

  it("leaves a link to a project the viewer never saw to the page's own not found", async () => {
    mocks.get.mockResolvedValue({ data: [] });
    renderGate("/otal/board/WEB");

    expect(await screen.findByText("Page body")).toBeInTheDocument();
    await vi.waitFor(() => expect(mocks.get).toHaveBeenCalled());
    expect(screen.queryByText("You no longer have access to this project")).not.toBeInTheDocument();
  });
});
