import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { PairComputerDialog } from "@/components/pairing/PairComputerDialog";
import { Button } from "@/components/ui/button";
import { setCachedTunnelStatus } from "@/hooks/PairingHooks";
import { useSetupActivityStore } from "@/stores/setupActivityStore";
import type { Computer, ComputerSetup, SetupTurnState } from "@/models/Pairing";

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

const apiError = (body: Record<string, string>, errors?: Record<string, string[]>) =>
  Object.assign(new Error("request failed"), { response: { data: { ...body, ...(errors && { errors }) } } });

const emptySetup: ComputerSetup = { computer_id: "c1", confirmed_at: null, providers: [], turns: [] };

const renderDialog = (setupFor?: Computer) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <PairComputerDialog setupFor={setupFor} trigger={<Button type="button">{setupFor ? "Set up" : "Pair a computer"}</Button>} />
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

const reachPairStep = async (user: ReturnType<typeof userEvent.setup>, client: QueryClient) => {
  mocks.post.mockResolvedValueOnce({ data: created });
  await nameTheComputer(user);
  await screen.findByText(/waiting for connection/i);
  act(() => setCachedTunnelStatus(client, { computer_id: "c1", tunnel: "healthy", harness_reachable: true }));
  await user.click(await screen.findByRole("button", { name: /^next$/i }));
  await screen.findByLabelText(/one-time pairing token/i);
};

describe("PairComputerDialog", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.post.mockReset();
    mocks.get.mockImplementation(async (url: string) => {
      if (url.endsWith("/tunnel/token")) return { data: { token: "eyJ-connector-token" } };
      if (url.endsWith("/tunnel/status")) return { data: { tunnel: "inactive", harness_reachable: false } };
      if (url.endsWith("/setup")) return { data: emptySetup };
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
    expect(await screen.findByLabelText(/one-time pairing token/i)).toBeInTheDocument();
  });

  it("pairs T3 Code over the verified hostname with only the token typed, then moves on to Set up", async () => {
    const user = userEvent.setup();
    const client = renderDialog();
    await reachPairStep(user, client);

    expect(screen.getByLabelText(/^name$/i)).toHaveValue("Work laptop");
    expect(screen.getByLabelText(/^name$/i)).toHaveAttribute("readonly");
    expect(screen.getByLabelText(/t3 server url/i)).toHaveValue("https://work-laptop-ab12cd34.example.com");
    expect(screen.getByLabelText(/t3 server url/i)).toHaveAttribute("readonly");
    expect(screen.getByText("t3 pair")).toBeInTheDocument();

    mocks.post.mockResolvedValueOnce({ data: { ...created, token_expires_at: "2026-10-24T00:00:00Z", harness_version: "0.0.40" } });
    await user.type(screen.getByLabelText(/one-time pairing token/i), "t3-pair-token");
    await user.click(screen.getByRole("button", { name: /pair t3 code/i }));

    await waitFor(() => expect(mocks.post).toHaveBeenLastCalledWith("/api/pairing/computers/c1/pair", { token: "t3-pair-token" }));
    expect(await screen.findByRole("button", { name: /start setup/i })).toBeInTheDocument();
    expect(screen.getByText(/one short setup turn per provider on work laptop/i)).toBeInTheDocument();
    expect(screen.getByRole("checkbox", { name: /~\/\.claude\/skills\//i })).toBeChecked();
    expect(screen.getByRole("checkbox", { name: /~\/\.agents\/skills\//i })).toBeChecked();
    expect(screen.getByRole("button", { name: /^done$/i })).toBeInTheDocument();
  });

  it("shows a refused token on the token field and an unreachable harness on the URL field", async () => {
    const user = userEvent.setup();
    const client = renderDialog();
    await reachPairStep(user, client);

    const refused = "the harness refused this token, run t3 pair for a fresh one";
    mocks.post.mockRejectedValueOnce(apiError({ message: refused, code: "INVALID" }, { token: [refused] }));
    await user.type(screen.getByLabelText(/one-time pairing token/i), "stale");
    await user.click(screen.getByRole("button", { name: /pair t3 code/i }));
    expect(await screen.findByText(refused)).toBeInTheDocument();
    expect(screen.getByLabelText(/one-time pairing token/i)).toHaveAttribute("aria-invalid", "true");

    const unreachable = "couldn't reach the harness";
    mocks.post.mockRejectedValueOnce(apiError({ message: unreachable, code: "RETRYABLE" }, { server_url: [unreachable] }));
    await user.click(screen.getByRole("button", { name: /pair t3 code/i }));
    expect(await screen.findByText(unreachable)).toBeInTheDocument();
    expect(screen.getByLabelText(/t3 server url/i)).toHaveAttribute("aria-invalid", "true");
    expect(screen.queryByText(/is paired/i)).not.toBeInTheDocument();
  });

  it("pairs a machine the server can already reach by URL, from Advanced", async () => {
    const user = userEvent.setup();
    renderDialog();

    await user.click(screen.getByRole("button", { name: /pair a computer/i }));
    await user.click(await screen.findByRole("button", { name: /advanced options/i }));
    await user.click(screen.getByRole("button", { name: /pair by url/i }));

    await user.click(screen.getByRole("button", { name: /pair t3 code/i }));
    expect(await screen.findByText(/computer name is required/i)).toBeInTheDocument();
    expect(mocks.post).not.toHaveBeenCalled();

    await user.type(screen.getByLabelText(/^name$/i), "VPS");
    await user.type(screen.getByLabelText(/t3 server url/i), "https://vps.example.com");
    await user.type(screen.getByLabelText(/one-time pairing token/i), "t3-pair-token");
    mocks.post.mockResolvedValueOnce({ data: { ...created, id: "c2", name: "VPS", tunnel: undefined } });
    await user.click(screen.getByRole("button", { name: /pair t3 code/i }));

    await waitFor(() =>
      expect(mocks.post).toHaveBeenCalledWith("/api/pairing/computers", {
        name: "VPS",
        server_url: "https://vps.example.com",
        token: "t3-pair-token",
      }),
    );
    expect(await screen.findByText(/one short setup turn per provider on vps/i)).toBeInTheDocument();
  });

  it("shows a failure with no field under the form", async () => {
    const user = userEvent.setup();
    renderDialog();

    await user.click(screen.getByRole("button", { name: /pair a computer/i }));
    await user.click(await screen.findByRole("button", { name: /advanced options/i }));
    await user.click(screen.getByRole("button", { name: /pair by url/i }));
    await user.type(screen.getByLabelText(/^name$/i), "VPS");
    await user.type(screen.getByLabelText(/t3 server url/i), "https://vps.example.com");
    await user.type(screen.getByLabelText(/one-time pairing token/i), "t3-pair-token");
    mocks.post.mockRejectedValueOnce(apiError({ message: "internal error", code: "INTERNAL" }));
    await user.click(screen.getByRole("button", { name: /pair t3 code/i }));

    expect(await screen.findByRole("alert")).toHaveTextContent("internal error");
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

describe("PairComputerDialog opened at Set up", () => {
  const paired: Computer = { ...created, token_expires_at: "2026-10-24T00:00:00Z", harness_version: "0.0.40" };
  const turn = (provider: string, name: string, state: SetupTurnState, status: string, run_id = "r0", turn_id = `t-${provider}`) => ({
    run_id,
    turn_id,
    provider,
    provider_name: name,
    state,
    status,
    updated_at: "2026-09-24T00:00:00Z",
  });
  let setup: ComputerSetup;

  beforeEach(() => {
    mocks.get.mockReset();
    mocks.post.mockReset();
    useSetupActivityStore.setState({ lines: {} });
    setup = emptySetup;
    mocks.get.mockImplementation(async (url: string) => {
      if (url.endsWith("/setup")) return { data: setup };
      return { data: { computers: [] } };
    });
  });

  it("starts at Set up with the earlier steps locked, shows each provider's newest turn, and retries only the failed one", async () => {
    setup = {
      ...emptySetup,
      confirmed_at: "2026-09-24T00:00:00Z",
      turns: [turn("codex", "Codex", "confirmed", "Confirmed with 12 skills"), turn("opencode", "opencode", "failed", "No result within 10m0s")],
    };
    mocks.post.mockResolvedValue({ data: { run_id: "r1", computer_id: "c1", providers: [{ provider: "opencode", name: "opencode" }] } });
    const user = userEvent.setup();
    renderDialog(paired);

    await user.click(screen.getByRole("button", { name: /^set up$/i }));
    expect(await screen.findByRole("heading", { name: /set up work laptop/i })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: /connect/i })).toBeDisabled();
    expect(await screen.findByText("Confirmed with 12 skills")).toBeInTheDocument();
    expect(screen.getByText("No result within 10m0s")).toBeInTheDocument();
    expect(screen.getByText("1/2 confirmed")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /retry/i }));
    await waitFor(() => expect(mocks.post).toHaveBeenCalledWith("/api/pairing/computers/c1/setup/providers/opencode/retry"));
    expect(mocks.post).toHaveBeenCalledTimes(1);
  });

  it("lists every provider of a started run at once, with the running turn's commentary folded under it", async () => {
    mocks.post.mockImplementation(async () => {
      setup = { ...emptySetup, turns: [turn("claudeagent", "Claude", "running", "Connecting Nexul and installing skills", "r1", "t2")] };
      return {
        data: {
          run_id: "r1",
          computer_id: "c1",
          providers: [
            { provider: "claudeagent", name: "Claude" },
            { provider: "codex", name: "Codex" },
          ],
        },
      };
    });
    const user = userEvent.setup();
    renderDialog(paired);

    await user.click(screen.getByRole("button", { name: /^set up$/i }));
    await user.click(await screen.findByRole("button", { name: /start setup/i }));

    await waitFor(() => expect(mocks.post).toHaveBeenCalledWith("/api/pairing/computers/c1/setup/runs"));
    expect(await screen.findByText("Connecting Nexul and installing skills")).toBeInTheDocument();
    expect(screen.getByText("Waiting for its turn")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /re-run setup/i })).toBeDisabled();

    act(() => useSetupActivityStore.getState().push("t2", "Wrote ~/.codex/config.toml"));
    await user.click(await screen.findByRole("button", { name: /agent steps \(1\)/i }));
    expect(screen.getByText("Wrote ~/.codex/config.toml")).toBeInTheDocument();
  });
});
