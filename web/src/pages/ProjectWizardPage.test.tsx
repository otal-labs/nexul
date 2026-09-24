import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ProjectWizardPage } from "@/pages/ProjectWizardPage";
import { useProjectWizardStore } from "@/stores/projectWizardStore";

// Step components are exercised in full by their own test files (WizardRepositoryStep.test.tsx,
// WizardServiceStep.test.tsx, WizardEnvStep.test.tsx, WizardReachStep.test.tsx); this file only cares about the
// stepper mechanics ProjectWizardPage/ProjectWizardStepper own themselves: which rung the URL opens, an
// unknown step redirecting back to the start, and door 2's preselected-project rung.
const mocks = vi.hoisted(() => ({ get: vi.fn() }));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get },
  errorMessage: vi.fn(() => ""),
}));

const project = { id: "p-1", name: "Backend", prefix: "BE", position: 0, created_at: "", updated_at: "" };

const stack = {
  id: "stack-1",
  project_id: "p-1",
  name: "api",
  slug: "api",
  machine: "prod",
  strategy: "compose",
  managed: false,
  created_at: "",
  updated_at: "",
};

const renderPage = (path: string) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[path]}>
        <Routes>
          <Route path="/wizard/project/:step" element={<ProjectWizardPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

const rung = (title: RegExp) => screen.getByRole("heading", { name: title }).closest("li")!;

beforeEach(() => {
  mocks.get.mockReset();
  mocks.get.mockImplementation(async (url: string) => {
    if (url === "/api/projects/p-1") return { data: project };
    if (url === "/api/repositories") return { data: { repositories: [] } };
    return { data: [] };
  });
  useProjectWizardStore.getState().reset();
});

describe("ProjectWizardPage", () => {
  it("redirects an unknown step back to the project step", async () => {
    renderPage("/wizard/project/nonsense");
    expect(await screen.findByRole("heading", { name: /^project$/i })).toBeInTheDocument();
    expect(rung(/^project$/i)).toHaveAttribute("data-state", "active");
  });

  it("opens the project rung by default with everything after it upcoming", async () => {
    renderPage("/wizard/project/project");
    await screen.findByRole("heading", { name: /^project$/i });
    expect(rung(/^project$/i)).toHaveAttribute("data-state", "active");
    expect(rung(/^repository$/i)).toHaveAttribute("data-state", "upcoming");
    expect(rung(/^service$/i)).toHaveAttribute("data-state", "upcoming");
    expect(rung(/^reach$/i)).toHaveAttribute("data-state", "upcoming");
    expect(rung(/^deploy branches$/i)).toHaveAttribute("data-state", "upcoming");
    expect(rung(/^done$/i)).toHaveAttribute("data-state", "upcoming");
  });

  it("falls back to the furthest rung the store can render when a later step is opened cold", async () => {
    renderPage("/wizard/project/service");
    expect(await screen.findByRole("heading", { name: /^project$/i })).toBeInTheDocument();
    expect(rung(/^project$/i)).toHaveAttribute("data-state", "active");
    expect(rung(/^service$/i)).toHaveAttribute("data-state", "upcoming");
  });

  it("door 2: preselects the project from ?project= and opens the repository rung with the project already done", async () => {
    renderPage("/wizard/project/repository?project=p-1");
    expect(await screen.findByRole("heading", { name: /^repository$/i })).toBeInTheDocument();

    const projectRung = rung(/^project$/i);
    expect(projectRung).toHaveAttribute("data-state", "done");
    expect(await within(projectRung).findByText("Backend")).toBeInTheDocument();
    // Locked: door 2 seeded it, so there's no Change affordance to re-open it.
    expect(within(projectRung).queryByRole("button", { name: /change/i })).not.toBeInTheDocument();
    expect(rung(/^repository$/i)).toHaveAttribute("data-state", "active");
  });

  it("door 3: preselects the project from ?stack= and opens the repository rung with the project already done", async () => {
    mocks.get.mockImplementation(async (url: string) => {
      if (url === "/api/stacks/stack-1") return { data: stack };
      if (url === "/api/projects/p-1") return { data: project };
      if (url === "/api/repositories") return { data: { repositories: [] } };
      return { data: [] };
    });
    renderPage("/wizard/project/repository?stack=stack-1");
    expect(await screen.findByRole("heading", { name: /^repository$/i })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: /^attach a repository$/i })).toBeInTheDocument();

    const projectRung = rung(/^project$/i);
    expect(projectRung).toHaveAttribute("data-state", "done");
    expect(await within(projectRung).findByText("Backend")).toBeInTheDocument();
    // Locked: door 3 seeded it too, so there's no Change affordance to re-open it.
    expect(within(projectRung).queryByRole("button", { name: /change/i })).not.toBeInTheDocument();
    expect(rung(/^repository$/i)).toHaveAttribute("data-state", "active");
  });
});
