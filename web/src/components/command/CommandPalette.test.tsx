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

const permissions = ["tickets:read", "docs:read", "memories:read", "projects:read", "chat:write"];

const LocationSpy = () => <div data-testid="location">{useLocation().pathname}</div>;

const hit = (type: string, id: string, title: string, extra: Record<string, unknown> = {}) => ({ type, id, title, in_body: false, updated_at: "", ...extra });

let found: { hits: ReturnType<typeof hit>[]; more: string[] } = { hits: [hit("ticket", "t-1", "Renew the staging certificate")], more: [] };

const renderPalette = (extra: string[] = []) => {
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/projects") return { data: [{ id: "p-1", name: "Backend", prefix: "BE", position: 0, created_at: "", updated_at: "" }] };
    if (url === "/api/workspaces/ws-1/me") return { data: { role_name: "Member", permissions: [...permissions, ...extra] } };
    if (url === "/api/machines") return { data: [{ id: "m-1", name: "box" }] };
    if (url === "/api/search") return { data: found };
    return { data: [] };
  });
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={["/acme/inbox"]}>
        <button type="button">Elsewhere</button>
        <CommandPalette />
        <LocationSpy />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  found = { hits: [hit("ticket", "t-1", "Renew the staging certificate")], more: [] };
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1", selectedProjectId: "" });
  useCommandPaletteStore.setState({ open: false });
});

describe("CommandPalette", () => {
  it("opens on Ctrl+K, ranks an action named by the query above search hits, and opens a page with Enter", async () => {
    const user = userEvent.setup();
    renderPalette();

    await user.keyboard("{Control>}k{/Control}");
    const input = await screen.findByRole("combobox", { name: /search pages/i });
    await user.type(input, "new");
    await screen.findByRole("option", { name: /Renew the staging certificate/ });
    expect(screen.getAllByRole("option")[0]).toHaveAccessibleName(/New direct message/);

    await user.clear(input);
    await user.type(input, "run");
    const runners = await screen.findByRole("option", { name: /Runners/ });
    expect(runners).toHaveAttribute("aria-selected", "true");
    expect(input).toHaveAttribute("aria-activedescendant", runners.id);

    await user.keyboard("{Enter}");
    expect(screen.getByTestId("location")).toHaveTextContent("/acme/runners");
    expect(screen.queryByRole("dialog", { name: "Command palette" })).not.toBeInTheDocument();
  });

  it("walks every group with the arrows, wraps at the ends, and hands focus back on Escape", async () => {
    const user = userEvent.setup();
    renderPalette();
    screen.getByRole("button", { name: "Elsewhere" }).focus();

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
    expect(screen.getByRole("button", { name: "Elsewhere" })).toHaveFocus();
  });

  it("lists no settings until something is typed, then finds a section by its keywords and opens its path", async () => {
    const user = userEvent.setup();
    renderPalette();

    await user.keyboard("{Control>}k{/Control}");
    const input = await screen.findByRole("combobox");
    expect(screen.queryByRole("option", { name: /Security/ })).not.toBeInTheDocument();

    await user.type(input, "tokens");
    const security = await screen.findByRole("option", { name: /Security/ });
    expect(security).toHaveTextContent("Settings");

    await user.keyboard("{Enter}");
    expect(screen.getByTestId("location")).toHaveTextContent("/settings/security");
  });

  it("offers the Roles tab only to someone who can write roles", async () => {
    const user = userEvent.setup();
    const { unmount } = renderPalette();
    await user.keyboard("{Control>}k{/Control}");
    await user.type(await screen.findByRole("combobox"), "roles");
    await screen.findByRole("option", { name: /Renew the staging certificate/ });
    expect(screen.queryByRole("option", { name: /Roles/ })).not.toBeInTheDocument();
    unmount();
    useCommandPaletteStore.setState({ open: false });

    renderPalette(["roles:write"]);
    await user.keyboard("{Control>}k{/Control}");
    await user.type(await screen.findByRole("combobox"), "roles");
    await user.click(await screen.findByRole("option", { name: /Roles/ }));
    expect(screen.getByTestId("location")).toHaveTextContent("/acme/access/roles");
  });

  it("groups each kind's title hits, puts text-only hits under In text, and opens a pull request's ticket", async () => {
    found = {
      hits: [
        hit("ticket", "t-1", "Checkout flow", { hint: "BE-4" }),
        hit("ticket", "t-2", "Payments", { hint: "BE-5", in_body: true }),
        hit("pull_request", "acme/app#9", "Speed up checkout", { hint: "app#9", ticket_id: "t-1" }),
        hit("channel", "c-1", "checkout"),
      ],
      more: ["ticket"],
    };
    const user = userEvent.setup();
    renderPalette();

    await user.keyboard("{Control>}k{/Control}");
    await user.type(await screen.findByRole("combobox"), "checkout");
    const inText = await screen.findByRole("group", { name: "In text" });
    expect(inText).toHaveTextContent("Payments");
    expect(screen.getByRole("group", { name: "Tickets" })).not.toHaveTextContent("Payments");
    expect(screen.getByText("More matches than shown. Add a word to narrow.")).toBeInTheDocument();

    await user.click(screen.getByRole("option", { name: /Speed up checkout/ }));
    expect(screen.getByTestId("location")).toHaveTextContent("/acme/tickets/t-1");
  });
});
