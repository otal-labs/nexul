import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { WizardReachStep } from "@/components/wizard/WizardReachStep";
import { useProjectWizardStore } from "@/stores/projectWizardStore";
import { pickOption } from "@/test/pickOption";

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), errorMessage: vi.fn(() => "") }));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, post: mocks.post },
  errorMessage: mocks.errorMessage,
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const zones = [{ id: "z1", name: "example.com", status: "active" }];
const services = [
  { id: "c1", stack_id: "stack-1", name: "web", declared: { ports: ["80"] }, status: "running" },
];
const gateways = [
  { id: "g1", kind: "tunnel", docker_network: "api_default", zone_id: "z1", zone: "example.com", created_at: "", updated_at: "" },
];

const renderStep = (onDone = vi.fn(), onSkip = vi.fn()) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <WizardReachStep onDone={onDone} onSkip={onSkip} />
    </QueryClientProvider>,
  );
  return { onDone, onSkip };
};

beforeEach(() => {
  mocks.get.mockReset();
  mocks.post.mockReset();
  useProjectWizardStore.getState().reset();
  useProjectWizardStore.getState().setStackId("stack-1");
  useProjectWizardStore.getState().setCandidate({
    kind: "compose",
    path: "docker-compose.yml",
    name: "api",
    services: [],
    reachable: { service: "web", port: 80 },
  });
  mocks.get.mockImplementation(async (url: string) => {
    if (url === "/api/dns/zones") return { data: zones };
    if (url === "/api/stacks/stack-1/services") return { data: services };
    if (url === "/api/dns/gateways") return { data: gateways };
    if (url === "/api/stacks/stack-1/deploys") return { data: [{ id: "d1", status: "healthy", created_at: "", log: "" }] };
    return { data: [] };
  });
});

describe("WizardReachStep", () => {
  it("exposes the reachable service through the gateway the server picked", async () => {
    mocks.post.mockResolvedValue({
      data: { id: "exp1", gateway_id: "g1", hostname: "app.example.com", service: "web", port: 80, zone_id: "z1", zone: "example.com", created_at: "", updated_at: "" },
    });
    const { onDone } = renderStep();
    const user = userEvent.setup();

    expect(await screen.findByText(/deploy is healthy/i)).toBeInTheDocument();
    await user.type(screen.getByLabelText("Subdomain"), "app");
    await pickOption(user, "Zone", "example.com");
    expect(screen.getByText("app.example.com")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /expose service/i }));

    await vi.waitFor(() => expect(onDone).toHaveBeenCalled());
    expect(mocks.post).toHaveBeenCalledWith("/api/dns/exposures", {
      hostname: "app.example.com",
      service_id: "c1",
      port: 80,
      zone_id: "z1",
      zone: "example.com",
    });
    expect(useProjectWizardStore.getState().exposureHostname).toMatch(/tunnel gateway/);
  });

  it("skips the reach step without creating an exposure", async () => {
    const { onSkip } = renderStep();
    const user = userEvent.setup();

    await user.click(await screen.findByRole("button", { name: /skip for now/i }));
    expect(onSkip).toHaveBeenCalled();
    expect(mocks.post).not.toHaveBeenCalled();
  });
});
