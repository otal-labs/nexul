import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { RouterProvider, createMemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ProjectWizardPage } from "@/pages/ProjectWizardPage";
import { useProjectWizardStore } from "@/stores/projectWizardStore";

const access = vi.hoisted(() => ({ areas: ["tickets"] as string[] }));
vi.mock("@/hooks/AccessHooks", () => ({ useAreaAccess: () => (area: string) => access.areas.includes(area) }));

beforeEach(() => {
  access.areas = ["tickets"];
});

// Step components are exercised in full by their own test files (WizardRepositoryStep.test.tsx,
// WizardServiceStep.test.tsx, WizardEnvStep.test.tsx, WizardReachStep.test.tsx); this file only cares about the
// stepper mechanics ProjectWizardPage/ProjectWizardStepper own themselves: which rung the URL opens, an
// unknown step redirecting back to the start, and door 2's preselected-project rung.
const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn() }));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, post: mocks.post },
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

const renderPage = (path: string | string[]) => {
  const entries = Array.isArray(path) ? path : [path];
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <RouterProvider
        router={createMemoryRouter(
          [
            { path: "/wizard/project/:step", element: <ProjectWizardPage /> },
            { path: "/", element: <p>home</p> },
            { path: "/board", element: <p>board</p> },
            { path: "/board/:token", element: <p>project board</p> },
          ],
          { initialEntries: entries, initialIndex: entries.length - 1 },
        )}
      />
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
  it("offers no way to skip the info rung, since the project does not exist until it is created", async () => {
    renderPage("/wizard/project/project");

    await screen.findByLabelText("Project name");

    expect(within(rung(/: info$/i)).queryByRole("button", { name: "Skip for now" })).not.toBeInTheDocument();
  });

  it("goes back from the info rung to the page the wizard was opened from, creating nothing", async () => {
    const user = userEvent.setup();
    renderPage(["/board/p-1", "/wizard/project/project"]);

    await user.click(await screen.findByRole("button", { name: "Back" }));

    expect(await screen.findByText("project board")).toBeInTheDocument();
    expect(mocks.post).not.toHaveBeenCalled();
  });

  it("goes back from a cold-opened info rung to the board", async () => {
    const user = userEvent.setup();
    renderPage("/wizard/project/project");

    await user.click(await screen.findByRole("button", { name: "Back" }));

    expect(await screen.findByText("board")).toBeInTheDocument();
  });

  it("goes back from the service rung to the repository rung while no stack exists yet", async () => {
    const store = useProjectWizardStore.getState();
    store.setProjectId("p-1", "Backend");
    store.setCandidate({ kind: "compose", path: "compose.yml", name: "api", services: [] });
    const user = userEvent.setup();
    renderPage("/wizard/project/service");
    await screen.findByRole("heading", { name: /: service$/i });

    await user.click(screen.getByRole("button", { name: "Back" }));

    expect(rung(/: repository$/i)).toHaveAttribute("data-state", "active");
    expect(screen.queryByRole("button", { name: "Back" })).not.toBeInTheDocument();
  });

  it("offers no way back from the service rung once its stack exists", async () => {
    const store = useProjectWizardStore.getState();
    store.setProjectId("p-1", "Backend");
    store.setCandidate({ kind: "compose", path: "compose.yml", name: "api", services: [] });
    store.setStackId("stack-1");
    renderPage("/wizard/project/service");
    await screen.findByRole("heading", { name: /: service$/i });

    expect(screen.queryByRole("button", { name: "Back" })).not.toBeInTheDocument();
  });

  it("keeps the project it just created when the rest is skipped, and lands on its board", async () => {
    mocks.post.mockResolvedValueOnce({ data: project });
    const user = userEvent.setup();
    renderPage("/wizard/project/project");

    await user.type(await screen.findByLabelText("Project name"), "Backend");
    await user.type(screen.getByLabelText("Prefix"), "BE");
    await user.click(screen.getByRole("button", { name: "Continue" }));
    await screen.findByRole("heading", { name: /: repository$/i });
    expect(within(rung(/: info$/i)).queryByRole("button", { name: /change/i })).not.toBeInTheDocument();
    await user.click(await within(rung(/: repository$/i)).findByRole("button", { name: "Skip for now" }));

    expect(await screen.findByText("project board")).toBeInTheDocument();
    expect(mocks.post).toHaveBeenCalledTimes(1);
    expect(useProjectWizardStore.getState().projectId).toBeNull();
  });

  it("redirects an unknown step back to the project step", async () => {
    renderPage("/wizard/project/nonsense");
    expect(await screen.findByRole("heading", { name: /: info$/i })).toBeInTheDocument();
    expect(rung(/: info$/i)).toHaveAttribute("data-state", "active");
  });

  it("opens the project rung by default with everything after it upcoming", async () => {
    renderPage("/wizard/project/project");
    await screen.findByRole("heading", { name: /: info$/i });
    expect(rung(/: info$/i)).toHaveAttribute("data-state", "active");
    expect(rung(/: repository$/i)).toHaveAttribute("data-state", "upcoming");
    expect(rung(/: service$/i)).toHaveAttribute("data-state", "upcoming");
    expect(rung(/: reach$/i)).toHaveAttribute("data-state", "upcoming");
    expect(rung(/: deploy branches$/i)).toHaveAttribute("data-state", "upcoming");
    expect(rung(/: done$/i)).toHaveAttribute("data-state", "upcoming");
    expect(screen.getByRole("heading", { name: "Step 1: Info" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Step 6: Done" })).toBeInTheDocument();
  });

  it("frames the project rung as the first project when the workspace has none", async () => {
    renderPage("/wizard/project/project");
    expect(await screen.findByRole("heading", { name: /^create your first project$/i })).toBeInTheDocument();
  });

  it("titles the project rung New project once the workspace has one", async () => {
    mocks.get.mockImplementation(async (url: string) => {
      if (url === "/api/projects") return { data: [project] };
      return { data: [] };
    });
    renderPage("/wizard/project/project");
    await screen.findByRole("heading", { name: /: info$/i });
    expect(await screen.findByRole("heading", { name: /^new project$/i })).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: /first project/i })).not.toBeInTheDocument();
  });

  it("falls back to the furthest rung the store can render when a later step is opened cold", async () => {
    renderPage("/wizard/project/service");
    expect(await screen.findByRole("heading", { name: /: info$/i })).toBeInTheDocument();
    expect(rung(/: info$/i)).toHaveAttribute("data-state", "active");
    expect(rung(/: service$/i)).toHaveAttribute("data-state", "upcoming");
  });

  it("door 2: preselects the project from ?project= and opens the repository rung with the project already done", async () => {
    renderPage("/wizard/project/repository?project=p-1");
    expect(await screen.findByRole("heading", { name: /: repository$/i })).toBeInTheDocument();

    const projectRung = rung(/: info$/i);
    expect(projectRung).toHaveAttribute("data-state", "done");
    expect(await within(projectRung).findByText("Backend")).toBeInTheDocument();
    // Locked: the project already exists, so there's no Change affordance to re-open it.
    expect(within(projectRung).queryByRole("button", { name: /change/i })).not.toBeInTheDocument();
    expect(rung(/: repository$/i)).toHaveAttribute("data-state", "active");
  });

  it("door 3: preselects the project from ?stack= and opens the repository rung with the project already done", async () => {
    mocks.get.mockImplementation(async (url: string) => {
      if (url === "/api/stacks/stack-1") return { data: stack };
      if (url === "/api/projects/p-1") return { data: project };
      if (url === "/api/repositories") return { data: { repositories: [] } };
      return { data: [] };
    });
    renderPage("/wizard/project/repository?stack=stack-1");
    expect(await screen.findByRole("heading", { name: /: repository$/i })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: /^attach a repository$/i })).toBeInTheDocument();

    const projectRung = rung(/: info$/i);
    expect(projectRung).toHaveAttribute("data-state", "done");
    expect(await within(projectRung).findByText("Backend")).toBeInTheDocument();
    // Locked: door 3 seeded it too, so there's no Change affordance to re-open it.
    expect(within(projectRung).queryByRole("button", { name: /change/i })).not.toBeInTheDocument();
    expect(rung(/: repository$/i)).toHaveAttribute("data-state", "active");
  });

  it("goes back to home instead of the board when the viewer can't read tickets", async () => {
    access.areas = [];
    const user = userEvent.setup();
    renderPage("/wizard/project/project");

    await user.click(await screen.findByRole("button", { name: "Back" }));

    expect(await screen.findByText("home")).toBeInTheDocument();
  });
});
