import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { PersonalAccessTokensSection } from "@/components/settings/PersonalAccessTokensSection";

const mocks = vi.hoisted(() => ({
  get: vi.fn(),
  put: vi.fn(),
  post: vi.fn(),
  patch: vi.fn(),
  del: vi.fn(),
  errorMessage: vi.fn(),
}));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, put: mocks.put, post: mocks.post, patch: mocks.patch, delete: mocks.del },
  errorMessage: mocks.errorMessage,
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const patList = (tokens: unknown[]) => ({ tokens });

const token = (overrides: Record<string, unknown> = {}) => ({
  id: "pat-1",
  user_id: "u1",
  name: "ci agent",
  prefix: "abc123",
  created_at: "2026-08-01T00:00:00Z",
  ...overrides,
});

const renderSection = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <PersonalAccessTokensSection />
    </QueryClientProvider>,
  );
};

describe("PersonalAccessTokensSection", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.put.mockReset();
    mocks.post.mockReset();
    mocks.patch.mockReset();
    mocks.del.mockReset();
    mocks.errorMessage.mockClear();
    mocks.get.mockResolvedValue({ data: patList([]) });
  });

  it("shows an error when the token list fails to load", async () => {
    mocks.get.mockRejectedValue(new Error("boom"));
    mocks.errorMessage.mockReturnValue("Tokens failed");
    renderSection();
    expect(await screen.findByText("Tokens failed")).toBeInTheDocument();
  });

  it("shows an empty state when there are no tokens", async () => {
    renderSection();
    expect(
      await screen.findByText(/no personal access tokens yet/i),
    ).toBeInTheDocument();
  });

  it("marks a paired computer's own token as belonging to it", async () => {
    mocks.get.mockResolvedValue({
      data: patList([token({ name: "Nexul MCP on Home", computer_id: "c1" }), token({ id: "pat-2" })]),
    });
    renderSection();
    expect(await screen.findByText("Nexul MCP on Home")).toBeInTheDocument();
    expect(screen.getAllByText("paired computer")).toHaveLength(1);
  });

  it("does not show the empty state while tokens are still loading", () => {
    mocks.get.mockReturnValue(new Promise(() => {}));
    renderSection();
    expect(screen.queryByText(/no personal access tokens yet/i)).not.toBeInTheDocument();
  });

  it("mints a personal access token and shows it once", async () => {
    mocks.post.mockResolvedValue({
      data: {
        token: "dep_ABC123rawvalue",
        id: "pat-1",
        name: "ci agent",
        prefix: "rawvalue",
        created_at: "2026-08-12T00:00:00Z",
      },
    });
    const user = userEvent.setup();
    renderSection();

    await user.type(await screen.findByLabelText(/token name/i), "ci agent");
    await user.click(screen.getByRole("button", { name: /^create token$/i }));

    expect(await screen.findByText(/copy this token now/i)).toBeInTheDocument();
    expect(screen.getByText("dep_ABC123rawvalue")).toBeInTheDocument();
    expect(mocks.post).toHaveBeenCalledWith("/api/auth/tokens", { name: "ci agent" });
  });

  it("does not call the api for an empty token name", async () => {
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: /^create token$/i }));

    expect(mocks.post).not.toHaveBeenCalled();
  });

  it("lists tokens with revoke state", async () => {
    mocks.get.mockResolvedValue({
      data: patList([token(), token({ id: "pat-2", name: "old token", revoked_at: "2026-07-15T00:00:00Z" })]),
    });
    renderSection();

    expect(await screen.findByText("ci agent")).toBeInTheDocument();
    expect(screen.getByText("old token")).toBeInTheDocument();
    expect(screen.getByText(/revoked/)).toBeInTheDocument();

    const revokeButtons = screen.getAllByRole("button", { name: /^revoke$/i });
    expect(revokeButtons).toHaveLength(2);
    expect(revokeButtons.filter((b) => (b as HTMLButtonElement).disabled)).toHaveLength(1);
  });

  it("revokes a personal access token after arming the confirm step", async () => {
    mocks.get.mockResolvedValue({ data: patList([token()]) });
    mocks.del.mockResolvedValue({ data: patList([]) });
    const user = userEvent.setup();
    renderSection();

    const revoke = await screen.findByRole("button", { name: /^revoke$/i });
    await user.click(revoke);
    expect(mocks.del).not.toHaveBeenCalled();

    await user.click(screen.getByRole("button", { name: /^confirm$/i }));

    await vi.waitFor(() => expect(mocks.del).toHaveBeenCalledWith("/api/auth/tokens/pat-1"));
  });

  it("does not revoke a personal access token when the confirm step is cancelled", async () => {
    mocks.get.mockResolvedValue({ data: patList([token()]) });
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: /^revoke$/i }));
    await user.click(screen.getByRole("button", { name: /^cancel$/i }));

    expect(mocks.del).not.toHaveBeenCalled();
    expect(await screen.findByRole("button", { name: /^revoke$/i })).toBeInTheDocument();
  });
});
