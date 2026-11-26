import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ContextAwareConfirmation } from "react-confirm";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { SignInMovesPanel } from "@/ee/components/SignInMovesPanel";
import { getSignInMovesKey, signInMoveFollower } from "@/ee/hooks/SignInMoveHooks";
import type { MoveAccount, SignInMoves } from "@/ee/models/SignInMove";
import { followFrame, isStale, seeded } from "@/test/followFrame";

vi.mock("@/hooks/useActionState", async () => (await import("@/test/liveActions")).liveActions);

const mocks = vi.hoisted(() => ({
  get: vi.fn(),
  put: vi.fn(),
  post: vi.fn(),
  delete: vi.fn(),
  toast: { success: vi.fn(), error: vi.fn() },
}));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, put: mocks.put, post: mocks.post, delete: mocks.delete },
  errorMessage: (err: unknown) => (err as Error).message,
}));

vi.mock("sonner", () => ({ toast: mocks.toast }));

const account = (id: string, email: string): MoveAccount => ({ id, login: id, name: id, email, created_at: "2026-11-20T10:00:00Z" });

const moves = (over: Partial<SignInMoves> = {}): SignInMoves => ({
  protocol: "saml",
  provider_name: "Acme SAML",
  people: [
    { ...account("alice", "alice@example.com"), match: account("dup-alice", "alice@example.com"), expected_email: "" },
    { ...account("bob", "bob@example.com"), match: account("dup-bob", "bob@example.com"), expected_email: "" },
    { ...account("carol", "carol@old.example.com"), match: null, expected_email: "" },
    { ...account("dave", "dave@example.com"), match: null, expected_email: "dave@example.com" },
  ],
  unmatched: [account("dup-carol", "carol@example.com")],
  expected: [{ account: account("dave", "dave@example.com"), created_by: "owner", created_at: "2026-11-21T10:00:00Z" }],
  ...over,
});

const renderPanel = (data: SignInMoves) => {
  mocks.get.mockResolvedValue({ data });
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <ContextAwareConfirmation.ConfirmationRoot />
        <SignInMovesPanel />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  vi.clearAllMocks();
  mocks.post.mockResolvedValue({ data: { results: [{ from_account_id: "dup-alice", to_account_id: "alice", moved: true, message: "" }] } });
  mocks.put.mockResolvedValue({ data: {} });
  mocks.delete.mockResolvedValue({ data: undefined });
});

describe("SignInMovesPanel", () => {
  it("links every ticked pair in one request once confirmed", async () => {
    const user = userEvent.setup();
    renderPanel(moves());

    await user.click(await screen.findByRole("checkbox", { name: "Link bob@example.com to bob" }));
    await user.click(screen.getByRole("button", { name: "Link 1 sign-in" }));
    await user.click(await screen.findByRole("button", { name: "Link" }));

    await waitFor(() => expect(mocks.post).toHaveBeenCalledWith("/api/instance/sso/moves", { pairs: [{ from_account_id: "dup-alice", to_account_id: "alice" }] }));
    expect(mocks.toast.success).toHaveBeenCalledWith("Sign-in linked");
  });

  it("says which pairs did not move", async () => {
    const user = userEvent.setup();
    mocks.post.mockResolvedValue({
      data: {
        results: [
          { from_account_id: "dup-alice", to_account_id: "alice", moved: true, message: "" },
          { from_account_id: "dup-bob", to_account_id: "bob", moved: false, message: "that account already signs in through single sign-on" },
        ],
      },
    });
    renderPanel(moves());

    await user.click(await screen.findByRole("button", { name: "Link 2 sign-ins" }));
    await user.click(await screen.findByRole("button", { name: "Link" }));

    await waitFor(() => expect(mocks.toast.error).toHaveBeenCalledWith("1 linked, 1 not: that account already signs in through single sign-on"));
  });

  it("expects an unmatched person's sign-in from the email Nexul knows, or one typed instead", async () => {
    const user = userEvent.setup();
    renderPanel(moves());

    const carol = (await screen.findByText("carol@old.example.com")).closest("li") as HTMLElement;
    expect(screen.queryByText("Expect dave's sign-in")).not.toBeInTheDocument();
    await user.click(within(carol).getByRole("button", { name: "Expect sign-in" }));
    const field = await screen.findByLabelText("Email");
    expect(field).toHaveValue("carol@old.example.com");
    await user.clear(field);
    await user.type(field, "carol@example.com");
    await user.click(screen.getByRole("button", { name: "Expect sign-in" }));

    await waitFor(() => expect(mocks.put).toHaveBeenCalledWith("/api/instance/sso/expected/carol", { email: "carol@example.com" }));
  });

  it("links a person to an account picked by hand", async () => {
    const user = userEvent.setup();
    renderPanel(moves());

    const carol = (await screen.findByText("carol@old.example.com")).closest("li") as HTMLElement;
    await user.click(within(carol).getByRole("button", { name: "Pick account" }));
    await user.click(await screen.findByRole("menuitem", { name: /carol@example\.com/ }));
    await user.click(await screen.findByRole("button", { name: "Link" }));

    await waitFor(() => expect(mocks.post).toHaveBeenCalledWith("/api/instance/sso/moves", { pairs: [{ from_account_id: "dup-carol", to_account_id: "carol" }] }));
  });

  it("removes an expected sign-in", async () => {
    const user = userEvent.setup();
    renderPanel(moves());

    await user.click(await screen.findByRole("button", { name: "Remove expected sign-in" }));
    await user.click(screen.getByRole("button", { name: "Remove" }));

    await waitFor(() => expect(mocks.delete).toHaveBeenCalledWith("/api/instance/sso/expected/dave"));
  });

  it("says so when no provider is saved, and when nothing is left to move", async () => {
    renderPanel(moves({ protocol: "", provider_name: "", people: [], unmatched: [], expected: [] }));
    expect(await screen.findByText(/Set up the single sign-on provider people move to first/)).toBeInTheDocument();
    expect(screen.getByText("No sign-in is expected.")).toBeInTheDocument();
    expect(screen.queryByText("Likely duplicates")).not.toBeInTheDocument();
  });

  it("follows every frame that changes who is still on the other protocol", async () => {
    for (const topic of ["instance.sso_expected_changed", "identity.linked", "account.admitted", "account.disabled", "instance.sso_changed", "instance.saml_changed"]) {
      const client = seeded([[[getSignInMovesKey], moves()]]);
      await followFrame(signInMoveFollower, topic, {}, client);
      expect(isStale(client, [getSignInMovesKey]), topic).toBe(true);
    }
  });
});
