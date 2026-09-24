import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { PairComputerDialog } from "@/components/pairing/PairComputerDialog";
import { setCachedTunnelStatus } from "@/hooks/PairingHooks";

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn() }));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, post: mocks.post },
  errorMessage: (e: { response?: { data?: { message?: string } } }) => e?.response?.data?.message ?? "Something went wrong",
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const created = {
  id: "c1",
  name: "Work laptop",
  server_url: "https://work-laptop-ab12cd34.example.com",
  token_expires_at: "0001-01-01T00:00:00Z",
  kind: "t3code",
  harness_version: "",
  created_at: "2026-09-24T00:00:00Z",
  updated_at: "2026-09-24T00:00:00Z",
  tunnel: { tunnel_id: "tun-1", hostname: "work-laptop-ab12cd34.example.com" },
};

const apiError = (body: Record<string, string>) => Object.assign(new Error("request failed"), { response: { data: body } });

const renderDialog = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <PairComputerDialog />
      </MemoryRouter>
    </QueryClientProvider>,
  );
  return client;
};

const nameTheComputer = async (user: ReturnType<typeof userEvent.setup>) => {
  await user.click(screen.getByRole("button", { name: /pair a computer/i }));
  await user.type(await screen.findByLabelText(/computer name/i), "Work laptop");
  await user.click(screen.getByRole("button", { name: /create tunnel/i }));
};

describe("PairComputerDialog", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.post.mockReset();
    mocks.get.mockImplementation(async (url: string) => {
      if (url.endsWith("/tunnel/token")) return { data: { token: "eyJ-connector-token" } };
      if (url.endsWith("/tunnel/status")) return { data: { tunnel: "inactive", harness_reachable: false } };
      return { data: { computers: [] } };
    });
  });

  it("creates the tunnel, shows the commands, and unlocks Next once both checks pass", async () => {
    mocks.post.mockResolvedValue({ data: created });
    const user = userEvent.setup();
    const client = renderDialog();

    await nameTheComputer(user);
    await waitFor(() => expect(mocks.post).toHaveBeenCalledWith("/api/pairing/computers/tunnel", { name: "Work laptop", port: 3773 }));

    expect(await screen.findByText(/sudo cloudflared service install eyJ-connector-token/)).toBeInTheDocument();
    expect(screen.getByText(/waiting for connection/i)).toBeInTheDocument();
    expect(screen.getByText("work-laptop-ab12cd34.example.com")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /^next$/i })).toBeDisabled();

    await user.click(screen.getByRole("tab", { name: /macos/i }));
    expect(screen.getByText("cloudflared service install eyJ-connector-token")).toBeInTheDocument();

    act(() => setCachedTunnelStatus(client, { computer_id: "c1", tunnel: "healthy", harness_reachable: false }));
    expect(await screen.findByText("Online")).toBeInTheDocument();
    expect(screen.getByText(/start t3 code on this computer/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /^next$/i })).toBeDisabled();

    act(() => setCachedTunnelStatus(client, { computer_id: "c1", tunnel: "healthy", harness_reachable: true, harness_version: "0.0.40" }));
    expect(await screen.findByText("Connected")).toBeInTheDocument();
    expect(screen.getByText("T3 Code 0.0.40")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /^next$/i }));
    expect(await screen.findByText(/pairing t3 code over this computer's hostname/i)).toBeInTheDocument();
  });

  it.each([
    ["zero_trust_disabled", /zero trust isn't enabled/i, /open zero trust/i],
    ["cloudflare_not_connected", /cloudflare isn't connected/i, /connect cloudflare/i],
  ])("explains a missing %s prerequisite with its fix and retries", async (reason, title, fix) => {
    mocks.post.mockRejectedValueOnce(apiError({ message: "missing", code: "INVALID", reason })).mockResolvedValueOnce({ data: created });
    const user = userEvent.setup();
    renderDialog();

    await nameTheComputer(user);
    const alert = await screen.findByRole("alert");
    expect(alert).toHaveTextContent(title);
    expect(screen.getByRole("link", { name: fix })).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /try again/i }));
    expect(await screen.findByText(/waiting for connection/i)).toBeInTheDocument();
    expect(mocks.post).toHaveBeenCalledTimes(2);
  });

  it("shows any other failure inline under the form", async () => {
    mocks.post.mockRejectedValue(apiError({ message: "the instance's host is in no Cloudflare zone", code: "INVALID" }));
    const user = userEvent.setup();
    renderDialog();

    await nameTheComputer(user);
    expect(await screen.findByRole("alert")).toHaveTextContent(/no cloudflare zone/i);
    expect(screen.queryByRole("button", { name: /try again/i })).not.toBeInTheDocument();
  });

  it("asks for a name and a valid port before creating anything", async () => {
    const user = userEvent.setup();
    renderDialog();

    await user.click(screen.getByRole("button", { name: /pair a computer/i }));
    await user.click(await screen.findByRole("button", { name: /advanced options/i }));
    const port = screen.getByLabelText(/t3 code port/i);
    await user.clear(port);
    await user.type(port, "70000");
    await user.click(screen.getByRole("button", { name: /create tunnel/i }));

    expect(await screen.findByText(/name this computer/i)).toBeInTheDocument();
    expect(screen.getByText(/port between 1 and 65535/i)).toBeInTheDocument();
    expect(mocks.post).not.toHaveBeenCalled();
  });

  it("starts over at the name step after closing", async () => {
    mocks.post.mockResolvedValue({ data: created });
    const user = userEvent.setup();
    renderDialog();

    await nameTheComputer(user);
    await screen.findByText(/waiting for connection/i);
    await user.click(screen.getByRole("button", { name: /close/i }));
    await user.click(screen.getByRole("button", { name: /pair a computer/i }));
    expect(await screen.findByLabelText(/computer name/i)).toBeInTheDocument();
  });
});
