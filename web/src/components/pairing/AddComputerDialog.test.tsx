import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AddComputerDialog } from "@/components/pairing/AddComputerDialog";
import { Button } from "@/components/ui/button";
import { pairingFollower } from "@/hooks/PairingHooks";
import type { ComputerFacts } from "@/models/ComputerFacts";
import type { Computer, ComputerSetup } from "@/models/Pairing";
import { followFrame } from "@/test/followFrame";

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn() }));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, post: mocks.post },
  errorMessage: (e: { response?: { data?: { message?: string } } }) => e?.response?.data?.message ?? "Something went wrong",
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const waiting: Computer = {
  id: "c9",
  name: "",
  server_url: "",
  token_expires_at: "0001-01-01T00:00:00Z",
  kind: "t3code",
  harness_version: "",
  created_at: "2026-10-10T00:00:00Z",
  updated_at: "2026-10-10T00:00:00Z",
};

const command = "curl -fsSL https://nexul.io/computer.sh | sudo sh -s -- eyJ.token.sig";

const enrollment = { computer: waiting, token: "eyJ.token.sig", expires_at: "2026-10-10T13:00:00Z", commands: { unix: command, windows: "" } };

const connected = { connected: true, last_seen: "2026-10-10T12:00:00Z" };

const facts = (t3: ComputerFacts["t3"]): ComputerFacts => ({
  hostname: "alice-laptop",
  os: "linux",
  arch: "amd64",
  t3,
  providers: [],
  projects: [],
});

const emptySetup: ComputerSetup = { computer_id: "c9", confirmed_at: null, providers: [], skipped_providers: [], models: {}, model_options: {}, folder: "", turns: [] };

// What the computer list answers next; each frame below refetches it.
let listed: Computer[] = [];

const renderDialog = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <AddComputerDialog trigger={<Button type="button">Add a computer</Button>} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
  return client;
};

const openDialog = (user: ReturnType<typeof userEvent.setup>) => user.click(screen.getByRole("button", { name: "Add a computer" }));

// The check row by its name, which comes before its state word (Paired names both).
const check = (name: string) => screen.getAllByText(name)[0]!.closest("li") as HTMLElement;

describe("AddComputerDialog", () => {
  beforeEach(() => {
    listed = [];
    mocks.get.mockReset();
    mocks.post.mockReset();
    mocks.post.mockResolvedValue({ data: enrollment });
    mocks.get.mockImplementation(async (url: string) => {
      if (url === "/api/pairing/computers") return { data: { computers: listed } };
      if (url.endsWith("/setup")) return { data: emptySetup };
      if (url.endsWith("/providers")) return { data: { providers: [] } };
      if (url.endsWith("/projects")) return { data: { projects: [] } };
      return { data: {} };
    });
  });

  it("makes a one-time command on open, says it needs sudo and who it installs for, and keeps Windows for later", async () => {
    const user = userEvent.setup();
    renderDialog();

    await openDialog(user);
    expect(await screen.findByText(command)).toBeInTheDocument();
    expect(mocks.post).toHaveBeenCalledWith("/api/pairing/computers/enrollments", {});
    expect(screen.getByText(/installs everything for the account that typed/i)).toBeInTheDocument();
    expect(screen.getByText(/never for root/i)).toBeInTheDocument();
    expect(screen.getByText(/on a mac, t3 code answers only while you're logged in/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Set up" })).toBeDisabled();

    await user.click(screen.getByRole("radio", { name: "Windows" }));
    expect(screen.getByText("Coming soon")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /copy install command/i })).not.toBeInTheDocument();
  });

  it("turns each check green as the computer's frames arrive, then opens Set up once paired", async () => {
    const user = userEvent.setup();
    const client = renderDialog();
    await openDialog(user);
    await screen.findByText(command);

    expect(within(check("Computer connected")).getByText("Waiting")).toBeInTheDocument();
    expect(within(check("T3 Code found")).getByText("Waiting")).toBeInTheDocument();
    expect(within(check("Paired")).getByText("Waiting")).toBeInTheDocument();

    listed = [{ ...waiting, name: "alice-laptop", runner: connected }];
    await act(() => followFrame(pairingFollower, "runner.personal_changed", { computer_id: "c9", state: "connected" }, client));
    expect(await within(check("Computer connected")).findByText("Connected")).toBeInTheDocument();
    expect(within(check("T3 Code found")).getByText("Looking")).toBeInTheDocument();

    listed = [{ ...listed[0]!, facts: facts({ state: "answering", version: "0.0.46", port: 3773, install: "service" }) }];
    await act(() => followFrame(pairingFollower, "computer.facts_changed", { computer_id: "c9" }, client));
    expect(await within(check("T3 Code found")).findByText("Found")).toBeInTheDocument();
    expect(screen.getByText("T3 Code 0.0.46 · port 3773")).toBeInTheDocument();
    expect(screen.getByText("alice-laptop · linux amd64")).toBeInTheDocument();
    expect(within(check("Paired")).getByText("Pairing")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Set up" })).toBeDisabled();

    listed = [{ ...listed[0]!, server_url: "http://c9.nexul-computer.invalid", token_expires_at: "2026-11-09T00:00:00Z", kind: "t3code-v2" }];
    await act(() => followFrame(pairingFollower, "computer.paired", { computer_id: "c9" }, client));
    expect(await within(check("Paired")).findByText("Paired")).toBeInTheDocument();
    expect(screen.getByText("All passed")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Set up" }));
    expect(await screen.findByRole("heading", { name: "Set up alice-laptop" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /^done$/i })).toBeInTheDocument();
  });

  it.each([
    ["went offline", { runner: { connected: false, last_seen: "2026-10-10T12:00:00Z" } }, "Computer connected", "Offline", /journalctl -u nexul-computer/],
    ["has a closed desktop app", { runner: connected, facts: facts({ state: "not_running", install: "desktop_app" }) }, "T3 Code found", "Closed", /open t3 code on the computer/i],
    ["has a stopped service", { runner: connected, facts: facts({ state: "not_running", install: "service", restart_error: "no user manager" }) }, "T3 Code found", "Not running", /no user manager/],
    ["has no T3 Code", { runner: connected, facts: facts({ state: "missing" }) }, "T3 Code found", "Missing", /without --no-t3/],
    ["serves T3 Code beyond loopback", { runner: connected, facts: facts({ state: "not_loopback" }) }, "T3 Code found", "Not local", /127\.0\.0\.1/],
    ["could not be paired", { runner: connected, facts: facts({ state: "answering" }), pair_error: "T3 Code on alice-laptop made no pairing token" }, "Paired", "Failed", /made no pairing token/],
  ])("names what to do when the computer %s", async (_, state, name, label, detail) => {
    const user = userEvent.setup();
    const client = renderDialog();
    await openDialog(user);
    await screen.findByText(command);

    listed = [{ ...waiting, ...state }];
    await act(() => followFrame(pairingFollower, "computer.pair_failed", { computer_id: "c9" }, client));
    expect(await within(check(name)).findByText(label)).toBeInTheDocument();
    expect(within(check(name)).getByText(detail)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Set up" })).toBeDisabled();
  });

  it("says why no command was made and makes one on Try again", async () => {
    mocks.post.mockRejectedValueOnce(Object.assign(new Error("failed"), { response: { data: { message: "set the instance URL in settings before adding a computer" } } }));
    const user = userEvent.setup();
    renderDialog();

    await openDialog(user);
    expect(await screen.findByText(/set the instance url/i)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Try again" }));
    expect(await screen.findByText(command)).toBeInTheDocument();
    await waitFor(() => expect(mocks.post).toHaveBeenCalledTimes(2));
  });
});
