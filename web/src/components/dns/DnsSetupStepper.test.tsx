import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { DnsSetupStepper } from "@/components/dns/DnsSetupStepper";
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

const projects = [{ id: "p1", name: "Main", created_at: "2026-01-01" }];
const zones = [{ id: "z1", name: "example.com", status: "active" }];
const runners = [
  { id: "r1", name: "instance", connected: true, last_seen: "2026-09-03T00:00:00Z", running_job: null, version: "0.1.0" },
];

const mockLists = () =>
  mocks.get.mockImplementation(async (url: string) => {
    if (url === "/api/dns/zones") return { data: zones };
    if (url === "/api/projects") return { data: projects };
    if (url === "/api/runners") return { data: runners };
    return { data: [] };
  });

const renderStepper = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <DnsSetupStepper />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

const step = (title: RegExp) => screen.getByRole("heading", { name: title }).closest("li")!;

const choosePath = async (user: ReturnType<typeof userEvent.setup>, label: RegExp) => {
  await user.click(await screen.findByRole("radio", { name: label }));
  await user.click(screen.getByRole("button", { name: /^continue$/i }));
};

describe("DnsSetupStepper", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.post.mockReset();
    mocks.errorMessage.mockClear();
    mocks.toast.success.mockClear();
    mocks.toast.error.mockClear();
  });

  it("starts with the entry-path choice open and the later rungs locked", async () => {
    mockLists();
    renderStepper();
    expect(await screen.findByRole("radio", { name: /bare public address/i })).toBeInTheDocument();
    expect(screen.getByRole("radio", { name: /cloudflare tunnel/i })).toBeInTheDocument();
    expect(screen.getByRole("radio", { name: /reverse proxy/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /^continue$/i })).toBeDisabled();
    expect(step(/point your hostname/i)).toHaveAttribute("data-state", "upcoming");
    expect(step(/go live/i)).toHaveAttribute("data-state", "upcoming");
  });

  it("collapses the choice to a summary with Change once a path is picked", async () => {
    mockLists();
    renderStepper();
    const user = userEvent.setup();
    await choosePath(user, /bare public address/i);

    const first = step(/how should traffic reach/i);
    expect(first).toHaveAttribute("data-state", "done");
    expect(within(first).getByText("Bare public address")).toBeInTheDocument();
    expect(step(/point your hostname/i)).toHaveAttribute("data-state", "active");
    expect(await screen.findByRole("button", { name: /create instance record/i })).toBeInTheDocument();

    await user.click(within(first).getByRole("button", { name: /change/i }));
    expect(await screen.findByRole("radio", { name: /bare public address/i })).toBeInTheDocument();
    expect(step(/point your hostname/i)).toHaveAttribute("data-state", "upcoming");
  });

  it("deploys the tunnel first, waits for cloudflared to connect, then routes the hostname and unlocks Go live", async () => {
    let tunnelStatus = "inactive";
    mocks.get.mockImplementation(async (url: string) => {
      if (url === "/api/dns/zones") return { data: zones };
      if (url === "/api/projects") return { data: projects };
      if (url === "/api/runners") return { data: runners };
      if (url === "/api/dns/tunnels/t1/status") return { data: { id: "t1", name: "instance", status: tunnelStatus } };
      if (url === "/api/services/svc-1/deploys") return { data: [{ id: "d1", status: "healthy", created_at: "2026-09-07T10:00:00Z" }] };
      return { data: [] };
    });
    mocks.post.mockImplementation(async (url: string) => {
      if (url === "/api/dns/tunnels") return { data: { id: "t1", name: "instance" } };
      if (url === "/api/dns/tunnels/t1/agent") return { data: { service_id: "svc-1" } };
      if (url === "/api/dns/tunnels/t1/route") return { data: { id: "t1" } };
      return { data: {} };
    });
    renderStepper();
    const user = userEvent.setup();
    await choosePath(user, /cloudflare tunnel/i);

    expect(step(/deploy the tunnel/i)).toHaveAttribute("data-state", "active");
    expect(step(/point your hostname/i)).toHaveAttribute("data-state", "upcoming");
    expect(await screen.findByLabelText(/tunnel name/i)).toHaveValue("instance");
    await user.click(screen.getByRole("button", { name: /deploy tunnel/i }));

    expect(await screen.findByRole("status")).toHaveTextContent(/waiting for it to reach cloudflare/i);
    expect(mocks.post).toHaveBeenCalledWith("/api/dns/tunnels", { name: "instance" });
    expect(mocks.post).toHaveBeenCalledWith(
      "/api/dns/tunnels/t1/agent",
      expect.objectContaining({ project_id: "p1", target: "instance", docker_network: "nexul_default" }),
    );
    expect(mocks.post).not.toHaveBeenCalledWith("/api/dns/tunnels/t1/route", expect.anything());

    // cloudflared reports in; the next poll flips the rung.
    tunnelStatus = "healthy";
    expect(await screen.findByLabelText(/subdomain/i, {}, { timeout: 6000 })).toBeInTheDocument();
    expect(step(/deploy the tunnel/i)).toHaveAttribute("data-state", "done");
    expect(step(/point your hostname/i)).toHaveAttribute("data-state", "active");

    await user.type(screen.getByLabelText(/subdomain/i), "app");
    expect(screen.getByText("app.example.com → http://web:80")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /point hostname at the tunnel/i }));

    expect(await screen.findByRole("button", { name: /continue to nexul/i })).toBeInTheDocument();
    expect(mocks.post).toHaveBeenCalledWith(
      "/api/dns/tunnels/t1/route",
      expect.objectContaining({ hostname: "app.example.com", zone_id: "z1", service: "http://web:80" }),
    );
    expect(step(/go live/i)).toHaveAttribute("data-state", "active");
  }, 15000);

  it("shows the failed deploy with a retry instead of unlocking the hostname rung", async () => {
    mocks.get.mockImplementation(async (url: string) => {
      if (url === "/api/dns/zones") return { data: zones };
      if (url === "/api/projects") return { data: projects };
      if (url === "/api/runners") return { data: runners };
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
    renderStepper();
    const user = userEvent.setup();
    await choosePath(user, /cloudflare tunnel/i);
    await user.click(await screen.findByRole("button", { name: /deploy tunnel/i }));

    expect(await screen.findByRole("status")).toHaveTextContent(/deploy on instance failed/i);
    expect(screen.getByText("pull access denied")).toBeInTheDocument();
    expect(step(/point your hostname/i)).toHaveAttribute("data-state", "upcoming");
    await user.click(screen.getByRole("button", { name: /try again/i }));
    expect(await screen.findByRole("button", { name: /deploy tunnel/i })).toBeInTheDocument();
  });

  it("reveals the advanced fields on demand with the runner preselected", async () => {
    mockLists();
    renderStepper();
    const user = userEvent.setup();
    await choosePath(user, /cloudflare tunnel/i);

    await user.click(await screen.findByRole("button", { name: /advanced options/i }));
    expect(screen.getByRole("combobox", { name: "Runs on" })).toHaveTextContent("instance");
    expect(screen.getByLabelText(/docker network/i)).toHaveValue("nexul_default");
  });

  it("provisions a reverse proxy and creates the instance record", async () => {
    mockLists();
    mocks.post.mockImplementation(async (url: string) => {
      if (url === "/api/dns/reverse-proxy") return { data: { service_id: "svc-1" } };
      if (url === "/api/dns/instance-record") return { data: { id: "r1" } };
      return { data: {} };
    });
    renderStepper();
    const user = userEvent.setup();
    await choosePath(user, /reverse proxy/i);

    await pickOption(user, /record type/i, "AAAA");
    await user.type(screen.getByLabelText(/server address/i), "2001:db8::1");
    await user.click(screen.getByRole("button", { name: /set up reverse proxy/i }));

    expect(await screen.findByRole("button", { name: /continue to nexul/i })).toBeInTheDocument();
    expect(mocks.post).toHaveBeenCalledWith(
      "/api/dns/reverse-proxy",
      expect.objectContaining({ project_id: "p1", target: "instance" }),
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
    renderStepper();
    const user = userEvent.setup();
    await choosePath(user, /bare public address/i);

    await user.type(await screen.findByLabelText(/points to/i), "203.0.113.10");
    await user.click(screen.getByRole("button", { name: /create instance record/i }));

    await vi.waitFor(() => expect(mocks.toast.error).toHaveBeenCalledWith("hostname is not under zone"));
    expect(step(/point your hostname/i)).toHaveAttribute("data-state", "active");
    expect(step(/go live/i)).toHaveAttribute("data-state", "upcoming");
  });
});
