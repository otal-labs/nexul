import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { WizardDoneStep } from "@/components/wizard/WizardDoneStep";
import { useProjectWizardStore } from "@/stores/projectWizardStore";

const mocks = vi.hoisted(() => ({ get: vi.fn(), confirm: vi.fn() }));
vi.mock("@/api/client", () => ({ api: { get: mocks.get }, errorMessage: vi.fn() }));
vi.mock("@/hooks/useConfirmationDialog", () => ({ useConfirmationDialog: () => ({ open: mocks.confirm }) }));

const project = { id: "p-1", name: "Backend", prefix: "BE", position: 0, icon: "", tests_location: "", created_at: "", updated_at: "" };

const mockApi = (memories: unknown[]) =>
  mocks.get.mockImplementation(async (url: string) => {
    if (url === "/api/projects") return { data: [project] };
    if (url === "/api/memories") return { data: memories };
    return { data: [] };
  });

const renderStep = () =>
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter initialEntries={["/wizard/project/done"]}>
        <Routes>
          <Route path="/wizard/project/done" element={<WizardDoneStep />} />
          <Route path="/projects/:projectId/interview" element={<p>interview page</p>} />
          <Route path="/topology" element={<p>canvas page</p>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );

beforeEach(() => {
  mocks.get.mockReset();
  mocks.confirm.mockReset();
  useProjectWizardStore.getState().reset();
  useProjectWizardStore.getState().setProjectId("p-1", "Backend");
  useProjectWizardStore.getState().setName("api");
  useProjectWizardStore.getState().setMachine("box-1");
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
    mockApi([{ id: "m-1", project_id: "p-1", kind: "interview" }]);
    renderStep();
    await vi.waitFor(() => expect(mocks.get).toHaveBeenCalledWith("/api/memories", { params: { project_id: "p-1" } }));
    await user.click(screen.getByRole("button", { name: "View on the canvas" }));
    expect(await screen.findByText("canvas page")).toBeInTheDocument();
    expect(mocks.confirm).not.toHaveBeenCalled();
    expect(screen.queryByRole("button", { name: "Start the interview" })).not.toBeInTheDocument();
  });
});
