import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ComputersSection } from "@/components/settings/ComputersSection";
import { pairingFollower } from "@/hooks/PairingHooks";
import type { Computer } from "@/models/Pairing";
import { followFrame } from "@/test/followFrame";

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn() }));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, post: mocks.post },
  errorMessage: (e: { response?: { data?: { message?: string } } }) => e?.response?.data?.message ?? "Something went wrong",
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

vi.mock("@/components/settings/ComputerMCPToken", () => ({ ComputerMCPToken: () => null }));

const byURL: Computer = {
  id: "c1",
  name: "alice-laptop",
  server_url: "https://vps.example.com:3773",
  token_expires_at: "2026-11-01T00:00:00Z",
  kind: "t3code",
  harness_version: "0.0.45",
  created_at: "2026-08-01T00:00:00Z",
  position: 0,
  runs_at_once: 1,
  running: 0,
  chat_project_id: "",
  skills_outdated: false,
  updated_at: "2026-08-01T00:00:00Z",
};

const tunnelLaptop: Computer = {
  ...byURL,
  server_url: "https://alice-laptop-ab12cd34.example.com",
  tunnel: { tunnel_id: "tun-1", hostname: "alice-laptop-ab12cd34.example.com" },
};

// Paired through its app: the relay address, a fresh session, and no tunnel.
const moved: Computer = { ...byURL, server_url: "http://c1.nexul-computer.invalid", token_expires_at: "2026-11-09T00:00:00Z", kind: "t3code-v2" };

const command = "curl -fsSL https://nexul.io/computer.sh | sudo sh -s -- eyJ.token.sig";
const enrollment = { computer: tunnelLaptop, token: "eyJ.token.sig", expires_at: "2026-10-10T13:00:00Z", commands: { unix: command, windows: "irm" } };
const connected = { connected: true, last_seen: "2026-10-10T12:00:00Z", reconnecting: false };
const answering = { hostname: "alice-laptop", t3: { state: "answering" as const, port: 3773, version: "0.0.46" }, providers: [], projects: [] };

let listed: Computer[] = [];

const renderSection = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <ComputersSection />
      </QueryClientProvider>
    </MemoryRouter>,
  );
  return client;
};

const check = (name: string) => within(screen.getByRole("dialog")).getAllByText(name)[0]!.closest("li") as HTMLElement;

describe("MoveComputerDialog", () => {
  beforeEach(() => {
    listed = [tunnelLaptop];
    mocks.get.mockReset();
    mocks.post.mockReset();
    mocks.post.mockResolvedValue({ data: enrollment });
    mocks.get.mockImplementation(async (url: string) => {
      if (url === "/api/pairing/computers") return { data: { computers: listed } };
      if (url === "/api/pairing/presence") return { data: { computers: {} } };
      if (url.endsWith("/setup")) return { data: { computer_id: "c1", confirmed_at: "2026-08-02T00:00:00Z", providers: [], skipped_providers: [], turns: [] } };
      return { data: {} };
    });
  });

  it("moves a computer paired before the app: the command for its own id, the checks, then the cloudflared removal", async () => {
    listed = [byURL];
    const user = userEvent.setup();
    const client = renderSection();

    expect(await screen.findByText("Move this computer to the Nexul app")).toBeInTheDocument();
    expect(screen.getByText(/agent work can't run here until it moves/i)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Move" }));
    expect(await screen.findByText(command)).toBeInTheDocument();
    expect(mocks.post).toHaveBeenCalledWith("/api/pairing/computers/enrollments", { id: "c1" });
    expect(screen.getByRole("heading", { name: "Move alice-laptop to the Nexul app" })).toBeInTheDocument();
    expect(within(check("Paired")).getByText("Waiting")).toBeInTheDocument();

    listed = [{ ...byURL, runner: connected, facts: answering }];
    await act(() => followFrame(pairingFollower, "computer.facts_changed", { computer_id: "c1" }, client));
    expect(await within(check("T3 Code found")).findByText("Found")).toBeInTheDocument();
    expect(within(check("Paired")).getByText("Pairing"), "its old session is not the move").toBeInTheDocument();
    expect(screen.queryByText("sudo cloudflared service uninstall")).not.toBeInTheDocument();

    listed = [{ ...moved, runner: connected, facts: answering }];
    await act(() => followFrame(pairingFollower, "computer.paired", { computer_id: "c1" }, client));
    expect(await screen.findByRole("heading", { name: "alice-laptop moved to the Nexul app" })).toBeInTheDocument();
    expect(screen.getByText("sudo cloudflared service uninstall")).toBeInTheDocument();
    expect(screen.queryByText("Move this computer to the Nexul app"), "the row is the app's row now").not.toBeInTheDocument();
  });

  it("keeps a computer whose move failed on its tunnel, says why on the row, and pairs again from the dialog", async () => {
    listed = [{ ...tunnelLaptop, runner: connected, facts: answering, pair_error: "the harness refused this token" }];
    mocks.post.mockResolvedValue({ data: listed[0] });
    const user = userEvent.setup();
    renderSection();

    expect(await screen.findByText(/not moved yet: the harness refused this token/i)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Show progress" }));

    expect(within(check("Paired")).getByText("Failed")).toBeInTheDocument();
    expect(mocks.post, "the app is already there, so no new command").not.toHaveBeenCalled();
    await user.click(screen.getByRole("button", { name: "Pair now" }));
    expect(mocks.post).toHaveBeenCalledWith("/api/pairing/computers/c1/pair");
  });
});
