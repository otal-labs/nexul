import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { WizardDoneStep } from "@/components/wizard/WizardDoneStep";
import { useProjectWizardStore } from "@/stores/projectWizardStore";

const access = vi.hoisted(() => ({ areas: ["stacks", "topology", "tickets"] as string[] }));
vi.mock("@/hooks/AccessHooks", () => ({ useAreaAccess: () => (area: string) => access.areas.includes(area), useVisibleSettingsSections: () => [] }));

beforeEach(() => {
  access.areas = ["stacks", "topology", "tickets"];
});

const mocks = vi.hoisted(() => ({ get: vi.fn(), put: vi.fn(), confirm: vi.fn() }));
vi.mock("@/api/client", () => ({ api: { get: mocks.get, put: mocks.put }, errorMessage: vi.fn() }));
vi.mock("@/hooks/useConfirmationDialog", () => ({ useConfirmationDialog: () => ({ open: mocks.confirm }) }));

const project = {
  id: "p-1",
  name: "Backend",
  prefix: "BE",
  position: 0,
  workspace_id: "ws-1",
  icon: "",
  tests_location: "",
  setup: { finished: false, stack_id: "s-1", env_keys: [], steps: { project: "done", people: "done", repository: "done", service: "done", reach: "skipped" } },
  created_at: "",
  updated_at: "",
};

const interview = [{ id: "m-1", project_id: "p-1", kind: "interview", body: "## Stack" }];

const mockApi = (memories: unknown[], recorded: object = project, repos: unknown[] = []) =>
  mocks.get.mockImplementation(async (url: string) => {
    if (url === "/api/projects") return { data: [recorded] };
    if (url === "/api/projects/p-1") return { data: recorded };
    if (url === "/api/projects/p-1/repos") return { data: repos };
    if (url === "/api/stacks/s-1") return { data: { id: "s-1", project_id: "p-1", name: "api", machine: "box-1" } };
    if (url === "/api/memories") return { data: memories };
    return { data: [] };
  });

const renderStep = () =>
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter initialEntries={["/acme/wizard/project/done"]}>
        <Routes>
          <Route path="/acme/wizard/project/done" element={<WizardDoneStep onBack={() => undefined} />} />
          <Route path="/acme/wizard/project/reach" element={<p>reach step</p>} />
          <Route path="/acme/wizard/project/deploy" element={<p>deploy step</p>} />
          <Route path="/acme/wizard/project/repository" element={<p>repository step</p>} />
          <Route path="/acme/board/:token" element={<p>project board</p>} />
          <Route path="/acme/projects/:projectId/interview" element={<p>interview page</p>} />
          <Route path="/acme/topology" element={<p>canvas page</p>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );

beforeEach(() => {
  mocks.get.mockReset();
  mocks.put.mockReset();
  mocks.confirm.mockReset();
  useProjectWizardStore.getState().reset();
  useProjectWizardStore.getState().setProjectId("p-1", "Backend");
  useProjectWizardStore.getState().setStackId("s-1");
});

describe("WizardDoneStep", () => {
  it("starts the interview on the project's Interview page", async () => {
    const user = userEvent.setup();
    mockApi([]);
    renderStep();
    await user.click(await screen.findByRole("button", { name: "Start the interview" }));
    expect(await screen.findByText("interview page")).toBeInTheDocument();
  });

  it("asks before skipping and says what is lost, then leaves a note", async () => {
    const user = userEvent.setup();
    mockApi([]);
    mocks.confirm.mockResolvedValue(true);
    renderStep();
    await user.click(await screen.findByRole("button", { name: "Skip" }));
    expect(mocks.confirm).toHaveBeenCalledWith(
      expect.objectContaining({ title: "Skip the interview?", message: expect.stringContaining("A banner stays on the project") }),
    );
    expect(await screen.findByText(/Interview skipped/)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Start the interview" })).not.toBeInTheDocument();
  });

  it("stays put when leaving is cancelled at the skip confirm", async () => {
    const user = userEvent.setup();
    mockApi([]);
    mocks.confirm.mockResolvedValue(false);
    renderStep();
    await screen.findByRole("button", { name: "Start the interview" });
    await user.click(screen.getByRole("button", { name: "View on the canvas" }));
    expect(mocks.confirm).toHaveBeenCalledOnce();
    expect(screen.queryByText("canvas page")).not.toBeInTheDocument();
  });

  it("offers nothing and asks nothing when the project already has an interview", async () => {
    const user = userEvent.setup();
    mockApi(interview);
    renderStep();
    await vi.waitFor(() => expect(mocks.get).toHaveBeenCalledWith("/api/memories", { params: { project_id: "p-1" } }));
    await user.click(screen.getByRole("button", { name: "View on the canvas" }));
    expect(await screen.findByText("canvas page")).toBeInTheDocument();
    expect(mocks.confirm).not.toHaveBeenCalled();
    expect(screen.queryByRole("button", { name: "Start the interview" })).not.toBeInTheDocument();
  });

  it("finishes setup with a step skipped and lands on the project's board", async () => {
    const user = userEvent.setup();
    mockApi(interview);
    mocks.put.mockResolvedValue({ data: { ...project, setup: { ...project.setup, finished: true } } });
    renderStep();

    await user.click(await screen.findByRole("button", { name: "Finish" }));

    expect(await screen.findByText("project board")).toBeInTheDocument();
    expect(mocks.put).toHaveBeenCalledWith("/api/projects/p-1/setup", { finished: true, steps: undefined });
  });

  it("lists each step left for later with the way back to it", async () => {
    const user = userEvent.setup();
    mockApi(interview);
    renderStep();

    const left = within(await screen.findByRole("region", { name: "Left for later" }));
    expect(left.getAllByRole("listitem").map((row) => row.textContent)).toEqual([
      expect.stringContaining("Domainskipped"),
      expect.stringContaining("Branchesnot done"),
    ]);
    await user.click(left.getByRole("button", { name: "Open Domain" }));

    expect(await screen.findByText("reach step")).toBeInTheDocument();
  });

  it("offers only the destinations the viewer can read", async () => {
    access.areas = ["topology"];
    mockApi([{ id: "mem-i", project_id: "p-1", kind: "interview", body: "## Stack" }]);
    renderStep();
    expect(await screen.findByRole("button", { name: "View on the canvas" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "View stack" })).not.toBeInTheDocument();
  });

  it("says the service is deploying when this run made it, and that it runs when an earlier run did", async () => {
    mockApi(interview);
    const { unmount } = renderStep();
    expect(await screen.findByText("api is deploying on box-1.")).toBeInTheDocument();
    unmount();

    useProjectWizardStore.setState({ stackId: null });
    renderStep();
    expect(await screen.findByText("api runs on box-1.")).toBeInTheDocument();
    expect(screen.queryByText(/is deploying on/)).not.toBeInTheDocument();
  });

  describe("without a service", () => {
    const unserved = { ...project, setup: { ...project.setup, stack_id: undefined, steps: { project: "done", people: "done", repository: "done" } } };
    const attached = [{ owner: "acme", name: "portal", full_name: "acme/portal", connector_id: "github", role: "app" }];

    beforeEach(() => {
      useProjectWizardStore.setState({ stackId: null });
    });

    it("folds the deploy steps into one Deploy row that opens Deploy when a repository is attached", async () => {
      const user = userEvent.setup();
      mockApi(interview, unserved, attached);
      renderStep();

      const left = within(await screen.findByRole("region", { name: "Left for later" }));
      expect(left.getAllByRole("listitem").map((row) => row.textContent)).toEqual([expect.stringContaining("Deploylater")]);
      await user.click(left.getByRole("button", { name: "Deploy now" }));

      expect(await screen.findByText("deploy step")).toBeInTheDocument();
    });

    it("offers no Deploy when the attached repository already deploys from a service made elsewhere", async () => {
      access.areas = ["stacks", "projects", "topology", "tickets"];
      mockApi(interview, unserved, attached);
      const built = mocks.get.getMockImplementation()!;
      let answerServices: (rows: unknown[]) => void = () => undefined;
      const services = new Promise<unknown[]>((resolve) => {
        answerServices = resolve;
      });
      mocks.get.mockImplementation(async (url: string, ...rest: unknown[]) =>
        url === "/api/services" ? { data: await services } : built(url, ...rest),
      );
      renderStep();

      expect(await screen.findByRole("region", { name: "Left for later" })).toBeInTheDocument();
      answerServices([{ id: "s-9", project_id: "p-1", name: "portal-staging", build_source: { repo_owner: "acme", repo_name: "portal" } }]);

      await vi.waitFor(() => expect(screen.queryByRole("region", { name: "Left for later" })).not.toBeInTheDocument());
      expect(screen.queryByRole("button", { name: "Deploy now" })).not.toBeInTheDocument();
      expect(screen.getByRole("button", { name: "Finish" })).toBeInTheDocument();
    });

    it("sends the Deploy row to Repository when none is attached", async () => {
      const user = userEvent.setup();
      mockApi(interview, unserved, []);
      renderStep();

      const left = within(await screen.findByRole("region", { name: "Left for later" }));
      expect(left.getByText("needs a repository")).toBeInTheDocument();
      await user.click(left.getByRole("button", { name: "Open Repository" }));

      expect(await screen.findByText("repository step")).toBeInTheDocument();
    });
  });
});
