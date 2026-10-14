import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useForm } from "react-hook-form";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { MachinePicker } from "@/components/MachinePicker";
import { useWorkspaceStore } from "@/stores/workspaceStore";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn() },
  errorMessage: vi.fn(),
}));

const Harness = () => {
  const form = useForm<{ target: string }>({ defaultValues: { target: "" } });
  return <MachinePicker control={form.control} name="target" />;
};

const renderPicker = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <Harness />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

describe("MachinePicker", () => {
  beforeEach(() => {
    vi.mocked(api.get).mockReset();
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1", selectedWorkspaceSlug: "acme" });
  });

  it("offers machines by name, never runners", async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: [{ id: "m1", name: "prod-1", stack_root: "/data/nexul", first_seen: "", last_seen: "" }],
    });
    const user = userEvent.setup();
    renderPicker();

    await user.click(await screen.findByRole("combobox", { name: "Machine" }));
    expect(await screen.findByRole("option", { name: "prod-1" })).toBeInTheDocument();
    expect(api.get).toHaveBeenCalledWith("/api/machines");
  });

  const mockEmptyMachines = (permissions: string[]) =>
    vi.mocked(api.get).mockImplementation(async (url: string) => {
      if (url === "/api/workspaces/ws-1/me") return { data: { role_name: "Member", permissions } };
      return { data: [] };
    });

  it("links an empty machine list to the Runners page for someone who can open it", async () => {
    mockEmptyMachines(["machines:create"]);
    renderPicker();

    expect(await screen.findByRole("link", { name: "Add a runner" })).toHaveAttribute("href", "/acme/runners");
  });

  it("says who to ask, with no link, when the viewer can't open Runners", async () => {
    mockEmptyMachines([]);
    renderPicker();

    expect(await screen.findByText(/Ask an admin to add a runner/)).toBeInTheDocument();
    expect(screen.queryByRole("link")).not.toBeInTheDocument();
  });
});
