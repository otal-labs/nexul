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

const permissions = ["tickets:read", "docs:read", "memories:read", "projects:read", "chat:write", "stacks:read", "automations:read"];

const LocationSpy = () => {
  const { pathname, search, hash } = useLocation();
  return <div data-testid="location">{`${pathname}${search}${hash}`}</div>;
};

const hits = [
  { type: "ticket", id: "t-1", title: "Renew the staging certificate", can_open: true, key: "BE-4", match: "title" },
  { type: "ticket", id: "t-2", title: "Slow board", can_open: true, key: "BE-9", match: "text", snippet: "the certificate expired", note_id: "m-1" },
  { type: "doc", id: "d-1", title: "Runbook", can_open: true, match: "text", snippet: "rotate the certificate", heading: "TLS certificates" },
];

const renderPalette = (extra: string[] = []) => {
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/projects") return { data: [{ id: "p-1", name: "Backend", prefix: "BE", position: 0, created_at: "", updated_at: "" }] };
    if (url === "/api/workspaces/ws-1/me") return { data: { role_name: "Member", permissions: [...permissions, ...extra] } };
    if (url === "/api/machines") return { data: [{ id: "m-1", name: "box" }] };
    if (url === "/api/mentions/search") return { data: { results: hits } };
    if (url === "/api/stacks") return { data: [{ id: "s-1", project_id: "p-1", name: "Certificate renewer" }, { id: "s-2", project_id: "p-other", name: "Certificate elsewhere" }] };
    if (url === "/api/automations") return { data: [{ id: "au-1", name: "Certificate alerts" }] };
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
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1", selectedProjectId: "" });
  useCommandPaletteStore.setState({ open: false });
});

describe("CommandPalette", () => {
  it("shows title matches apart from text matches, and opens a text match where its words are", async () => {
    const user = userEvent.setup();
    renderPalette();

    await user.keyboard("{Control>}k{/Control}");
    await user.type(await screen.findByRole("combobox"), "certificate");
    const found = await screen.findByRole("group", { name: "Found in tickets" });
    expect(screen.getByRole("group", { name: "Tickets" })).toHaveTextContent("Renew the staging certificateBE-4");
    expect(found).toHaveTextContent("the certificate expired");
    expect(screen.getByRole("group", { name: "Found in docs" })).toHaveTextContent("TLS certificates · rotate the certificate");
    expect(await screen.findByRole("option", { name: /Certificate renewer/ })).toBeInTheDocument();
    expect(screen.queryByRole("option", { name: /Certificate elsewhere/ })).not.toBeInTheDocument();
    expect(await screen.findByRole("option", { name: /Certificate alerts/ })).toBeInTheDocument();

    await user.click(screen.getByRole("option", { name: /Slow board/ }));
    expect(screen.getByTestId("location")).toHaveTextContent("/acme/tickets/BE-9/notes?note=m-1");

    await user.keyboard("{Control>}k{/Control}");
    await user.type(await screen.findByRole("combobox"), "certificate");
    await user.click(await screen.findByRole("option", { name: /Runbook/ }));
    expect(screen.getByTestId("location")).toHaveTextContent("/acme/docs/d-1#tls-certificates");
  });

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

  it("lists no settings until something is typed, then finds a tab by name and opens its path", async () => {
    const user = userEvent.setup();
    renderPalette();

    await user.keyboard("{Control>}k{/Control}");
    const input = await screen.findByRole("combobox");
    expect(screen.queryByRole("option", { name: /Tokens/ })).not.toBeInTheDocument();

    await user.type(input, "tokens");
    const tokens = await screen.findByRole("option", { name: /Tokens/ });
    expect(tokens).toHaveTextContent("Settings › Security");

    await user.keyboard("{Enter}");
    expect(screen.getByTestId("location")).toHaveTextContent("/settings/security/tokens");
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
});
