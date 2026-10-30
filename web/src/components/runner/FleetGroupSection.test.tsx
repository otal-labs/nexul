import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { MemoryRouter } from "react-router";
import { toast } from "sonner";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { FleetGroupSection } from "@/components/runner/FleetGroupSection";
import type { Machine, Runner } from "@/models/Runner";
import { resetViewer, viewer } from "@/test/viewerActions";

vi.mock("@/api/client", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/api/client")>()),
  api: { get: vi.fn(), delete: vi.fn() },
}));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));
vi.mock("@/hooks/useActionState", async () => (await import("@/test/viewerActions")).mockedActionState);
vi.mock("@/hooks/AccessHooks", () => ({ useAreaAccess: () => () => false, useIsInstanceOwner: () => true }));

const machine = (actions: string[]): Machine => ({
  id: "m-1", name: "edge-box", stack_root: "/data/nexul", first_seen: "2026-08-01T00:00:00Z", last_seen: "2026-08-12T12:00:00Z", actions,
});

const held = { allowed: true, source: { kind: "role" }, fallback: { allowed: true, source: { kind: "role" } } };

beforeEach(() => {
  resetViewer();
  viewer.workspace = ["stacks:write"];
  vi.mocked(api.get).mockReset().mockImplementation(((url: string) => {
    if (url === "/api/permissions/explain") return Promise.resolve({ data: { effective: [{ action: "machines:write", ...held }] } });
    if (url.startsWith("/api/permissions?")) return Promise.resolve({ data: { grants: [] } });
    if (url === "/api/team") return Promise.resolve({ data: { people: [], workspaces: [], can_manage_accounts: true } });
    if (url === "/api/version") return Promise.resolve({ data: { version: "v0.3.0" } });
    return Promise.reject(new Error(`unexpected GET ${url}`));
  }) as typeof api.get);
});

const runner = (name: string, connected: boolean): Runner => ({
  id: `r-${name}`, name, connected, last_seen: "2026-08-12T12:00:00Z", running_job: null, version: "v0.3.0", machine: "edge-box",
});

const renderGroup = (actions: string[], runners: Runner[] = []) =>
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <ContextAwareConfirmation.ConfirmationRoot />
      <MemoryRouter initialEntries={["/acme/runners"]}>
        <FleetGroupSection group={{ machine: machine(actions), runners, codes: [] }} />
      </MemoryRouter>
    </QueryClientProvider>,
  );

describe("FleetGroupSection people", () => {
  it("names who the machine's rules reach, and offers the ⋯ and a right-click Permissions… to a Manage holder", async () => {
    const user = userEvent.setup();
    renderGroup(["machines:see", "machines:write"]);
    expect(await screen.findByRole("button", { name: "People on this machine: Only you" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "More actions for edge-box" })).toBeInTheDocument();

    await user.pointer({ keys: "[MouseRight]", target: screen.getByRole("heading", { name: "edge-box" }) });
    expect(await screen.findByRole("menuitem", { name: "Permissions…" })).toBeInTheDocument();
  });

  it("shows nobody else's people to someone who may only see and deploy to the machine", async () => {
    const user = userEvent.setup();
    renderGroup(["machines:see", "machines:deploy"]);
    expect(await screen.findByRole("heading", { name: "edge-box" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /People on this machine/ })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "More actions for edge-box" })).not.toBeInTheDocument();

    await user.pointer({ keys: "[MouseRight]", target: screen.getByRole("heading", { name: "edge-box" }) });
    expect(screen.queryByRole("menu")).not.toBeInTheDocument();
    expect(api.get).not.toHaveBeenCalledWith("/api/permissions?resource_type=machine&resource_id=m-1");
  });
});

describe("FleetGroupSection on a machine the viewer only sees", () => {
  it("shows no write control and no ⋯ when no role shows locked actions", async () => {
    renderGroup(["machines:see"], [runner("builder", true)]);
    await screen.findByRole("heading", { name: "edge-box" });

    expect(screen.queryByRole("button", { name: "Add runner to this machine" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Edit stack root /data/nexul" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "More actions for edge-box" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "More actions for builder" })).not.toBeInTheDocument();
    expect(screen.getByText("/data/nexul")).toBeInTheDocument();
  });

  it("locks each write with its reason, in place and in the menus, when a role shows locked actions", async () => {
    viewer.showLocked = true;
    const user = userEvent.setup();
    renderGroup(["machines:see"], [runner("builder", true)]);

    for (const name of ["Add runner to this machine", "Edit stack root /data/nexul"]) {
      expect(await screen.findByRole("button", { name })).toHaveAttribute("aria-disabled", "true");
    }

    await user.click(screen.getByRole("button", { name: "More actions for builder" }));
    const remove = await screen.findByRole("menuitem", { name: /Remove runner/ });
    expect(remove).toHaveAttribute("aria-disabled", "true");
    expect(remove).toHaveTextContent("Your role can't remove runners");
    await user.keyboard("{Escape}");

    await user.click(screen.getByRole("button", { name: "More actions for edge-box" }));
    expect(await screen.findByRole("menuitem", { name: /Import from this machine/ })).toHaveAttribute("aria-disabled", "true");
    const del = screen.getByRole("menuitem", { name: /Delete/ });
    expect(del).toHaveAttribute("aria-disabled", "true");
    expect(del).toHaveTextContent("Your role can't delete machines");
    await user.click(del);
    expect(screen.getByRole("menu")).toBeInTheDocument();
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });
});

describe("FleetGroupSection import", () => {
  it("opens the import wizard on this machine from the ⋯", async () => {
    const user = userEvent.setup();
    renderGroup(["machines:see", "machines:write"]);

    await user.click(await screen.findByRole("button", { name: "More actions for edge-box" }));
    await user.click(await screen.findByRole("menuitem", { name: "Import from this machine" }));
    expect(screen.queryByRole("menu")).not.toBeInTheDocument();
  });

  it.each([
    ["can't write stacks", [], ["machines:see", "machines:write"]],
    ["doesn't manage the machine", ["stacks:write"], ["machines:see", "machines:delete"]],
  ])("leaves Import out for someone who %s, the two things the import writes", async (_, workspace, actions) => {
    viewer.workspace = workspace;
    const user = userEvent.setup();
    renderGroup(actions);

    await user.click(await screen.findByRole("button", { name: "More actions for edge-box" }));
    await screen.findByRole("menu");
    expect(screen.queryByRole("menuitem", { name: /Import from this machine/ })).not.toBeInTheDocument();
  });
});

describe("FleetGroupSection delete", () => {
  const openDelete = async (user: ReturnType<typeof userEvent.setup>) => {
    await user.click(await screen.findByRole("button", { name: "More actions for edge-box" }));
    await user.click(await screen.findByRole("menuitem", { name: "Delete" }));
    return within(await screen.findByRole("dialog"));
  };

  beforeEach(() => {
    vi.mocked(api.delete).mockReset().mockResolvedValue({ data: undefined });
    vi.mocked(toast.error).mockClear();
  });

  it("names what goes in the confirm, and deletes nothing when it is cancelled", async () => {
    const user = userEvent.setup();
    renderGroup(["machines:see", "machines:delete"], [runner("builder", false), runner("live", true)]);

    const dialog = await openDelete(user);
    expect(dialog.getByText(/The offline runner/)).toHaveTextContent("builder");
    expect(dialog.getByText(/The offline runner/)).not.toHaveTextContent("live");
    await user.click(dialog.getByRole("button", { name: "Cancel" }));

    expect(api.delete).not.toHaveBeenCalled();
  });

  it("deletes the machine once confirmed", async () => {
    const user = userEvent.setup();
    renderGroup(["machines:see", "machines:delete"], [runner("builder", false)]);

    await user.click((await openDelete(user)).getByRole("button", { name: "Delete machine" }));

    expect(api.delete).toHaveBeenCalledWith("/api/machines/m-1");
  });

  it("shows the refusal that names what still uses the machine", async () => {
    const refusal = 'machine "edge-box" is still used by stack api, connected runner live; move or remove them first';
    vi.mocked(api.delete).mockRejectedValue({ response: { status: 409, data: { message: refusal, code: "CONFLICT" } } });
    const user = userEvent.setup();
    renderGroup(["machines:see", "machines:delete"]);

    await user.click((await openDelete(user)).getByRole("button", { name: "Delete machine" }));

    expect(toast.error).toHaveBeenCalledWith(refusal);
  });

  it("offers no Delete to someone who manages the machine without holding Delete", async () => {
    const user = userEvent.setup();
    renderGroup(["machines:see", "machines:write"]);

    await user.click(await screen.findByRole("button", { name: "More actions for edge-box" }));
    expect(await screen.findByRole("menuitem", { name: "Permissions…" })).toBeInTheDocument();
    expect(screen.queryByRole("menuitem", { name: "Delete" })).not.toBeInTheDocument();
  });
});
