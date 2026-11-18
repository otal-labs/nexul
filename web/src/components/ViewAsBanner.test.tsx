import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, useLocation } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ViewAsBanner } from "@/components/ViewAsBanner";
import { useViewAsStore } from "@/stores/viewAsStore";

vi.mock("@/api/client", () => ({ api: { get: vi.fn() }, errorMessage: vi.fn() }));

const Where = () => <p>at {useLocation().pathname}</p>;

const renderBanner = () =>
  render(
    <QueryClientProvider client={new QueryClient()}>
      <MemoryRouter initialEntries={["/acme"]}>
        <ViewAsBanner />
        <Where />
      </MemoryRouter>
    </QueryClientProvider>,
  );

describe("ViewAsBanner", () => {
  beforeEach(() => {
    useViewAsStore.getState().exit();
  });

  it("is absent outside a view", () => {
    renderBanner();
    expect(screen.queryByRole("status")).not.toBeInTheDocument();
  });

  it.each([
    { from: "a role on the Access page", returnTo: "/acme/access/roles/r-dev", lands: "/acme/access/roles/r-dev" },
    { from: "a view started before the page was remembered", returnTo: undefined, lands: "/settings/team" },
  ])("names who is being viewed and where, and Stop viewing ends the view and returns to where it started: $from", async ({ returnTo, lands }) => {
    useViewAsStore.getState().start({ workspaceId: "ws-1", workspaceName: "Acme", userId: "u-alice", name: "Alice", avatarUrl: "", ...(returnTo && { returnTo }) });
    renderBanner();

    expect(screen.getByRole("status")).toHaveTextContent("Viewing as Alice in Acme.");
    await userEvent.setup().click(screen.getByRole("button", { name: "Stop viewing" }));

    expect(useViewAsStore.getState().target).toBeNull();
    expect(screen.queryByRole("status")).not.toBeInTheDocument();
    expect(screen.getByText(`at ${lands}`)).toBeInTheDocument();
  });
});
