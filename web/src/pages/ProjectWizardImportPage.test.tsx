import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ProjectWizardImportPage } from "@/pages/ProjectWizardImportPage";
import { pickOption } from "@/test/pickOption";

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), errorMessage: vi.fn(() => "") }));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, post: mocks.post },
  errorMessage: mocks.errorMessage,
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const machines = [{ id: "m-1", name: "prod", stack_root: "/data/nexul", first_seen: "", last_seen: "" }];
const projects = [{ id: "p-1", name: "Backend", prefix: "BE", position: 0, created_at: "", updated_at: "" }];
const report = {
  stacks: [{ project: "myapp", containers: [{ name: "myapp-web-1", image: "nginx", status: "running" }] }],
  standalone: [{ name: "redis", image: "redis:7", status: "running" }],
  gateways: [],
  networks: [],
};

const renderPage = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <ProjectWizardImportPage />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  mocks.get.mockReset();
  mocks.post.mockReset();
  mocks.get.mockImplementation(async (url: string) => {
    if (url === "/api/machines") return { data: machines };
    if (url === "/api/projects") return { data: projects };
    return { data: [] };
  });
});

describe("ProjectWizardImportPage", () => {
  it("discovers a machine, ticks everything by default, and imports the selection", async () => {
    mocks.post.mockImplementation(async (url: string) => {
      if (url === "/api/machines/m-1/discover") return { data: report };
      if (url === "/api/machines/m-1/import") return { data: { stacks: [{ id: "s-1", name: "myapp" }], gateways: [] } };
      return { data: {} };
    });
    renderPage();
    const user = userEvent.setup();

    await pickOption(user, "Machine", "prod");
    await user.click(screen.getByRole("button", { name: /discover/i }));

    expect(await screen.findByText("myapp")).toBeInTheDocument();
    expect(screen.getByText("myapp-web-1")).toBeInTheDocument();
    expect(screen.getByText("redis")).toBeInTheDocument();
    expect(screen.getAllByRole("checkbox").every((box) => box.getAttribute("data-state") === "checked")).toBe(true);

    await pickOption(user, "Project", "Backend");
    await user.click(screen.getByRole("button", { name: /^import$/i }));

    await screen.findByText(/imported 1 stack/i);
    expect(mocks.post).toHaveBeenCalledWith("/api/machines/m-1/import", {
      project_id: "p-1",
      stacks: [{ project: "myapp", containers: ["myapp-web-1"] }],
      standalone: ["redis"],
      gateways: [],
    });
    expect(screen.getByRole("link", { name: /view on the canvas/i })).toHaveAttribute("href", "/topology");
  });

  it("shows what a discovered tunnel already serves, and a describe failure on its own row", async () => {
    mocks.post.mockImplementation(async (url: string) => {
      if (url === "/api/machines/m-1/discover") {
        return {
          data: {
            ...report,
            gateways: [
              {
                name: "cloudflared-local",
                image: "cloudflare/cloudflared:latest",
                status: "running",
                tunnel_id: "tun-1",
                tunnel: {
                  id: "tun-1",
                  name: "home",
                  status: "healthy",
                  tracked: false,
                  routes: [{ hostname: "app.example.com", service: "http://web:80" }],
                  records: [{ name: "app.example.com", content: "tun-1.cfargotunnel.com" }],
                },
              },
              {
                name: "cloudflared-old",
                image: "cloudflare/cloudflared:latest",
                status: "running",
                tunnel_id: "tun-2",
                tunnel_error: "not found: tunnel tun-2",
              },
            ],
          },
        };
      }
      return { data: {} };
    });
    renderPage();
    const user = userEvent.setup();

    await pickOption(user, "Machine", "prod");
    await user.click(screen.getByRole("button", { name: /discover/i }));

    expect(await screen.findByText("cloudflared-local")).toBeInTheDocument();
    expect(screen.getByText("home")).toBeInTheDocument();
    expect(screen.getByRole("list", { name: /hostnames routed through home/i })).toHaveTextContent(
      "app.example.com → http://web:80",
    );
    expect(screen.getByRole("list", { name: /dns records pointing at home/i })).toHaveTextContent(
      "app.example.com CNAME tun-1.cfargotunnel.com",
    );
    expect(screen.getByText(/could not read the tunnel: not found: tunnel tun-2/i)).toBeInTheDocument();
  });

  it("reports each ticked gateway's adoption: hostnames now on the canvas, unmatched routes, or the reason", async () => {
    mocks.post.mockImplementation(async (url: string) => {
      if (url === "/api/machines/m-1/discover") {
        return {
          data: {
            ...report,
            gateways: [
              { name: "cloudflared-local", image: "cloudflare/cloudflared:latest", status: "running", tunnel_id: "tun-1" },
              { name: "cloudflared-old", image: "cloudflare/cloudflared:latest", status: "running" },
            ],
          },
        };
      }
      if (url === "/api/machines/m-1/import") {
        return {
          data: {
            stacks: [{ id: "s-1", name: "myapp" }],
            gateways: [
              { name: "cloudflared-local", gateway_id: "gw-1", exposed: ["api.example.com"], unmatched: ["web.example.com"] },
              { name: "cloudflared-old", exposed: [], unmatched: [], error: "no tunnel token on the container" },
            ],
          },
        };
      }
      return { data: {} };
    });
    renderPage();
    const user = userEvent.setup();

    await pickOption(user, "Machine", "prod");
    await user.click(screen.getByRole("button", { name: /discover/i }));
    await screen.findByText("cloudflared-local");
    await pickOption(user, "Project", "Backend");
    await user.click(screen.getByRole("button", { name: /^import$/i }));

    await screen.findByText(/imported 1 stack/i);
    expect(mocks.post).toHaveBeenCalledWith(
      "/api/machines/m-1/import",
      expect.objectContaining({ gateways: ["cloudflared-local", "cloudflared-old"] }),
    );
    expect(screen.getByRole("list", { name: /via cloudflared-local/i })).toHaveTextContent("api.example.com");
    expect(screen.getByText(/not linked.*web\.example\.com/i)).toBeInTheDocument();
    expect(screen.getByText(/not adopted as a gateway: no tunnel token/i)).toBeInTheDocument();
  });

  it("preselects the machine from ?machine=", async () => {
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <MemoryRouter initialEntries={["/wizard/project/import?machine=m-1"]}>
          <ProjectWizardImportPage />
        </MemoryRouter>
      </QueryClientProvider>,
    );
    expect(await screen.findByRole("combobox", { name: "Machine" })).toHaveTextContent("prod");
  });
});
