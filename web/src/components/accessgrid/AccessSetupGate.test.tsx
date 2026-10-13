import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { Outlet, RouterProvider, createMemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AccessSetupGate } from "@/components/accessgrid/AccessSetupGate";
import { AccessSetupNav } from "@/components/sidebar/AccessSetupNav";
import { useAccessSetupRedirect } from "@/hooks/useAccessSetupRedirect";
import { useAccessSetupStore } from "@/stores/accessSetupStore";
import { useWorkspaceStore } from "@/stores/workspaceStore";

const setup = vi.hoisted(() => ({ state: undefined as "owner" | "member" | undefined }));
vi.mock("@/hooks/AccessGridHooks", () => ({ useAccessSetupState: () => setup.state }));

// The workspace home's redirect, without the rest of the home.
const Home = () => {
  useAccessSetupRedirect();
  return <p>the home</p>;
};

const renderAt = (path: string) =>
  render(
    <QueryClientProvider client={new QueryClient()}>
      <RouterProvider
        router={createMemoryRouter(
          [
            {
              path: "/acme",
              element: (
                <AccessSetupGate>
                  <Outlet />
                </AccessSetupGate>
              ),
              children: [
                { index: true, handle: { needsProject: true }, element: <Home /> },
                { path: "board", handle: { area: "tickets", needsProject: true }, element: <p>the board</p> },
                { path: "chat", element: <p>the chat</p> },
                { path: "access", element: <p>the access page</p> },
              ],
            },
          ],
          { initialEntries: [path] },
        )}
      />
    </QueryClientProvider>,
  );

beforeEach(() => {
  setup.state = undefined;
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1", selectedWorkspaceSlug: "acme" });
  useAccessSetupStore.setState({ redirected: [] });
});

describe("while the access setup is open", () => {
  it("a member no project lists waits on a page that shows a project, and reaches chat", async () => {
    setup.state = "member";
    renderAt("/acme/board");
    expect(await screen.findByText("Your projects are on their way")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Open chat" })).toHaveAttribute("href", "/acme/chat");
  });

  it("leaves a page with no project alone, and every page once done", async () => {
    setup.state = "member";
    const { unmount } = renderAt("/acme/chat");
    expect(await screen.findByText("the chat")).toBeInTheDocument();
    unmount();
    setup.state = undefined;
    renderAt("/acme/board");
    expect(await screen.findByText("the board")).toBeInTheDocument();
  });

  it("sends an Owner from the workspace home to the setup once", async () => {
    setup.state = "owner";
    const { unmount } = renderAt("/acme");
    expect(await screen.findByText("the access page")).toBeInTheDocument();
    expect(useAccessSetupStore.getState().redirected).toEqual(["ws-1"]);
    unmount();
    renderAt("/acme");
    expect(await screen.findByText("the home")).toBeInTheDocument();
  });
});

describe("the sidebar while the access setup is open", () => {
  const renderNav = (state: "owner" | "member", collapsed = false) =>
    render(
      <RouterProvider router={createMemoryRouter([{ path: "*", element: <AccessSetupNav setup={state} collapsed={collapsed} /> }])} />,
    );

  it("gets the Owner back to the setup", () => {
    renderNav("owner");
    expect(screen.getByRole("link", { name: "Set up access" })).toHaveAttribute("href", "/acme/access");
  });

  it("tells a member their projects are being set up, by name on the rail", () => {
    renderNav("member", true);
    expect(screen.getByLabelText("Projects being set up")).toBeInTheDocument();
  });
});
