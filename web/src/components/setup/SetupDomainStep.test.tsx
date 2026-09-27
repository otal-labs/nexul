import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { SetupDomainStep } from "@/components/setup/SetupDomainStep";

const mocks = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
  toast: { success: vi.fn(), error: vi.fn() },
}));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, post: mocks.post, put: mocks.put },
  errorMessage: (e: { response?: { data?: { message?: string } }; message?: string }) =>
    e?.response?.data?.message ?? e?.message ?? "Something went wrong",
}));

vi.mock("sonner", () => ({ toast: mocks.toast }));

type User = ReturnType<typeof userEvent.setup>;

const cloudflare = (configured: boolean) => ({
  connector: {
    id: "cloudflare",
    name: "Cloudflare",
    description: "DNS and tunnels",
    category: "infrastructure",
    icon: "cloudflare",
    manual: [{ key: "api_token", label: "API token", secret: true }],
    checks: [{ key: "zone_dns", label: "Zone DNS edit" }],
  },
  status: { configured },
  available: true,
  app_configured: false,
});

const renderStep = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <SetupDomainStep />
    </QueryClientProvider>,
  );
};

const step = (title: RegExp) => screen.getByRole("heading", { name: title }).closest("li")!;

const choosePath = async (user: User, label: RegExp) => {
  await user.click(screen.getByRole("radio", { name: label }));
  await user.click(screen.getByRole("button", { name: /^continue$/i }));
};

describe("SetupDomainStep", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.post.mockReset();
    mocks.put.mockReset();
  });

  describe("I already have HTTPS", () => {
    it("refuses an http:// address inline and never asks the server", async () => {
      renderStep();
      const user = userEvent.setup();
      await choosePath(user, /i already have https/i);

      await user.type(screen.getByLabelText(/https address/i), "http://deploy.example.com");
      await user.click(screen.getByRole("button", { name: /check address/i }));

      expect(await screen.findByRole("alert")).toHaveTextContent(/https/i);
      expect(mocks.put).not.toHaveBeenCalled();
    });

    it("shows the server's reason with a retry when the address does not answer as Nexul", async () => {
      mocks.put.mockRejectedValue({ response: { status: 400, data: { message: "got 502 from the proxy" } } });
      renderStep();
      const user = userEvent.setup();
      await choosePath(user, /i already have https/i);

      expect(screen.getByText(`http://${window.location.host}`)).toBeInTheDocument();
      await user.type(screen.getByLabelText(/https address/i), "https://deploy.example.com/");
      await user.click(screen.getByRole("button", { name: /check address/i }));

      expect(await screen.findByRole("alert")).toHaveTextContent(/got 502 from the proxy/i);
      expect(mocks.put).toHaveBeenCalledWith("/api/setup/instance-url", { url: "https://deploy.example.com" });
      expect(step(/go live/i)).toHaveAttribute("data-state", "active");

      mocks.put.mockResolvedValue({ data: { instance_url: "https://deploy.example.com" } });
      await user.click(screen.getByRole("button", { name: /try again/i }));
      expect(await screen.findByText(/saved https:\/\/deploy\.example\.com/i)).toBeInTheDocument();
      expect(mocks.put).toHaveBeenCalledTimes(2);
    });
  });

  describe("Cloudflare tunnel", () => {
    it("will not deploy the tunnel until a Cloudflare token is connected", async () => {
      mocks.get.mockImplementation(async (url: string) => {
        if (url === "/api/connectors") return { data: [cloudflare(false)] };
        return { data: [] };
      });
      renderStep();
      const user = userEvent.setup();
      await choosePath(user, /cloudflare tunnel/i);

      expect(await screen.findByRole("button", { name: /^connect$/i })).toBeInTheDocument();
      expect(step(/connect cloudflare/i)).toHaveAttribute("data-state", "active");
      expect(step(/deploy the tunnel/i)).toHaveAttribute("data-state", "upcoming");
      expect(mocks.get).not.toHaveBeenCalledWith("/api/machines");
    });

    it("deploys on the only machine, leaves the project to the server, routes the hostname and finishes", async () => {
      mocks.get.mockImplementation(async (url: string) => {
        if (url === "/api/connectors") return { data: [cloudflare(true)] };
        if (url === "/api/dns/zones") return { data: [{ id: "z1", name: "example.com", status: "active" }] };
        if (url === "/api/machines") return { data: [{ id: "m1", name: "instance" }] };
        if (url === "/api/dns/tunnels/t1/status") return { data: { id: "t1", name: "instance", status: "healthy" } };
        if (url === "/api/services/svc-1/deploys") return { data: [{ id: "d1", status: "healthy", created_at: "2026-09-27" }] };
        return { data: [] };
      });
      mocks.post.mockImplementation(async (url: string) => {
        if (url === "/api/dns/tunnels") return { data: { id: "t1", name: "instance", account_id: "" } };
        if (url === "/api/dns/tunnels/t1/agent") return { data: { service_id: "svc-1" } };
        if (url === "/api/dns/tunnels/t1/route") return { data: { id: "t1" } };
        return { data: { detail: "ok" } };
      });
      mocks.put.mockResolvedValue({ data: { instance_url: "https://app.example.com" } });
      renderStep();
      const user = userEvent.setup();
      await choosePath(user, /cloudflare tunnel/i);

      await vi.waitFor(() => expect(step(/connect cloudflare/i)).toHaveAttribute("data-state", "done"));
      await user.click(await screen.findByRole("button", { name: /deploy tunnel/i }));
      expect(mocks.post).toHaveBeenCalledWith(
        "/api/dns/tunnels/t1/agent",
        expect.objectContaining({ project_id: "", target: "instance" }),
      );
      // Before sign-in there is no workspace, so projects are never listed.
      expect(mocks.get).not.toHaveBeenCalledWith("/api/projects", expect.anything());

      await user.type(await screen.findByLabelText(/subdomain/i), "app");
      await user.click(screen.getByRole("button", { name: /point hostname at the tunnel/i }));
      await user.click(await screen.findByRole("button", { name: /^continue$/i }));

      await vi.waitFor(() =>
        expect(mocks.put).toHaveBeenCalledWith("/api/setup/instance-url", { url: "https://app.example.com" }),
      );
      expect(within(step(/point your hostname/i)).getByText("app.example.com")).toBeInTheDocument();
    });
  });

  describe("Reverse proxy", () => {
    it("shows the records to create and what the domain resolves to until it points here", async () => {
      mocks.get.mockImplementation(async (url: string) => {
        if (url === "/api/setup/public-address") return { data: { ipv4: "203.0.113.10", ipv6: "2001:db8::1" } };
        if (url === "/api/dns/resolve") return { data: { addresses: ["198.51.100.7"] } };
        return { data: [] };
      });
      renderStep();
      const user = userEvent.setup();
      await choosePath(user, /reverse proxy/i);

      await user.type(await screen.findByLabelText(/^domain$/i), "deploy.example.com");
      const records = screen.getByRole("list", { name: /dns records/i });
      expect(within(records).getByText("203.0.113.10")).toBeInTheDocument();
      expect(within(records).getByText("2001:db8::1")).toBeInTheDocument();
      await user.click(screen.getByRole("button", { name: /i've added the record/i }));

      expect(await screen.findByText(/resolves to 198\.51\.100\.7 now/i)).toBeInTheDocument();
      expect(mocks.get).toHaveBeenCalledWith("/api/dns/resolve", { params: { host: "deploy.example.com" } });
      expect(mocks.post).not.toHaveBeenCalledWith("/api/dns/instance-proxy", expect.anything());
    });

    it("polls until the domain points here, deploys Traefik, then finishes while the certificate is issued", async () => {
      let answers: string[] = [];
      let release: (value: unknown) => void = () => {};
      mocks.get.mockImplementation(async (url: string) => {
        if (url === "/api/setup/public-address") return { data: { ipv4: "203.0.113.10" } };
        if (url === "/api/dns/resolve") return { data: { addresses: answers } };
        if (url === "/api/services/svc-9/deploys") return { data: [{ id: "d9", status: "healthy", created_at: "2026-09-27" }] };
        return { data: [] };
      });
      mocks.post.mockResolvedValue({ data: { service_id: "svc-9" } });
      mocks.put.mockImplementation(() => new Promise((resolve) => (release = resolve)));
      renderStep();
      const user = userEvent.setup();
      await choosePath(user, /reverse proxy/i);

      await user.type(await screen.findByLabelText(/^domain$/i), "deploy.example.com");
      await user.click(screen.getByRole("button", { name: /advanced options/i }));
      await user.type(screen.getByLabelText(/email/i), "ops@example.com");
      await user.click(screen.getByRole("button", { name: /i've added the record/i }));
      expect(await screen.findByText(/does not resolve yet/i)).toBeInTheDocument();

      answers = ["203.0.113.10"];
      await vi.waitFor(
        () =>
          expect(mocks.post).toHaveBeenCalledWith("/api/dns/instance-proxy", {
            domain: "deploy.example.com",
            email: "ops@example.com",
          }),
        { timeout: 8000 },
      );
      await vi.waitFor(() =>
        expect(mocks.put).toHaveBeenCalledWith("/api/setup/instance-url", { url: "https://deploy.example.com" }),
      );
      expect(await screen.findByText(/let's encrypt can take a couple of minutes/i)).toBeInTheDocument();
      expect(mocks.post).toHaveBeenCalledTimes(1);
      release({ data: { instance_url: "https://deploy.example.com" } });
    }, 15000);
  });
});
