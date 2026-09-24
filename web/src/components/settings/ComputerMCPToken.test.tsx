import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ComputerMCPToken } from "@/components/settings/ComputerMCPToken";

const mocks = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  del: vi.fn(),
  errorMessage: vi.fn(),
}));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, post: mocks.post, delete: mocks.del },
  errorMessage: mocks.errorMessage,
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const token = {
  id: "pat-1",
  name: "Nexul MCP on Home",
  prefix: "abc123",
  created_at: "2026-09-24T00:00:00Z",
};

const renderToken = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <ComputerMCPToken computerId="c1" />
    </QueryClientProvider>,
  );
};

describe("ComputerMCPToken", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.post.mockReset();
    mocks.del.mockReset();
    mocks.errorMessage.mockReset();
  });

  it("shows an error when the token fails to load", async () => {
    mocks.get.mockRejectedValue(new Error("boom"));
    mocks.errorMessage.mockReturnValue("Token lookup failed");
    renderToken();
    expect(await screen.findByText("Token lookup failed")).toBeInTheDocument();
  });

  it("mints a token and shows its secret once", async () => {
    mocks.get.mockResolvedValue({ data: { mcp_token: null } });
    mocks.post.mockResolvedValue({ data: { ...token, token: "dep_secret" } });
    renderToken();

    await userEvent.click(await screen.findByRole("button", { name: /mint mcp token/i }));

    expect(mocks.post).toHaveBeenCalledWith("/api/pairing/computers/c1/mcp-token");
    expect(await screen.findByText("dep_secret")).toBeInTheDocument();
  });

  it("lists the computer's token and revokes it after a confirm", async () => {
    mocks.get.mockResolvedValue({ data: { mcp_token: token } });
    mocks.del.mockResolvedValue({ data: { status: "revoked" } });
    renderToken();

    expect(await screen.findByText(/Nexul MCP on Home · dep_…abc123/)).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Revoke MCP token" }));
    await userEvent.click(screen.getByRole("button", { name: "Revoke" }));

    await waitFor(() => expect(mocks.del).toHaveBeenCalledWith("/api/pairing/computers/c1/mcp-token"));
  });
});
