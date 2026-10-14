import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import type { ReactElement } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { getMyRoleKey } from "@/hooks/WorkspaceHooks";
import { BranchRulesForm } from "@/components/wizard/BranchRulesForm";
import { WizardEnvStep } from "@/components/wizard/WizardEnvStep";
import { WizardPeopleStep } from "@/components/wizard/WizardPeopleStep";
import { WizardProjectStep } from "@/components/wizard/WizardProjectStep";
import { WizardRepositoryStep } from "@/components/wizard/WizardRepositoryStep";
import { WizardServiceStep } from "@/components/wizard/WizardServiceStep";
import type { Project } from "@/models/Project";
import { useProjectWizardStore } from "@/stores/projectWizardStore";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { asViewer, memberWho } from "@/test/viewer";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn() },
  errorMessage: vi.fn(() => ""),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const stack = {
  id: "stack-1",
  project_id: "p-1",
  name: "api",
  slug: "api",
  machine: "prod",
  strategy: "compose" as const,
  managed: true,
  build_source: { repo_owner: "acme", repo_name: "api", branch: "main" },
  created_at: "",
  updated_at: "",
};

const project = { id: "p-1", name: "Storefront", workspace_id: "ws-1" } as Project;
const role = { id: "r-dev", workspace_id: "ws-1", name: "Developer", permissions: [], is_owner_role: false, created_at: "", updated_at: "" };
const repository = { id: 1, owner: "acme", name: "api", full_name: "acme/api", default_branch: "main", html_url: "", provider: "github" };

const data = (url: string): unknown => {
  if (url.startsWith("/api/permissions?")) return { grants: [] };
  if (url === "/api/workspaces/ws-1/roles") return [role];
  if (url === "/api/workspaces/ws-1/people") return { people: [] };
  if (url === "/api/machines") return [{ id: "m-1", name: "prod", stack_root: "/data", first_seen: "", last_seen: "" }];
  if (url === "/api/stacks/stack-1") return stack;
  if (url === "/api/auth/github-link") return { state: "connected" };
  if (url === "/api/dns/exposures" || url === "/api/stacks/stack-1/services") return [];
  return [];
};

const seedService = (envKeys: string[]) => {
  const store = useProjectWizardStore.getState();
  store.setProjectId("p-1", "Storefront");
  store.setRepository(repository);
  store.setScanResult({ default_branch: "main", candidates: [], env_keys: envKeys });
  store.setCandidate({ kind: "compose", path: "docker-compose.yml", name: "api", services: [] });
};

interface Row {
  name: string;
  // What the viewer holds, a step short of what the control writes.
  holds: string[];
  step: () => ReactElement;
  control: { role: "button" | "checkbox"; name: string };
  reason: string;
}

const rows: Row[] = [
  {
    name: "Info creates the project",
    holds: ["projects:write"],
    step: () => <WizardProjectStep onDone={vi.fn()} />,
    control: { role: "button", name: "Continue" },
    reason: "Your role can't create projects",
  },
  {
    name: "Who's in it ticks a role into the project",
    holds: ["projects:read"],
    step: () => <WizardPeopleStep project={project} continueLabel="Continue" onDone={vi.fn()} onBack={vi.fn()} onSkip={vi.fn()} />,
    control: { role: "checkbox", name: "Developer" },
    reason: "Your role can't choose who is in a project",
  },
  {
    name: "Service creates the stack",
    holds: ["projects:write", "deploys:write"],
    step: () => {
      seedService(["API_KEY"]);
      return <WizardServiceStep onDone={vi.fn()} />;
    },
    control: { role: "button", name: "Create" },
    reason: "Your role can't create and update stacks",
  },
  {
    name: "Service creates the stack and starts its first deploy",
    holds: ["projects:write", "stacks:write"],
    step: () => {
      seedService([]);
      return <WizardServiceStep onDone={vi.fn()} />;
    },
    control: { role: "button", name: "Create & deploy" },
    reason: "Your role can't create and update deploys",
  },
  {
    name: "Environment saves the values and deploys",
    holds: ["stacks:write"],
    step: () => <WizardEnvStep stackId="stack-1" envKeys={["API_KEY"]} onDone={vi.fn()} />,
    control: { role: "button", name: "Save & deploy" },
    reason: "Your role can't create and update deploys",
  },
  {
    name: "Deploy branches saves the rules",
    holds: ["deploys:write"],
    step: () => <BranchRulesForm stack={stack} exposure={undefined} defaultPort="" networks={[]} onDone={vi.fn()} onSkip={vi.fn()} />,
    control: { role: "button", name: "Save branches" },
    reason: "Your role can't create and update stacks",
  },
];

const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });

const renderStep = (step: Row["step"]) =>
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter>{step()}</MemoryRouter>
    </QueryClientProvider>,
  );

// Hidden is also how a control looks before the viewer's permissions arrive, so wait for them before asserting it is gone.
const viewerKnown = () => vi.waitFor(() => expect(client.getQueryState([getMyRoleKey, "ws-1"])?.status).toBe("success"));

const serve = (holds: string[], showLocked: boolean) => vi.mocked(api.get).mockImplementation(asViewer(memberWho(holds, showLocked), data));

describe("a wizard step the viewer's role stops short of", () => {
  beforeEach(() => {
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
    useProjectWizardStore.getState().reset();
    vi.mocked(api.get).mockReset();
    client.clear();
  });

  it.each(rows)("$name: locked with the reason when a role shows locked actions", async ({ holds, step, control, reason }) => {
    serve(holds, true);
    renderStep(step);

    const locked = await screen.findByRole(control.role, { name: control.name });
    await vi.waitFor(() => expect(locked).toHaveAttribute("aria-disabled", "true"));
    if (control.role === "button") expect(locked).toHaveAccessibleDescription(reason);
    if (control.role === "checkbox") expect(screen.getByText(reason)).toBeInTheDocument();
  });

  it.each(rows)("$name: shows no way to write, and no lock, otherwise", async ({ holds, step, control, reason }) => {
    serve(holds, false);
    renderStep(step);

    await viewerKnown();
    if (control.role === "button") await vi.waitFor(() => expect(screen.queryByRole("button", { name: control.name })).not.toBeInTheDocument());
    if (control.role === "checkbox") expect(await screen.findByRole("checkbox", { name: control.name })).toHaveAttribute("aria-disabled", "true");
    expect(screen.queryByText(reason)).not.toBeInTheDocument();
  });

  it("Repository offers no search, and says why, to a role that can't scan", async () => {
    serve(["projects:write"], true);
    useProjectWizardStore.getState().setProjectId("p-1", "Storefront");
    renderStep(() => <WizardRepositoryStep onDone={vi.fn()} />);

    expect(await screen.findByText("Your role can't scan repositories")).toBeInTheDocument();
    expect(screen.queryByLabelText("Search repositories")).not.toBeInTheDocument();
  });
});
