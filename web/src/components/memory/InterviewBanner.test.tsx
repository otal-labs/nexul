import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { InterviewBanner } from "@/components/memory/InterviewBanner";
import { useInterviewBannerStore } from "@/stores/interviewBannerStore";
import type { Memory } from "@/models/Memory";
import type { Project } from "@/models/Project";

const mocks = vi.hoisted(() => ({ get: vi.fn() }));
vi.mock("@/api/client", () => ({ api: mocks, errorMessage: vi.fn() }));

const project: Project = { id: "p-1", name: "Backend", prefix: "BE", position: 0, icon: "", tests_location: "", created_at: "", updated_at: "" };

const memory = (kind: string): Memory => ({
  id: `mem-${kind}`,
  workspace_id: "ws-1",
  project_id: "p-1",
  kind,
  title: kind,
  when_to_use: "",
  body: "",
  always_included: kind === "interview",
  version: 1,
  created_by: "u-1",
  created_at: "",
  updated_by: "u-1",
  updated_at: "",
});

const renderBanner = () =>
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <InterviewBanner project={project} />
      </MemoryRouter>
    </QueryClientProvider>,
  );

beforeEach(() => {
  mocks.get.mockReset();
  useInterviewBannerStore.setState({ dismissedProjectIds: [] });
});

describe("InterviewBanner", () => {
  it("links to the Interview page while the project has no interview", async () => {
    mocks.get.mockResolvedValue({ data: [memory("")] });
    renderBanner();
    expect(await screen.findByText(/Backend has no interview yet/)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Run the interview" })).toHaveAttribute("href", "/projects/BE/interview");
  });

  it("stays away once the interview exists", async () => {
    mocks.get.mockResolvedValue({ data: [memory("interview")] });
    renderBanner();
    await vi.waitFor(() => expect(mocks.get).toHaveBeenCalled());
    expect(screen.queryByText(/has no interview yet/)).not.toBeInTheDocument();
  });

  it("dismisses for the session", async () => {
    const user = userEvent.setup();
    mocks.get.mockResolvedValue({ data: [] });
    renderBanner();
    await user.click(await screen.findByRole("button", { name: "Dismiss for this session" }));
    expect(screen.queryByText(/has no interview yet/)).not.toBeInTheDocument();
    expect(useInterviewBannerStore.getState().dismissedProjectIds).toEqual(["p-1"]);
  });
});
