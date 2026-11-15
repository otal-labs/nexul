import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { TeamSsoRow } from "@/ee/components/TeamSsoRow";
import type { TeamPerson } from "@/models/Team";

const mocks = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  permissions: [] as string[],
  confirm: vi.fn(),
  toast: { success: vi.fn(), error: vi.fn() },
}));

vi.mock("@/api/client", () => ({ api: { get: mocks.get, post: mocks.post }, errorMessage: (err: unknown) => (err as Error).message }));
vi.mock("@/hooks/AccessHooks", () => ({ useHasInstancePermission: (value: string) => mocks.permissions.includes(value) }));
vi.mock("@/hooks/useConfirmationDialog", () => ({ useConfirmationDialog: () => ({ open: mocks.confirm }) }));
vi.mock("sonner", () => ({ toast: mocks.toast }));

const person = (id: string, providers: string[], over: Partial<TeamPerson> = {}): TeamPerson => ({
  id,
  login: id,
  name: id,
  avatar_url: "",
  status: "active",
  created_at: "",
  providers: providers as TeamPerson["providers"],
  online: false,
  last_seen_at: null,
  workspaces: [],
  ...over,
});

const alice = person("alice", ["github"]);
const duplicate = person("dup", ["oidc"], { login: "alice@example.com", name: "Alice" });
const removed = person("gone", ["oidc"], { login: "gone@example.com", status: "removed" });

const serve = (active: boolean) =>
  mocks.get.mockImplementation((url: string) =>
    Promise.resolve({
      data: url === "/api/auth/sso" ? { active, name: "Acme SSO", only: false } : { people: [alice, duplicate, removed], workspaces: [] },
    }),
  );

const renderRow = (who: TeamPerson) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <TeamSsoRow person={who} />
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  vi.clearAllMocks();
  mocks.permissions = ["instance:write"];
  mocks.post.mockResolvedValue({ data: undefined });
});

describe("TeamSsoRow", () => {
  it("moves the duplicate a first sign-in made onto the person once the Owner confirms, offering no removed account", async () => {
    serve(true);
    mocks.confirm.mockResolvedValue(true);
    renderRow(alice);

    await userEvent.click(await screen.findByRole("button", { name: "Move from another account" }));
    expect(screen.queryByText("gone@example.com")).not.toBeInTheDocument();
    await userEvent.click(screen.getByRole("menuitem", { name: /alice@example\.com/ }));

    await waitFor(() => expect(mocks.post).toHaveBeenCalledWith("/api/instance/sso/move", { from_account_id: "dup", to_account_id: "alice" }));
    expect(mocks.confirm).toHaveBeenCalledWith(expect.objectContaining({ confirmLabel: "Move sign-in" }));
  });

  it("says who signs in through the provider, and offers nothing to someone who can't move it", async () => {
    serve(true);
    mocks.permissions = [];
    renderRow(duplicate);
    expect(await screen.findByText("Signs in through Acme SSO")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Move from another account" })).not.toBeInTheDocument();
  });

  it("shows nothing while single sign-on is off", async () => {
    serve(false);
    renderRow(alice);
    await waitFor(() => expect(mocks.get).toHaveBeenCalledWith("/api/auth/sso"));
    expect(screen.queryByText("Single sign-on")).not.toBeInTheDocument();
  });
});
