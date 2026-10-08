import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, useLocation } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { CommandPalette } from "@/components/command/CommandPalette";
import { useCommandPaletteStore } from "@/stores/commandPaletteStore";
import { useWorkspaceStore } from "@/stores/workspaceStore";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn(),
}));

const permissions = ["tickets:read", "docs:read", "memories:read", "projects:read", "runners:read"];

const LocationSpy = () => <div data-testid="location">{useLocation().pathname}</div>;

const renderPalette = () => {
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/projects") return { data: [{ id: "p-1", name: "Backend", prefix: "BE", position: 0, created_at: "", updated_at: "" }] };
    if (url === "/api/workspaces/ws-1/me") return { data: { role_name: "Member", permissions } };
    return { data: [] };
  });
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={["/acme/inbox"]}>
        <CommandPalette />
        <LocationSpy />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1", selectedProjectId: "" });
  useCommandPaletteStore.setState({ open: false });
});

describe("CommandPalette", () => {
  it("opens on Ctrl+K, filters as you type, and opens the active result with Enter", async () => {
    const user = userEvent.setup();
    renderPalette();

    await user.keyboard("{Control>}k{/Control}");
    const input = await screen.findByRole("combobox", { name: /search pages/i });
    await user.type(input, "run");

    const runners = await screen.findByRole("option", { name: /Runners/ });
    expect(runners).toHaveAttribute("aria-selected", "true");
    expect(input).toHaveAttribute("aria-activedescendant", runners.id);

    await user.keyboard("{Enter}");
    expect(screen.getByTestId("location")).toHaveTextContent("/acme/runners");
    expect(screen.queryByRole("dialog", { name: "Command palette" })).not.toBeInTheDocument();
  });

  it("walks every group with the arrows and wraps at the ends", async () => {
    const user = userEvent.setup();
    renderPalette();

    await user.keyboard("{Control>}k{/Control}");
    const input = await screen.findByRole("combobox");
    await user.type(input, "b");
    const options = await screen.findAllByRole("option");
    expect(options[0]).toHaveAttribute("aria-selected", "true");

    await user.keyboard("{ArrowUp}");
    expect(options.at(-1)).toHaveAttribute("aria-selected", "true");
    await user.keyboard("{ArrowDown}{ArrowDown}");
    expect(options[1]).toHaveAttribute("aria-selected", "true");

    await user.keyboard("{Escape}");
    expect(screen.queryByRole("combobox")).not.toBeInTheDocument();
  });
});
