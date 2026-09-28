import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { TunnelDeployStep } from "@/components/dns/TunnelDeployStep";
import { TunnelHostnameStep } from "@/components/dns/TunnelHostnameStep";
import { pickOption } from "@/test/pickOption";

const mocks = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  errorMessage: vi.fn((err: unknown) => (err instanceof Error ? err.message : String(err))),
  toast: { success: vi.fn(), error: vi.fn() },
}));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, post: mocks.post },
  errorMessage: mocks.errorMessage,
}));

vi.mock("sonner", () => ({ toast: mocks.toast }));

vi.mock("@/utils/RetryUtility", () => ({
  retry: (_attempts: number, _delay: number, task: () => Promise<unknown>) => task(),
}));

const zones = [
  { id: "z-law", name: "law.example", account_id: "acct-law", account_name: "Onlaw" },
  { id: "z-otal", name: "otal.dev", account_id: "acct-otal", account_name: "Otal Software" },
];

const wrap = (ui: ReactNode) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>{ui}</MemoryRouter>
    </QueryClientProvider>,
  );
};

describe("tunnel account and checks", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.post.mockReset();
    mocks.get.mockImplementation(async (url: string) => {
      if (url === "/api/dns/zones") return { data: zones };
      if (url === "/api/machines") return { data: [{ id: "m1", name: "prod-1", stack_root: "", first_seen: "", last_seen: "" }] };
      return { data: [] };
    });
  });

  it("asks which Cloudflare account to create the tunnel in when the token reaches several", async () => {
    mocks.post.mockImplementation(async (url: string) => {
      if (url === "/api/dns/tunnels") return { data: { id: "t1", name: "instance", account_id: "acct-otal" } };
      return { data: { service_id: "svc-1" } };
    });
    const user = userEvent.setup();
    wrap(<TunnelDeployStep onConnected={vi.fn()} />);

    await user.click(await screen.findByRole("button", { name: /deploy tunnel/i }));
    expect(await screen.findByText("Choose the Cloudflare account")).toBeInTheDocument();
    expect(mocks.post).not.toHaveBeenCalled();

    await pickOption(user, "Cloudflare account", "Otal Software");
    await user.click(screen.getByRole("button", { name: /deploy tunnel/i }));
    expect(mocks.post).toHaveBeenCalledWith("/api/dns/tunnels", { name: "instance", account_id: "acct-otal" });
  });

  it("hides the account choice when the token reaches one account", async () => {
    mocks.get.mockImplementation(async (url: string) => {
      if (url === "/api/dns/zones") return { data: [zones[1]] };
      if (url === "/api/machines") return { data: [{ id: "m1", name: "prod-1", stack_root: "", first_seen: "", last_seen: "" }] };
      return { data: [] };
    });
    mocks.post.mockResolvedValue({ data: { id: "t1", name: "instance", account_id: "acct-otal", service_id: "svc-1" } });
    const user = userEvent.setup();
    wrap(<TunnelDeployStep onConnected={vi.fn()} />);

    await user.click(await screen.findByRole("button", { name: /deploy tunnel/i }));
    expect(screen.queryByText("Cloudflare account")).not.toBeInTheDocument();
    expect(mocks.post).toHaveBeenCalledWith("/api/dns/tunnels", { name: "instance", account_id: "acct-otal" });
  });

  it("offers only the tunnel account's zones and holds Continue while a check fails", async () => {
    mocks.post.mockImplementation(async (url: string, _body: unknown, config?: { params?: { check?: string } }) => {
      if (url.endsWith("/route")) return { data: { id: "t1" } };
      if (config?.params?.check === "reachable") throw new Error("otal.dev answered HTTP 530");
      return { data: { detail: "fine" } };
    });
    const user = userEvent.setup();
    const deployment = { tunnelId: "t1", tunnelName: "instance", accountId: "acct-otal", serviceId: "svc-1", target: "prod-1" };
    wrap(<TunnelHostnameStep deployment={deployment} onDone={vi.fn()} />);

    // The only zone in the tunnel's account is preselected; the other account's zone is not offered.
    await user.type(await screen.findByLabelText(/subdomain/i), "nexul");
    expect(screen.getByText("nexul.otal.dev → this instance")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /point hostname at the tunnel/i }));
    expect(await screen.findByText("otal.dev answered HTTP 530")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /^continue$/i })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: /check again/i })).toBeInTheDocument();
  });
});
