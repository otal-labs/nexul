import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import type { Memory } from "@/models/Memory";
import { InterviewPage } from "@/pages/InterviewPage";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn(),
}));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));
vi.mock("@/hooks/WorkspaceHooks", () => ({ useHasPermission: () => true }));
vi.mock("@/components/memory/MemoryDetail", () => ({
  MemoryDetail: ({ memory }: { memory: Memory }) => <div data-testid="memory-detail">{memory.title}</div>,
}));

const project = { id: "p-1", name: "Backend", prefix: "BE", position: 0, created_at: "", updated_at: "" };

const interview: Memory = {
  id: "mem-i",
  workspace_id: "ws-1",
  project_id: "p-1",
  kind: "interview",
  title: "Interview",
  when_to_use: "",
  body: "## Stack",
  always_included: true,
  version: 1,
  created_by: "u-1",
  created_at: "",
  updated_by: "u-1",
  updated_at: "",
};

const renderPage = (entry = "/projects/BE/interview") => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[entry]}>
        <Routes>
          <Route path="/projects/:projectId/interview" element={<InterviewPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

const mockMemories = (memories: Memory[]) =>
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/projects") return { data: [project] };
    if (url === "/api/memories") return { data: memories };
    return { data: [] };
  });

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  vi.mocked(api.post).mockReset();
});

describe("InterviewPage", () => {
  it("shows an error state", async () => {
    vi.mocked(api.get).mockRejectedValue(new Error("boom"));
    renderPage();
    expect(await screen.findByText("Something went wrong")).toBeInTheDocument();
  });

  it("says so when the project does not exist", async () => {
    mockMemories([]);
    renderPage("/projects/NOPE/interview");
    expect(await screen.findByText("Project not found")).toBeInTheDocument();
  });

  it("offers to start from the template when the project has no interview", async () => {
    const user = userEvent.setup();
    mockMemories([{ ...interview, id: "mem-other", kind: "", title: "Deploy quirks" }]);
    vi.mocked(api.post).mockResolvedValue({ data: interview });
    renderPage();

    await user.click(await screen.findByRole("button", { name: "Start from the template" }));
    expect(api.post).toHaveBeenCalledWith("/api/memories/interview", { project_id: "p-1" });
  });

  it("edits the existing interview", async () => {
    mockMemories([interview]);
    renderPage();
    expect(await screen.findByTestId("memory-detail")).toHaveTextContent("Interview");
    expect(screen.queryByText("No interview yet")).not.toBeInTheDocument();
  });
});
