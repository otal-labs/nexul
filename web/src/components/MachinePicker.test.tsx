import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useForm } from "react-hook-form";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { MachinePicker } from "@/components/MachinePicker";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { asViewer, memberWho } from "@/test/viewer";

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

  it("offers the machines the viewer may deploy to by name, never runners", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: [{ id: "m1", name: "prod-1" }] });
    const user = userEvent.setup();
    renderPicker();

    await user.click(await screen.findByRole("combobox", { name: "Machine" }));
    expect(await screen.findByRole("option", { name: "prod-1" })).toBeInTheDocument();
    expect(api.get).toHaveBeenCalledWith("/api/machines/deployable");
  });

  // Adding a machine is an instance permission: the workspace's own permissions never offer it.
  it.each([
    { who: "the instance Owner", workspace: [], instance: { instance_permissions: [], instance_owner: true }, link: true },
    { who: "someone holding Add machines on the instance", workspace: [], instance: { instance_permissions: ["machines:create"], instance_owner: false }, link: true },
    { who: "a workspace role that lists Add machines but holds nothing on the instance", workspace: ["machines:create"], instance: { instance_permissions: [], instance_owner: false }, link: false },
  ])("with no machine to deploy to, links Runners for $who: $link", async ({ workspace, instance, link }) => {
    vi.mocked(api.get).mockImplementation(asViewer(memberWho(workspace, false), () => [], instance));
    renderPicker();

    expect(await screen.findByText(/No machine you can deploy to yet/)).toBeInTheDocument();
    await vi.waitFor(() => expect(screen.queryByRole("link", { name: "Add a runner" }) !== null).toBe(link));
    if (link) expect(screen.getByRole("link", { name: "Add a runner" })).toHaveAttribute("href", "/acme/runners");
    if (!link) expect(await screen.findByText(/Ask an admin for access to a machine/)).toBeInTheDocument();
  });
});
