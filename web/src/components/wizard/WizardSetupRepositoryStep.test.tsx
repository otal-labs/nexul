import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { WizardSetupRepositoryStep } from "@/components/wizard/WizardSetupRepositoryStep";
import { useProjectWizardStore } from "@/stores/projectWizardStore";
import { useWorkspaceStore } from "@/stores/workspaceStore";

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), put: vi.fn(), link: { state: "connected" } as { state: string } }));

vi.mock("@/api/client", async () => {
  const { withOwner } = await import("@/test/viewer");
  return {
    api: {
      get: withOwner((url: string, ...rest: unknown[]) =>
        url === "/api/auth/github-link" ? Promise.resolve({ data: mocks.link }) : mocks.get(url, ...rest),
      ),
      post: mocks.post,
      put: mocks.put,
    },
    errorMessage: () => "",
  };
});
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const repos = [
  { id: 1, owner: "onik97", name: "api", full_name: "onik97/api", default_branch: "main", html_url: "", provider: "github" },
  { id: 2, owner: "onik97", name: "worker", full_name: "onik97/worker", default_branch: "main", html_url: "", provider: "github" },
];
const app = { owner: "onik97", name: "api", full_name: "onik97/api", connector_id: "github", role: "app" };
const tests = { owner: "onik97", name: "worker", full_name: "onik97/worker", connector_id: "github", role: "tests" };

const serve = (attached: object[] = []) =>
  mocks.get.mockImplementation((url: string) => {
    if (url === "/api/repositories") return Promise.resolve({ data: { repositories: repos } });
    if (url === "/api/projects/p-1") return Promise.resolve({ data: { id: "p-1", tests_location: "separate", setup: { finished: false, steps: {} } } });
    return Promise.resolve({ data: attached });
  });

const renderStep = () => {
  const onDone = vi.fn();
  const onLeaveEarly = vi.fn().mockResolvedValue(undefined);
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <WizardSetupRepositoryStep onDone={onDone} onBack={vi.fn()} onLeaveEarly={onLeaveEarly} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
  return { onDone, onLeaveEarly, user: userEvent.setup() };
};

beforeEach(() => {
  mocks.get.mockReset();
  mocks.post.mockReset();
  mocks.put.mockReset();
  mocks.link = { state: "connected" };
  mocks.post.mockResolvedValue({ data: undefined });
  mocks.put.mockResolvedValue({ data: {} });
  useProjectWizardStore.getState().reset();
  useProjectWizardStore.getState().setProjectId("p-1", "Backend");
  useWorkspaceStore.getState().selectWorkspace("ws-1", "acme");
});

describe("WizardSetupRepositoryStep", () => {
  it("holds the picked repository as a card without scanning it, and attaches it on Continue", async () => {
    serve();
    const { onDone, user } = renderStep();

    await user.type(await screen.findByLabelText("Search repositories"), "onik97");
    await user.click(await screen.findByText("onik97/api"));

    expect(screen.getByText("Attaches when you continue")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "No repository yet" })).not.toBeInTheDocument();
    expect(mocks.post).not.toHaveBeenCalled();
    await user.click(screen.getByRole("button", { name: "Continue" }));

    await vi.waitFor(() => expect(onDone).toHaveBeenCalled());
    expect(mocks.post).toHaveBeenCalledTimes(1);
    expect(mocks.post).toHaveBeenCalledWith("/api/projects/p-1/repos", { owner: "onik97", name: "api", connector_id: "github" });
    expect(useProjectWizardStore.getState().repository?.full_name).toBe("onik97/api");
  });

  it("saves the answer before leaving without a repository, which needs no GitHub link", async () => {
    mocks.link = { state: "none" };
    serve();
    const { onLeaveEarly, user } = renderStep();

    await user.click(await screen.findByRole("radio", { name: "The same repository" }));
    await user.click(await screen.findByRole("button", { name: "No repository yet" }));

    await vi.waitFor(() => expect(onLeaveEarly).toHaveBeenCalledWith("skipped"));
    expect(mocks.put).toHaveBeenCalledWith("/api/projects/p-1/tests-location", { tests_location: "same" });
  });

  describe("revisited after setup recorded it", () => {
    it("opens on the saved answer and the repositories already attached", async () => {
      serve([app, tests]);
      renderStep();

      await vi.waitFor(() => expect(screen.getByRole("radio", { name: "A separate one" })).toBeChecked());
      expect(await screen.findByText("onik97/worker")).toBeInTheDocument();
      expect(screen.getByText("onik97/api")).toBeInTheDocument();
      expect(screen.queryByLabelText("Search repositories")).not.toBeInTheDocument();
      expect(screen.queryByRole("button", { name: "No repository yet" })).not.toBeInTheDocument();
    });

    it("keeps the attached repository and attaches the tests repository picked, without scanning or attaching it again", async () => {
      serve([app]);
      const { onDone, user } = renderStep();

      await user.type(await screen.findByLabelText("Search tests repositories"), "onik97");
      await user.click(await screen.findByRole("button", { name: /onik97\/worker/ }));
      await user.click(screen.getByRole("button", { name: "Continue" }));

      await vi.waitFor(() => expect(onDone).toHaveBeenCalled());
      expect(mocks.post).toHaveBeenCalledTimes(1);
      expect(mocks.post).toHaveBeenCalledWith("/api/projects/p-1/repos", { owner: "onik97", name: "worker", connector_id: "github", role: "tests" });
    });

    it("searches for another repository only after Change, and Cancel puts the attached one back", async () => {
      serve([app]);
      const { user } = renderStep();

      await user.click(await screen.findByRole("button", { name: "Change" }));
      expect(screen.queryByRole("button", { name: "Continue" })).not.toBeInTheDocument();
      await user.type(await screen.findByLabelText("Search repositories"), "onik97");
      await screen.findByRole("button", { name: /onik97\/worker/ });
      await user.click(screen.getByRole("button", { name: "Cancel" }));

      expect(screen.queryByLabelText("Search repositories")).not.toBeInTheDocument();
      expect(screen.getByRole("button", { name: "Continue" })).toBeInTheDocument();
    });
  });
});
