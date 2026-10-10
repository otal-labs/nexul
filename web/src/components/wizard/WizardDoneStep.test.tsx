import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { WizardDoneStep } from "@/components/wizard/WizardDoneStep";
import { useProjectWizardStore } from "@/stores/projectWizardStore";

const access = vi.hoisted(() => ({ areas: ["stacks", "topology", "tickets"] as string[] }));
vi.mock("@/hooks/AccessHooks", () => ({ useAreaAccess: () => (area: string) => access.areas.includes(area) }));

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
  setup: { finished: false, steps: { project: "done", repository: "done", service: "done", reach: "skipped" } },
  created_at: "",
  updated_at: "",
};

const interview = [{ id: "m-1", project_id: "p-1", kind: "interview", body: "## Stack" }];

const mockApi = (memories: unknown[]) =>
  mocks.get.mockImplementation(async (url: string) => {
    if (url === "/api/projects") return { data: [project] };
    if (url === "/api/projects/p-1") return { data: project };
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
  useProjectWizardStore.getState().setName("api");
  useProjectWizardStore.getState().setMachine("box-1");
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
      expect.stringContaining("Reachskipped"),
      expect.stringContaining("Deploy branchesnot done"),
    ]);
    await user.click(left.getByRole("button", { name: "Open Reach" }));

    expect(await screen.findByText("reach step")).toBeInTheDocument();
  });

  it("offers only the destinations the viewer can read", async () => {
    access.areas = ["topology"];
    mockApi([{ id: "mem-i", project_id: "p-1", kind: "interview", body: "## Stack" }]);
    renderStep();
    expect(await screen.findByRole("button", { name: "View on the canvas" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "View stack" })).not.toBeInTheDocument();
  });
});
