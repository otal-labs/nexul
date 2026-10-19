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

  it("names who is being viewed and where, and Stop viewing ends the view and returns to Team settings", async () => {
    useViewAsStore.getState().start({ workspaceId: "ws-1", workspaceName: "Acme", userId: "u-alice", name: "Alice", avatarUrl: "" });
    renderBanner();

    expect(screen.getByRole("status")).toHaveTextContent("Viewing as Alice in Acme.");
    await userEvent.setup().click(screen.getByRole("button", { name: "Stop viewing" }));

    expect(useViewAsStore.getState().target).toBeNull();
    expect(screen.queryByRole("status")).not.toBeInTheDocument();
    expect(screen.getByText("at /settings/team")).toBeInTheDocument();
  });
});
