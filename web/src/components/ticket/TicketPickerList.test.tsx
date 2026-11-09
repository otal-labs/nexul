import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { TicketPickerList } from "@/components/ticket/TicketPickerList";
import { useWorkspaceStore } from "@/stores/workspaceStore";

vi.mock("@/api/client", () => ({ api: { get: vi.fn() }, errorMessage: vi.fn() }));

const hits = [
  { type: "ticket", id: "t-1", title: "Checkout login", hint: "ACME-1", in_body: false, updated_at: "" },
  { type: "ticket", id: "t-9", title: "Checkout flow", hint: "GLOBEX-4", project_id: "p-2", in_body: false, updated_at: "" },
];

const renderPicker = (onSelect = vi.fn()) =>
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <TicketPickerList projectId="p-1" excludeId="t-1" onSelect={onSelect} />
    </QueryClientProvider>,
  );

const requested = (path: string) => vi.mocked(api.get).mock.calls.filter(([url]) => url === path);

beforeEach(() => {
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  vi.mocked(api.get).mockReset();
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/projects") return { data: [{ id: "p-1", prefix: "ACME" }] };
    if (url === "/api/search") return { data: { hits, more: [] } };
    return {
      data: [
        { id: "t-1", project_id: "p-1", number: 1, title: "Login" },
        { id: "t-2", project_id: "p-1", number: 2, title: "Checkout" },
      ],
    };
  });
});

describe("TicketPickerList", () => {
  it("shows the open project's tickets before anything is typed, leaving out the excluded one, and searches nothing", async () => {
    const onSelect = vi.fn();
    renderPicker(onSelect);

    await userEvent.click(await screen.findByRole("button", { name: /Checkout/ }));
    expect(onSelect).toHaveBeenCalledWith("t-2");
    expect(screen.queryByRole("button", { name: /Login/ })).not.toBeInTheDocument();
    expect(requested("/api/tickets").map(([, config]) => config)).toEqual([{ params: { project_id: "p-1", without: "body" } }]);
    expect(requested("/api/search")).toEqual([]);
  });

  it("searches the whole workspace on the server as the user types, offering another project's ticket and never loading every ticket", async () => {
    const onSelect = vi.fn();
    renderPicker(onSelect);

    await userEvent.type(screen.getByRole("textbox", { name: "Search tickets" }), "chec");
    await userEvent.click(await screen.findByRole("button", { name: /GLOBEX-4/ }));

    expect(onSelect).toHaveBeenCalledWith("t-9");
    expect(screen.queryByRole("button", { name: /ACME-1/ })).not.toBeInTheDocument();
    expect(requested("/api/search").map(([, config]) => config?.params)).toEqual([{ q: "chec", workspace_id: "ws-1", limit: 50, kind: "ticket" }]);
    expect(requested("/api/tickets").map(([, config]) => config)).toEqual([{ params: { project_id: "p-1", without: "body" } }]);
  });
});
