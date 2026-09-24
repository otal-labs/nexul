import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { WizardBranchesStep } from "@/components/wizard/WizardBranchesStep";
import { useProjectWizardStore } from "@/stores/projectWizardStore";
import { pickOption } from "@/test/pickOption";

const mocks = vi.hoisted(() => ({ get: vi.fn(), patch: vi.fn(), errorMessage: vi.fn(() => "") }));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, patch: mocks.patch },
  errorMessage: mocks.errorMessage,
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const stackBase = { project_id: "p-1", machine: "prod", strategy: "compose", managed: true, created_at: "", updated_at: "" };
const web = { ...stackBase, id: "stack-1", name: "web", slug: "web", build_source: { repo_owner: "acme", repo_name: "web", branch: "main" } };
const qa = { ...stackBase, id: "stack-qa", name: "qa", slug: "qa" };
const container = (stackId: string, id: string, name: string, network: string) => ({
  id,
  stack_id: stackId,
  name,
  declared: {},
  status: "running",
  networks: [{ name: network }],
});
const exposure = { id: "e1", gateway_id: "g1", hostname: "app.example.com", service_id: "c1", service: "web", port: 80, zone_id: "z1", zone: "example.com", created_at: "", updated_at: "" };

const renderStep = () => {
  const onDone = vi.fn();
  const onSkip = vi.fn();
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <WizardBranchesStep onDone={onDone} onSkip={onSkip} />
    </QueryClientProvider>,
  );
  return { onDone, onSkip, user: userEvent.setup() };
};

const productionWarning = /uses production's services/i;

beforeEach(() => {
  mocks.get.mockReset();
  mocks.patch.mockReset();
  mocks.patch.mockImplementation(async (_url: string, body: unknown) => ({ data: body }));
  useProjectWizardStore.getState().reset();
  useProjectWizardStore.getState().setStackId("stack-1");
  mocks.get.mockImplementation(async (url: string) => {
    if (url === "/api/stacks/stack-1") return { data: web };
    if (url === "/api/stacks") return { data: [web, qa] };
    if (url === "/api/stacks/stack-1/services") return { data: [container("stack-1", "c1", "web", "web_default")] };
    if (url === "/api/stacks/stack-qa/services")
      return { data: [container("stack-qa", "c2", "postgres", "qa_default"), container("stack-qa", "c3", "redis", "qa_default")] };
    if (url === "/api/dns/exposures") return { data: [exposure] };
    if (url === "/api/dns/gateways") return { data: [{ id: "g1", kind: "tunnel", docker_network: "web_default", zone_id: "z1", zone: "example.com" }] };
    return { data: [] };
  });
});

describe("WizardBranchesStep", () => {
  it("seeds the default branch's row, switched on, with the service's hostname and network", async () => {
    renderStep();
    expect(await screen.findByText("main → app.example.com on web_default")).toBeInTheDocument();
    expect(screen.getByRole("switch", { name: "Every push to main redeploys it" })).toBeChecked();
    expect(screen.queryByText(productionWarning)).not.toBeInTheDocument();
  });

  it("saves a wildcard row with its hostname as a rule next to the default branch's", async () => {
    const { onDone, user } = renderStep();

    await user.click(await screen.findByRole("button", { name: /add branch/i }));
    await user.type(screen.getByLabelText("Branch"), "feature/*");
    expect(screen.getByText("feature/security-test → security-test.example.com on web_default")).toBeInTheDocument();
    expect(screen.getByText(productionWarning)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /save branches/i }));

    await vi.waitFor(() => expect(onDone).toHaveBeenCalled());
    expect(mocks.patch).toHaveBeenCalledWith(
      "/api/stacks/stack-1",
      expect.objectContaining({
        branch_deploy_rules: [
          { pattern: "main", docker_network: "web_default" },
          { pattern: "feature/*", docker_network: "web_default", hostname_template: "{branch}.example.com", port: 80 },
        ],
      }),
    );
    expect(useProjectWizardStore.getState().branchesSummary).toBe("main, feature/*");
  });

  it("labels a network without a gateway and disables the row's hostname on it", async () => {
    const { onDone, user } = renderStep();

    await user.click(await screen.findByRole("button", { name: /add branch/i }));
    await user.type(screen.getByLabelText("Branch"), "feature/*");
    await pickOption(user, "Network", "qa_default — postgres, redis · no gateway, hostnames unavailable");
    expect(screen.queryByText(productionWarning)).not.toBeInTheDocument();
    expect(screen.getByLabelText("Hostname")).toBeDisabled();
    expect(screen.getByText("qa_default: no gateway, hostnames unavailable.")).toBeInTheDocument();
    expect(screen.getByText("feature/security-test → no hostname on qa_default")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /save branches/i }));

    await vi.waitFor(() => expect(onDone).toHaveBeenCalled());
    expect(mocks.patch.mock.calls[0]?.[1].branch_deploy_rules[1]).toEqual({
      pattern: "feature/*",
      docker_network: "qa_default",
    });
  });

  it("refuses a second row with the same hostname pattern", async () => {
    const { user } = renderStep();

    await user.click(await screen.findByRole("button", { name: /add branch/i }));
    await user.click(screen.getByRole("button", { name: /add branch/i }));
    const [first, second] = screen.getAllByLabelText("Branch");
    await user.type(first!, "feature/*");
    await user.type(second!, "bugfix/*");
    await user.click(screen.getByRole("button", { name: /save branches/i }));

    expect(await screen.findByText("Another branch row already uses this hostname pattern")).toBeInTheDocument();
    expect(screen.getAllByRole("alert")).toHaveLength(1);
    expect(mocks.patch).not.toHaveBeenCalled();
  });

  it("saves no rule for the default branch once its switch is off", async () => {
    const { onDone, user } = renderStep();

    const toggle = await screen.findByRole("switch", { name: "Every push to main redeploys it" });
    expect(toggle).toBeChecked();
    await user.click(toggle);
    expect(screen.queryByText("main → app.example.com on web_default")).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /save branches/i }));

    await vi.waitFor(() => expect(onDone).toHaveBeenCalled());
    expect(mocks.patch).toHaveBeenCalledWith("/api/stacks/stack-1", expect.objectContaining({ branch_deploy_rules: [] }));
  });

  it("drops the production warning once an override replaces a value, and derives an exact row's own copy", async () => {
    const { onDone, user } = renderStep();

    await user.click(await screen.findByRole("button", { name: /add branch/i }));
    await user.type(screen.getByLabelText("Branch"), "staging");
    expect(screen.getByText(productionWarning)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /advanced options/i }));
    await user.type(screen.getByLabelText("Overrides"), "DATABASE_URL=postgres://staging/app");
    expect(screen.queryByText(productionWarning)).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /save branches/i }));

    await vi.waitFor(() => expect(onDone).toHaveBeenCalled());
    expect(mocks.patch.mock.calls[0]?.[1].branch_deploy_rules[1]).toEqual({
      pattern: "staging",
      docker_network: "web_default",
      name_suffix: "staging",
      hostname_template: "{branch}.example.com",
      port: 80,
      overrides: { DATABASE_URL: "postgres://staging/app" },
    });
  });

  it("refuses a wildcard hostname without a * for the branch", async () => {
    const { user } = renderStep();

    await user.click(await screen.findByRole("button", { name: /add branch/i }));
    await user.type(screen.getByLabelText("Branch"), "feature/*");
    await user.clear(screen.getByLabelText("Hostname"));
    await user.type(screen.getByLabelText("Hostname"), "preview.example.com");
    await user.click(screen.getByRole("button", { name: /save branches/i }));

    expect(await screen.findByText("Put * where the branch goes, like *.example.com")).toBeInTheDocument();
    expect(mocks.patch).not.toHaveBeenCalled();
  });

  it("skips without saving any rule", async () => {
    const { onSkip, user } = renderStep();

    await user.click(await screen.findByRole("button", { name: /skip for now/i }));
    expect(onSkip).toHaveBeenCalled();
    expect(mocks.patch).not.toHaveBeenCalled();
  });
});
