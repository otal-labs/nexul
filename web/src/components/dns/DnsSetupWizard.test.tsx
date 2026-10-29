import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { DnsSetupWizard } from "@/components/dns/DnsSetupWizard";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { pickOption } from "@/test/pickOption";

const mocks = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  errorMessage: vi.fn(),
  toast: { success: vi.fn(), error: vi.fn() },
}));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, post: mocks.post },
  errorMessage: mocks.errorMessage,
}));

vi.mock("sonner", () => ({ toast: mocks.toast }));

const zones = [{ id: "z1", name: "example.com", status: "active" }];
const machines = [
  { id: "m1", name: "prod-1", stack_root: "/data/nexul", first_seen: "2026-09-03T00:00:00Z", last_seen: "2026-09-03T00:00:00Z" },
];

const mockLists = () =>
  mocks.get.mockImplementation(async (url: string) => {
    if (url === "/api/dns/zones") return { data: zones };
    if (url === "/api/machines/deployable") return { data: machines };
    return { data: [] };
  });

const renderWizard = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <DnsSetupWizard />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

const rung = (label: string) => within(screen.getByRole("navigation", { name: "DNS setup steps" })).getByText(label).closest("li")!;

const choosePath = async (user: ReturnType<typeof userEvent.setup>, label: RegExp) => {
  await user.click(await screen.findByRole("radio", { name: label }));
  await user.click(screen.getByRole("button", { name: /^continue$/i }));
};

describe("DnsSetupWizard", () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  beforeEach(() => {
    useWorkspaceStore.setState({ selectedWorkspaceId: "w1" });
    mocks.get.mockReset();
    mocks.post.mockReset();
    mocks.errorMessage.mockClear();
    mocks.toast.success.mockClear();
    mocks.toast.error.mockClear();
  });

  it("starts with the entry-path choice open and the later rungs locked", async () => {
    mockLists();
    renderWizard();
    expect(await screen.findByRole("radio", { name: /bare public address/i })).toBeInTheDocument();
    expect(screen.getByRole("radio", { name: /cloudflare tunnel/i })).toBeInTheDocument();
    expect(screen.getByRole("radio", { name: /reverse proxy/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /^continue$/i })).toBeDisabled();
    expect(rung("Hostname")).toHaveAttribute("data-state", "locked");
    expect(rung("Go live")).toHaveAttribute("data-state", "locked");
  });

  it("moves on to the hostname, and reopens the path from the rail as a summary whose Change starts over", async () => {
    mockLists();
    renderWizard();
    const user = userEvent.setup();
    await choosePath(user, /bare public address/i);

    expect(rung("Path")).toHaveAttribute("data-state", "done");
    expect(rung("Hostname")).toHaveAttribute("data-state", "current");
    expect(await screen.findByRole("button", { name: /create instance record/i })).toBeInTheDocument();

    await user.click(within(rung("Path")).getByRole("button"));
    expect(screen.getByText("Bare public address")).toBeInTheDocument();
    expect(rung("Hostname")).toHaveAttribute("data-state", "unvisited");
    await user.click(screen.getByRole("button", { name: /change/i }));
    expect(await screen.findByRole("radio", { name: /bare public address/i })).toBeInTheDocument();
    expect(rung("Hostname")).toHaveAttribute("data-state", "locked");
  });

  it("deploys the tunnel first, waits for cloudflared to connect, then routes the hostname and unlocks Go live", async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    let tunnelStatus = "inactive";
    mocks.get.mockImplementation(async (url: string) => {
      if (url === "/api/dns/zones") return { data: zones };
      if (url === "/api/machines/deployable") return { data: machines };
      if (url === "/api/dns/tunnels/t1/status") return { data: { id: "t1", name: "instance", status: tunnelStatus } };
      if (url === "/api/services/svc-1/deploys") return { data: [{ id: "d1", status: "healthy", created_at: "2026-09-07T10:00:00Z" }] };
      return { data: [] };
    });
    mocks.post.mockImplementation(async (url: string) => {
      if (url === "/api/dns/tunnels") return { data: { id: "t1", name: "instance" } };
      if (url === "/api/dns/tunnels/t1/agent") return { data: { service_id: "svc-1" } };
      if (url === "/api/dns/tunnels/t1/route") return { data: { id: "t1" } };
      if (url === "/api/dns/tunnels/t1/verify") return { data: { detail: "checked" } };
      return { data: {} };
    });
    renderWizard();
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    await choosePath(user, /cloudflare tunnel/i);

    expect(rung("Tunnel")).toHaveAttribute("data-state", "current");
    expect(rung("Hostname")).toHaveAttribute("data-state", "locked");
    expect(await screen.findByLabelText(/tunnel name/i)).toHaveValue("instance");
    await user.click(screen.getByRole("button", { name: /deploy tunnel/i }));

    expect(await screen.findByRole("status")).toHaveTextContent(/waiting for it to reach cloudflare/i);
    expect(mocks.post).toHaveBeenCalledWith("/api/dns/tunnels", { name: "instance", account_id: "" });
    expect(mocks.post).toHaveBeenCalledWith(
      "/api/dns/tunnels/t1/agent",
      { tunnel_id: "t1", target: "prod-1", docker_network: "nexul_default" },
    );
    expect(mocks.post).not.toHaveBeenCalledWith("/api/dns/tunnels/t1/route", expect.anything());

    // cloudflared reports in; the next poll flips the rung.
    tunnelStatus = "healthy";
    await act(() => vi.advanceTimersByTimeAsync(3000));
    expect(await screen.findByLabelText(/subdomain/i)).toBeInTheDocument();
    expect(rung("Tunnel")).toHaveAttribute("data-state", "done");
    expect(rung("Hostname")).toHaveAttribute("data-state", "current");

    await user.type(screen.getByLabelText(/subdomain/i), "app");
    expect(screen.getByText("app.example.com → this instance")).toBeInTheDocument();
    expect(screen.queryByLabelText(/different local service/i)).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /point hostname at the tunnel/i }));

    // The ticker checks the route, the record and the hostname before Continue appears.
    await user.click(await screen.findByRole("button", { name: /^continue$/i }));
    for (const check of ["ingress", "record", "reachable"]) {
      expect(mocks.post).toHaveBeenCalledWith("/api/dns/tunnels/t1/verify", null, { params: { check } });
    }
    expect(await screen.findByRole("button", { name: /continue to nexul/i })).toBeInTheDocument();
    expect(mocks.post).toHaveBeenCalledWith(
      "/api/dns/tunnels/t1/route",
      expect.objectContaining({ hostname: "app.example.com", zone_id: "z1", service: "" }),
    );
    expect(rung("Go live")).toHaveAttribute("data-state", "current");
  });

  it("shows the failed deploy with a retry instead of unlocking the hostname rung", async () => {
    mocks.get.mockImplementation(async (url: string) => {
      if (url === "/api/dns/zones") return { data: zones };
      if (url === "/api/machines/deployable") return { data: machines };
      if (url === "/api/dns/tunnels/t1/status") return { data: { id: "t1", name: "instance", status: "inactive" } };
      if (url === "/api/services/svc-1/deploys") return { data: [{ id: "d1", status: "failed", created_at: "2026-09-07T10:00:00Z" }] };
      if (url === "/api/deploys/d1/log") return { data: [{ seq: 1, ts: 1, phase: "deploy", text: "pull access denied" }] };
      return { data: [] };
    });
    mocks.post.mockImplementation(async (url: string) => {
      if (url === "/api/dns/tunnels") return { data: { id: "t1", name: "instance" } };
      if (url === "/api/dns/tunnels/t1/agent") return { data: { service_id: "svc-1" } };
      return { data: {} };
    });
    renderWizard();
    const user = userEvent.setup();
    await choosePath(user, /cloudflare tunnel/i);
    await user.click(await screen.findByRole("button", { name: /deploy tunnel/i }));

    expect(await screen.findByRole("status")).toHaveTextContent(/deploy on prod-1 failed/i);
    expect(screen.getByText("pull access denied")).toBeInTheDocument();
    expect(rung("Hostname")).toHaveAttribute("data-state", "locked");
    await user.click(screen.getByRole("button", { name: /try again/i }));
    expect(await screen.findByRole("button", { name: /deploy tunnel/i })).toBeInTheDocument();
  });

  it("reveals the advanced fields on demand with the machine preselected", async () => {
    mockLists();
    renderWizard();
    const user = userEvent.setup();
    await choosePath(user, /cloudflare tunnel/i);

    await user.click(await screen.findByRole("button", { name: /advanced options/i }));
    expect(screen.getByRole("combobox", { name: "Machine" })).toHaveTextContent("prod-1");
    expect(screen.getByLabelText(/docker network/i)).toHaveValue("nexul_default");
  });

  it("provisions a reverse proxy and creates the instance record", async () => {
    mockLists();
    mocks.post.mockImplementation(async (url: string) => {
      if (url === "/api/dns/reverse-proxy") return { data: { service_id: "svc-1" } };
      if (url === "/api/dns/instance-record") return { data: { id: "r1" } };
      return { data: {} };
    });
    renderWizard();
    const user = userEvent.setup();
    await choosePath(user, /reverse proxy/i);

    await pickOption(user, /record type/i, "AAAA");
    await user.type(screen.getByLabelText(/server address/i), "2001:db8::1");
    await user.click(screen.getByRole("button", { name: /set up reverse proxy/i }));

    expect(await screen.findByRole("button", { name: /continue to nexul/i })).toBeInTheDocument();
    expect(mocks.post).toHaveBeenCalledWith(
      "/api/dns/reverse-proxy",
      { target: "prod-1", docker_network: "nexul_default" },
    );
    expect(mocks.post).toHaveBeenCalledWith(
      "/api/dns/instance-record",
      expect.objectContaining({ type: "AAAA", target: "2001:db8::1" }),
    );
  });

  it("keeps the hostname step open when the provider rejects the record", async () => {
    mockLists();
    mocks.post.mockRejectedValue(new Error("boom"));
    mocks.errorMessage.mockReturnValue("hostname is not under zone");
    renderWizard();
    const user = userEvent.setup();
    await choosePath(user, /bare public address/i);

    await user.type(await screen.findByLabelText(/points to/i), "203.0.113.10");
    await user.click(screen.getByRole("button", { name: /create instance record/i }));

    await vi.waitFor(() => expect(mocks.toast.error).toHaveBeenCalledWith("hostname is not under zone"));
    expect(rung("Hostname")).toHaveAttribute("data-state", "current");
    expect(rung("Go live")).toHaveAttribute("data-state", "locked");
  });
});
