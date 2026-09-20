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
  renameMutateAsync: vi.fn(),
  renameWorkspaceMutateAsync: vi.fn(),
  setPrefixMutateAsync: vi.fn(),
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

// Step components are exercised in full by their own test files; this file only cares about the stepper mechanics OwnerWizardPage itself owns: sequencing, back navigation, and the finish handoff to /wizard/onboarding/dns.
vi.mock("@/components/auth/IntroduceYourselfStep", () => ({
  IntroduceYourselfStep: ({ onContinue }: { onContinue: () => void }) => (
    <button onClick={onContinue}>Step1 continue</button>
  ),
}));

const workspaceSetupFixture = {
  workspaceId: "workspace-default",
  workspaceName: "Acme",
  projectId: "p-1",
  projectName: "General",
  projectPrefix: "GEN",
};

vi.mock("@/components/auth/SetupWorkspaceStep", () => ({
  SetupWorkspaceStep: ({ onContinue }: { onContinue: (data: typeof workspaceSetupFixture) => void }) => (
    <button onClick={() => onContinue(workspaceSetupFixture)}>Step2 continue</button>
  ),
}));

vi.mock("@/hooks/AuthHooks", () => ({
  useFetchSettings: mocks.useFetchSettings,
  useFetchMe: () => ({ data: { user: { can_create_workspace: true } } }),
  useCompleteOwnerWizard: () => ({ mutateAsync: mocks.completeMutateAsync, isPending: false }),
}));

vi.mock("@/hooks/ProjectHooks", () => ({
  useRenameProject: () => ({ mutateAsync: mocks.renameMutateAsync }),
  useSetProjectPrefix: () => ({ mutateAsync: mocks.setPrefixMutateAsync }),
}));

vi.mock("@/hooks/WorkspaceHooks", () => ({
  useRenameWorkspace: () => ({ mutateAsync: mocks.renameWorkspaceMutateAsync }),
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
    mocks.renameMutateAsync.mockReset();
    mocks.renameMutateAsync.mockResolvedValue({});
    mocks.renameWorkspaceMutateAsync.mockReset();
    mocks.renameWorkspaceMutateAsync.mockResolvedValue({});
    mocks.setPrefixMutateAsync.mockReset();
    mocks.setPrefixMutateAsync.mockResolvedValue({});
    mocks.useFetchConnectors.mockReturnValue({ data: connectorFixture, isPending: false, error: undefined });
  });

  it("walks through all three steps in sequence as each step's onContinue fires", async () => {
    const user = userEvent.setup();
    renderPage();

    expect(screen.getByText("Introduce yourself")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Step1 continue" }));

    expect(screen.getByText("Set up your workspace")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Step2 continue" }));

    expect(screen.getByText("Connect your tools")).toBeInTheDocument();
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
    expect(screen.getByText("Connect your tools")).toBeInTheDocument();

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
    expect(screen.getByText("Connect your tools")).toBeInTheDocument();

    // Step 3's Connect leaves the SPA via window.location.assign; coming back reloads everything, simulated here with a full unmount + fresh render.
    first.unmount();
    renderPage();

    expect(screen.getByText("Connect your tools")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /finish setup/i }));
    expect(mocks.setPrefixMutateAsync).toHaveBeenCalledWith({ id: "p-1", prefix: "GEN" });
  });

  it("finishes step 3 using the instance URL from settings (not empty), then navigates to DNS onboarding", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByRole("button", { name: "Step1 continue" }));
    await user.click(screen.getByRole("button", { name: "Step2 continue" }));
    await user.click(screen.getByRole("button", { name: /finish setup/i }));

    expect(mocks.completeMutateAsync).toHaveBeenCalledWith("https://deploy.example.com");
    // Applied only after CompleteOwnerWizard resolves, since step 2's rename/prefix endpoints 403 until can_create_workspace is granted (requireOwner).
    expect(mocks.renameWorkspaceMutateAsync).toHaveBeenCalledWith({ id: "workspace-default", name: "Acme" });
    expect(mocks.renameMutateAsync).toHaveBeenCalledWith({ id: "p-1", name: "General" });
    expect(mocks.setPrefixMutateAsync).toHaveBeenCalledWith({ id: "p-1", prefix: "GEN" });
    expect(await screen.findByText("Workspace ready")).toBeInTheDocument();
    expect(await screen.findByText("DNS page", {}, { timeout: 2000 })).toBeInTheDocument();
  });

  it("disables Finish setup for the whole sequence — a double click runs it once", async () => {
    // Keep the first mutation in flight so the second click lands mid-sequence.
    let release: (value: unknown) => void = () => {};
    mocks.completeMutateAsync.mockImplementation(() => new Promise((r) => (release = r)));
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByRole("button", { name: "Step1 continue" }));
    await user.click(screen.getByRole("button", { name: "Step2 continue" }));

    const finish = screen.getByRole("button", { name: /finish setup/i });
    await user.click(finish);
    expect(screen.getByRole("button", { name: /finishing/i })).toBeDisabled();
    await user.click(screen.getByRole("button", { name: /finishing/i })).catch(() => {});
    release({});

    expect(await screen.findByText("Workspace ready")).toBeInTheDocument();
    expect(mocks.completeMutateAsync).toHaveBeenCalledTimes(1);
    expect(mocks.renameWorkspaceMutateAsync).toHaveBeenCalledTimes(1);
  });
});
