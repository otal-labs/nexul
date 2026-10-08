import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { OwnerWizardPage } from "@/pages/OwnerWizardPage";
import type { ConnectorStatus } from "@/models/Connectors";
import { useOwnerWizardStore } from "@/stores/ownerWizardStore";

const mocks = vi.hoisted(() => ({
  useFetchSettings: vi.fn(),
  completeMutateAsync: vi.fn(),
  renameWorkspaceMutateAsync: vi.fn(),
  useFetchConnectors: vi.fn(),
}));

// ConnectorsSection.test.tsx already covers the full tab/card behavior; this file only needs enough to prove the real list shows up and the Finish setup handoff still works.
const connectorFixture: ConnectorStatus[] = [
  {
    connector: { id: "github", name: "GitHub", description: "Turns tasks into issues", category: "development", icon: "github" },
    status: { configured: false },
    available: true,
    app_configured: true,
  },
];

vi.mock("@/hooks/ConnectorsHooks", () => ({
  useFetchConnectors: mocks.useFetchConnectors,
  useStartConnectorOAuth: () => ({ mutate: vi.fn(), isPending: false }),
  useDisconnectConnector: () => ({ mutate: vi.fn(), isPending: false }),
}));

// Step components are exercised in full by their own test files; this file only cares about the stepper mechanics OwnerWizardPage itself owns: sequencing, back navigation, and the finish handoff to the app home.
vi.mock("@/components/auth/IntroduceYourselfStep", () => ({
  IntroduceYourselfStep: ({ onContinue }: { onContinue: () => void }) => (
    <button onClick={onContinue}>Step1 continue</button>
  ),
}));

const workspaceSetupFixture = { workspaceId: "workspace-default", workspaceName: "Acme" };

vi.mock("@/components/auth/SetupWorkspaceStep", () => ({
  SetupWorkspaceStep: ({ onContinue }: { onContinue: (data: typeof workspaceSetupFixture) => Promise<void> }) => (
    <button onClick={() => void onContinue(workspaceSetupFixture)}>Step2 continue</button>
  ),
}));

vi.mock("@/components/auth/SetupT3CodeStep", () => ({
  SetupT3CodeStep: ({ onFinish }: { onFinish: () => void }) => <button onClick={onFinish}>Finish setup</button>,
}));

vi.mock("@/hooks/AuthHooks", () => ({
  useFetchSettings: mocks.useFetchSettings,
  useFetchMe: () => ({ data: { user: {}, instance_permissions: [] } }),
  useCompleteOwnerWizard: () => ({ mutateAsync: mocks.completeMutateAsync, isPending: false }),
}));

vi.mock("@/hooks/WorkspaceHooks", () => ({
  useRenameWorkspace: () => ({ mutateAsync: mocks.renameWorkspaceMutateAsync }),
  useSelectedWorkspace: () => ({ id: "workspace-default", slug: "default" }),
}));

const renderPage = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={["/wizard/onboarding/owner"]}>
        <Routes>
          <Route path="/wizard/onboarding/owner" element={<OwnerWizardPage />} />
          <Route path="/login" element={<div>Login page</div>} />
          <Route path="/wizard/onboarding/dns" element={<div>DNS page</div>} />
          <Route path="/" element={<div>Home page</div>} />
          <Route path="/acme/wizard/project/project" element={<div>Project wizard</div>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

describe("OwnerWizardPage", () => {
  beforeEach(() => {
    // The store is a module singleton; reset it so progress from one test doesn't leak into the next.
    useOwnerWizardStore.getState().reset();
    mocks.useFetchSettings.mockReturnValue({
      data: { instance_url: "https://deploy.example.com", settings_version: 1, oauth_callback: "" },
      isPending: false,
      error: undefined,
    });
    mocks.completeMutateAsync.mockReset();
    mocks.completeMutateAsync.mockResolvedValue({});
    mocks.renameWorkspaceMutateAsync.mockReset();
    mocks.renameWorkspaceMutateAsync.mockResolvedValue({ slug: "acme" });
    mocks.useFetchConnectors.mockReturnValue({ data: connectorFixture, isPending: false, error: undefined });
  });

  it("walks through all four steps in sequence as each step's onContinue fires", async () => {
    const user = userEvent.setup();
    renderPage();

    expect(screen.getByText("Introduce yourself")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Step1 continue" }));

    expect(screen.getByText("Set up your workspace")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Step2 continue" }));

    expect(await screen.findByText("Connect your tools")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Continue" }));

    expect(screen.getByText("Set up T3 Code")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /finish setup/i })).toBeInTheDocument();
  });

  it("renders the real connectors list in step 3, not the old placeholder", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByRole("button", { name: "Step1 continue" }));
    await user.click(screen.getByRole("button", { name: "Step2 continue" }));

    expect(await screen.findByText("GitHub")).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: /not connected/i })).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Connectors" })).not.toBeInTheDocument();
    expect(screen.queryByText(/tool connections coming soon/i)).not.toBeInTheDocument();
  });

  it("returns to the previous step (not away from the page) on back from steps 2 and 3", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByRole("button", { name: "Step1 continue" }));
    expect(screen.getByText("Set up your workspace")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /back/i }));
    expect(screen.getByText("Introduce yourself")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Step1 continue" }));
    await user.click(screen.getByRole("button", { name: "Step2 continue" }));
    expect(await screen.findByText("Connect your tools")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /back/i }));
    expect(screen.getByText("Set up your workspace")).toBeInTheDocument();
  });

  it("navigates to /login on back from step 1, since there's nothing before it", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByRole("button", { name: /back/i }));
    expect(await screen.findByText("Login page")).toBeInTheDocument();
  });

  it("resumes at the saved step after a remount instead of restarting at step 1", async () => {
    const user = userEvent.setup();
    const first = renderPage();

    await user.click(screen.getByRole("button", { name: "Step1 continue" }));
    await user.click(screen.getByRole("button", { name: "Step2 continue" }));
    expect(await screen.findByText("Connect your tools")).toBeInTheDocument();

    // Step 3's Connect leaves the SPA via window.location.assign; coming back reloads everything, simulated here with a full unmount + fresh render.
    first.unmount();
    renderPage();

    expect(screen.getByText("Connect your tools")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Continue" }));
    await user.click(screen.getByRole("button", { name: /finish setup/i }));
    expect(await screen.findByText("Project wizard", {}, { timeout: 2000 })).toBeInTheDocument();
  });

  it("makes the caller owner and names the workspace when step 2 continues, so steps 3 and 4 run with owner permissions", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByRole("button", { name: "Step1 continue" }));
    await user.click(screen.getByRole("button", { name: "Step2 continue" }));

    expect(await screen.findByText("Connect your tools")).toBeInTheDocument();
    expect(mocks.completeMutateAsync).toHaveBeenCalledWith("https://deploy.example.com");
    // Applied only after CompleteOwnerWizard resolves, since the rename endpoint 403s until the caller is the default workspace's Owner.
    expect(mocks.renameWorkspaceMutateAsync).toHaveBeenCalledWith({ id: "workspace-default", name: "Acme", slug: "acme" });
  });

  it("stays on step 2 when making the owner fails, so it can be retried", async () => {
    mocks.completeMutateAsync.mockRejectedValue(new Error("boom"));
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByRole("button", { name: "Step1 continue" }));
    await user.click(screen.getByRole("button", { name: "Step2 continue" }));

    expect(screen.getByText("Set up your workspace")).toBeInTheDocument();
    expect(mocks.renameWorkspaceMutateAsync).not.toHaveBeenCalled();
  });

  it("finishes step 4 into the project wizard of the renamed workspace", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByRole("button", { name: "Step1 continue" }));
    await user.click(screen.getByRole("button", { name: "Step2 continue" }));
    await user.click(await screen.findByRole("button", { name: "Continue" }));
    await user.click(screen.getByRole("button", { name: /finish setup/i }));

    expect(await screen.findByText("Workspace ready")).toBeInTheDocument();
    expect(await screen.findByText("Project wizard", {}, { timeout: 2000 })).toBeInTheDocument();
    expect(useOwnerWizardStore.getState().step).toBe(1);
  });
});
