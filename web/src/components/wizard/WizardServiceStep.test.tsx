import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { WizardServiceStep } from "@/components/wizard/WizardServiceStep";
import { useProjectWizardStore } from "@/stores/projectWizardStore";
import { pickOption } from "@/test/pickOption";

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), patch: vi.fn(), errorMessage: vi.fn(() => "") }));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, post: mocks.post, patch: mocks.patch },
  errorMessage: mocks.errorMessage,
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const machines = [{ id: "m-1", name: "prod", stack_root: "/data/nexul", first_seen: "", last_seen: "" }];

const repository = { id: 1, owner: "onik97", name: "api", full_name: "onik97/api", default_branch: "main", html_url: "" };

const attachStack = {
  id: "stack-9",
  project_id: "p-1",
  name: "api",
  slug: "api",
  machine: "prod",
  strategy: "compose",
  managed: false,
  created_at: "",
  updated_at: "",
};

const renderStep = (onDone = vi.fn()) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <WizardServiceStep onDone={onDone} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
  return onDone;
};

beforeEach(() => {
  mocks.get.mockReset();
  mocks.post.mockReset();
  mocks.patch.mockReset();
  mocks.get.mockResolvedValue({ data: machines });
  useProjectWizardStore.getState().reset();
});

describe("WizardServiceStep", () => {
  it("creates a compose stack with its declared services and starts the first deploy in one call", async () => {
    useProjectWizardStore.getState().setProjectId("p-1", "API");
    useProjectWizardStore.getState().setRepository(repository);
    useProjectWizardStore.getState().setScanResult({ default_branch: "main", candidates: [], env_keys: [] });
    useProjectWizardStore.getState().setCandidate({
      kind: "compose",
      path: "docker-compose.yml",
      name: "api",
      services: [{ name: "web", image: "nginx", ports: [80], expose: [], env_keys: ["API_KEY"] }],
      reachable: { service: "web", port: 80 },
    });
    mocks.post.mockResolvedValue({
      data: { id: "stack-1", name: "api", slug: "api", machine: "prod", strategy: "compose", managed: true, deploy: { id: "d1", stack_id: "stack-1", status: "pending", kind: "deploy" } },
    });
    const onDone = renderStep();
    const user = userEvent.setup();

    expect(await screen.findByText(/reachable: web:80/i)).toBeInTheDocument();
    await pickOption(user, "Machine", "prod");
    await user.click(screen.getByRole("button", { name: /create & deploy/i }));

    await vi.waitFor(() => expect(onDone).toHaveBeenCalled());
    expect(mocks.post).toHaveBeenCalledWith(
      "/api/stacks",
      expect.objectContaining({
        project_id: "p-1",
        name: "api",
        machine: "prod",
        strategy: "compose",
        compose_path: "docker-compose.yml",
        link_repository: true,
        deploy: true,
        declared: { web: { image: "nginx", ports: ["80"], env_keys: ["API_KEY"] } },
        build_source: expect.objectContaining({ repo_owner: "onik97", repo_name: "api", branch: "main" }),
      }),
    );
    expect(useProjectWizardStore.getState().stackId).toBe("stack-1");
  });

  it("holds the first deploy back when an env step follows", async () => {
    useProjectWizardStore.getState().setProjectId("p-1", "API");
    useProjectWizardStore.getState().setRepository(repository);
    useProjectWizardStore.getState().setScanResult({ default_branch: "main", candidates: [], env_keys: ["API_KEY"] });
    useProjectWizardStore.getState().setCandidate({ kind: "compose", path: "docker-compose.yml", name: "api", services: [] });
    mocks.post.mockResolvedValue({ data: { id: "stack-3", name: "api", slug: "api", machine: "prod", strategy: "compose", managed: true } });
    const onDone = renderStep();
    const user = userEvent.setup();

    await pickOption(user, "Machine", "prod");
    expect(screen.queryByRole("button", { name: /create & deploy/i })).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /^create$/i }));

    await vi.waitFor(() => expect(onDone).toHaveBeenCalled());
    expect(mocks.post).toHaveBeenCalledWith("/api/stacks", expect.objectContaining({ deploy: false }));
  });

  it("keys a Dockerfile candidate's declared entry by the stack's own slug", async () => {
    useProjectWizardStore.getState().setProjectId("p-1", "API");
    useProjectWizardStore.getState().setCandidate({ kind: "dockerfile", path: "Dockerfile", name: "worker", services: [] });
    mocks.post.mockResolvedValue({ data: { id: "stack-2", name: "worker", slug: "worker", machine: "prod", strategy: "run", managed: true } });
    renderStep();
    const user = userEvent.setup();

    await pickOption(user, "Machine", "prod");
    await user.click(screen.getByRole("button", { name: /create & deploy/i }));

    await vi.waitFor(() =>
      expect(mocks.post).toHaveBeenCalledWith(
        "/api/stacks",
        expect.objectContaining({ strategy: "run", declared: { worker: {} }, docker_network: "worker_default" }),
      ),
    );
  });

  it("attach mode: PATCHes the existing stack with the build source and starts the deploy when no env keys", async () => {
    useProjectWizardStore.getState().setProjectId("p-1", "API");
    useProjectWizardStore.getState().setAttachStackId("stack-9");
    useProjectWizardStore.getState().setRepository(repository);
    useProjectWizardStore.getState().setScanResult({ default_branch: "main", candidates: [], env_keys: [] });
    useProjectWizardStore.getState().setCandidate({ kind: "compose", path: "docker-compose.yml", name: "api", services: [] });
    mocks.get.mockImplementation(async (url: string) => {
      if (url === "/api/stacks/stack-9") return { data: attachStack };
      return { data: [] };
    });
    mocks.patch.mockResolvedValue({ data: { ...attachStack, build_source: { repo_owner: "onik97", repo_name: "api", branch: "main", compose_path: "docker-compose.yml" }, compose_path: "docker-compose.yml" } });
    mocks.post.mockResolvedValue({ data: { id: "d-1" } });
    const onDone = renderStep();
    const user = userEvent.setup();

    const attachButton = await screen.findByRole("button", { name: /^attach$/i });
    expect(screen.getByText("api", { selector: "span.font-medium" })).toBeInTheDocument();
    expect(screen.getByText("prod", { selector: "span.font-mono" })).toBeInTheDocument();
    await user.click(attachButton);

    await vi.waitFor(() => expect(onDone).toHaveBeenCalled());
    expect(mocks.patch).toHaveBeenCalledWith(
      "/api/stacks/stack-9",
      expect.objectContaining({
        id: "stack-9",
        strategy: "compose",
        compose_path: "docker-compose.yml",
        build_source: expect.objectContaining({ repo_owner: "onik97", repo_name: "api", branch: "main" }),
      }),
    );
    expect(mocks.post).toHaveBeenCalledWith("/api/deploys", expect.objectContaining({ stack_id: "stack-9", ref: "main" }));
    expect(useProjectWizardStore.getState().stackId).toBe("stack-9");
    // The done rung reads name and machine from the store, so attach seeds them from the stack it patched.
    expect(useProjectWizardStore.getState().name).toBe("api");
    expect(useProjectWizardStore.getState().machine).toBe("prod");
  });

  it("attach mode: a Dockerfile candidate turns the adopted compose stack into a run stack with a network", async () => {
    useProjectWizardStore.getState().setProjectId("p-1", "API");
    useProjectWizardStore.getState().setAttachStackId("stack-9");
    useProjectWizardStore.getState().setRepository(repository);
    useProjectWizardStore.getState().setScanResult({ default_branch: "main", candidates: [], env_keys: [] });
    useProjectWizardStore.getState().setCandidate({ kind: "dockerfile", path: "Dockerfile", name: "api", services: [] });
    mocks.get.mockImplementation(async (url: string) => {
      if (url === "/api/stacks/stack-9") return { data: attachStack };
      return { data: [] };
    });
    mocks.patch.mockResolvedValue({ data: attachStack });
    mocks.post.mockResolvedValue({ data: { id: "d-1" } });
    const onDone = renderStep();
    const user = userEvent.setup();

    await user.click(await screen.findByRole("button", { name: /^attach$/i }));

    await vi.waitFor(() => expect(onDone).toHaveBeenCalled());
    expect(mocks.patch).toHaveBeenCalledWith(
      "/api/stacks/stack-9",
      expect.objectContaining({
        strategy: "run",
        docker_network: "api_default",
        build_source: expect.objectContaining({ dockerfile: "Dockerfile" }),
      }),
    );
  });

  it("attach mode: holds the deploy back when an env step follows", async () => {
    useProjectWizardStore.getState().setProjectId("p-1", "API");
    useProjectWizardStore.getState().setAttachStackId("stack-9");
    useProjectWizardStore.getState().setRepository(repository);
    useProjectWizardStore.getState().setScanResult({ default_branch: "main", candidates: [], env_keys: ["API_KEY"] });
    useProjectWizardStore.getState().setCandidate({ kind: "compose", path: "docker-compose.yml", name: "api", services: [] });
    mocks.get.mockImplementation(async (url: string) => {
      if (url === "/api/stacks/stack-9") return { data: attachStack };
      return { data: [] };
    });
    mocks.patch.mockResolvedValue({ data: attachStack });
    const onDone = renderStep();
    const user = userEvent.setup();

    const attachButton = await screen.findByRole("button", { name: /^attach$/i });
    await user.click(attachButton);

    await vi.waitFor(() => expect(onDone).toHaveBeenCalled());
    expect(mocks.patch).toHaveBeenCalled();
    expect(mocks.post).not.toHaveBeenCalledWith("/api/deploys", expect.anything());
  });
});
