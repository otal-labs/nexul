import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { RouterProvider, createMemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ProjectWizardPage } from "@/pages/ProjectWizardPage";
import { useProjectWizardStore } from "@/stores/projectWizardStore";
import { useWorkspaceStore } from "@/stores/workspaceStore";

const access = vi.hoisted(() => ({ areas: ["tickets"] as string[] }));
vi.mock("@/hooks/AccessHooks", () => ({ useAreaAccess: () => (area: string) => access.areas.includes(area) }));

beforeEach(() => {
  access.areas = ["tickets"];
});

// Step components are exercised in full by their own test files (WizardRepositoryStep.test.tsx,
// WizardServiceStep.test.tsx, WizardEnvStep.test.tsx, WizardReachStep.test.tsx); this file only cares about the
// stepper mechanics ProjectWizardPage and its progress row own themselves: which rung the URL opens, an
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
            { path: "/acme/wizard/project/:step", element: <ProjectWizardPage /> },
            { path: "/acme", element: <p>home</p> },
            { path: "/acme/board", element: <p>board</p> },
            { path: "/acme/board/:token", element: <p>project board</p> },
          ],
          { initialEntries: entries, initialIndex: entries.length - 1 },
        )}
      />
    </QueryClientProvider>,
  );
};

const rung = (label: string) =>
  within(screen.getByRole("list", { name: "Project wizard steps" }))
    .getAllByRole("listitem")
    .find((item) => item.textContent?.includes(label))!;

beforeEach(() => {
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
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
    renderPage("/acme/wizard/project/project");

    await screen.findByLabelText("Project name");

    expect(screen.queryByRole("button", { name: "Skip for now" })).not.toBeInTheDocument();
  });

  it("goes back from the info rung to the page the wizard was opened from, creating nothing", async () => {
    const user = userEvent.setup();
    renderPage(["/acme/board/p-1", "/acme/wizard/project/project"]);

    await user.click(await screen.findByRole("button", { name: "Back" }));

    expect(await screen.findByText("project board")).toBeInTheDocument();
    expect(mocks.post).not.toHaveBeenCalled();
  });

  it("goes back from a cold-opened info rung to the board", async () => {
    const user = userEvent.setup();
    renderPage("/acme/wizard/project/project");

    await user.click(await screen.findByRole("button", { name: "Back" }));

    expect(await screen.findByText("board")).toBeInTheDocument();
  });

  it("goes back from the service rung to the repository rung while no stack exists yet", async () => {
    const store = useProjectWizardStore.getState();
    store.setProjectId("p-1", "Backend");
    store.setCandidate({ kind: "compose", path: "compose.yml", name: "api", services: [] });
    const user = userEvent.setup();
    renderPage("/acme/wizard/project/service");
    await screen.findByRole("heading", { name: "Service" });

    await user.click(screen.getByRole("button", { name: "Back" }));

    expect(rung("Repository")).toHaveAttribute("data-state", "current");
    expect(screen.queryByRole("button", { name: "Back" })).not.toBeInTheDocument();
  });

  it("offers no way back from the service rung once its stack exists", async () => {
    const store = useProjectWizardStore.getState();
    store.setProjectId("p-1", "Backend");
    store.setCandidate({ kind: "compose", path: "compose.yml", name: "api", services: [] });
    store.setStackId("stack-1");
    renderPage("/acme/wizard/project/service");
    await screen.findByRole("heading", { name: "Service" });

    expect(screen.queryByRole("button", { name: "Back" })).not.toBeInTheDocument();
  });

  it("keeps the project it just created when the rest is skipped, and lands on its board", async () => {
    mocks.post.mockResolvedValueOnce({ data: project });
    const user = userEvent.setup();
    renderPage("/acme/wizard/project/project");

    await user.type(await screen.findByLabelText("Project name"), "Backend");
    await user.type(screen.getByLabelText("Prefix"), "BE");
    await user.click(screen.getByRole("button", { name: "Continue to Repository" }));
    await screen.findByRole("heading", { name: "Repository" });
    expect(within(rung("Info")).queryByRole("button")).not.toBeInTheDocument();
    await user.click(await screen.findByRole("button", { name: "Skip for now" }));

    expect(await screen.findByText("project board")).toBeInTheDocument();
    expect(mocks.post).toHaveBeenCalledTimes(1);
    expect(useProjectWizardStore.getState().projectId).toBeNull();
  });

  it("redirects an unknown step back to the project step", async () => {
    renderPage("/acme/wizard/project/nonsense");
    expect(await screen.findByRole("heading", { name: "Info" })).toBeInTheDocument();
    expect(rung("Info")).toHaveAttribute("data-state", "current");
  });

  it("opens the project rung by default with everything after it upcoming", async () => {
    renderPage("/acme/wizard/project/project");
    await screen.findByRole("heading", { name: "Info" });
    expect(rung("Info")).toHaveAttribute("data-state", "current");
    expect(rung("Repository")).toHaveAttribute("data-state", "future");
    expect(rung("Service")).toHaveAttribute("data-state", "future");
    expect(rung("Reach")).toHaveAttribute("data-state", "future");
    expect(rung("Deploy branches")).toHaveAttribute("data-state", "future");
    expect(rung("Done")).toHaveAttribute("data-state", "future");
    expect(within(screen.getByRole("list", { name: "Project wizard steps" })).getAllByRole("listitem")).toHaveLength(6);
  });

  it("frames the project rung as the first project when the workspace has none", async () => {
    renderPage("/acme/wizard/project/project");
    expect(await screen.findByRole("heading", { name: /^create your first project$/i })).toBeInTheDocument();
  });

  it("titles the project rung New project once the workspace has one", async () => {
    mocks.get.mockImplementation(async (url: string) => {
      if (url === "/api/projects") return { data: [project] };
      return { data: [] };
    });
    renderPage("/acme/wizard/project/project");
    await screen.findByRole("heading", { name: "Info" });
    expect(await screen.findByRole("heading", { name: /^new project$/i })).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: /first project/i })).not.toBeInTheDocument();
  });

  it("falls back to the furthest rung the store can render when a later step is opened cold", async () => {
    renderPage("/acme/wizard/project/service");
    expect(await screen.findByRole("heading", { name: "Info" })).toBeInTheDocument();
    expect(rung("Info")).toHaveAttribute("data-state", "current");
    expect(rung("Service")).toHaveAttribute("data-state", "future");
  });

  it("door 2: preselects the project from ?project= and opens the repository rung with the project already done", async () => {
    renderPage("/acme/wizard/project/repository?project=p-1");
    expect(await screen.findByRole("heading", { name: "Repository" })).toBeInTheDocument();

    const projectRung = rung("Info");
    expect(projectRung).toHaveAttribute("data-state", "done");
    // Locked: the project already exists, so its node is not a control that could re-open the step.
    expect(within(projectRung).queryByRole("button")).not.toBeInTheDocument();
    expect(rung("Repository")).toHaveAttribute("data-state", "current");
  });

  it("door 3: preselects the project from ?stack= and opens the repository rung with the project already done", async () => {
    mocks.get.mockImplementation(async (url: string) => {
      if (url === "/api/stacks/stack-1") return { data: stack };
      if (url === "/api/projects/p-1") return { data: project };
      if (url === "/api/repositories") return { data: { repositories: [] } };
      return { data: [] };
    });
    renderPage("/acme/wizard/project/repository?stack=stack-1");
    expect(await screen.findByRole("heading", { name: "Repository" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: /^attach a repository$/i })).toBeInTheDocument();

    const projectRung = rung("Info");
    expect(projectRung).toHaveAttribute("data-state", "done");
    // Locked: door 3 seeded it too, so its node is not a control that could re-open the step.
    expect(within(projectRung).queryByRole("button")).not.toBeInTheDocument();
    expect(rung("Repository")).toHaveAttribute("data-state", "current");
  });

  it("goes back to home instead of the board when the viewer can't read tickets", async () => {
    access.areas = [];
    const user = userEvent.setup();
    renderPage("/acme/wizard/project/project");

    await user.click(await screen.findByRole("button", { name: "Back" }));

    expect(await screen.findByText("home")).toBeInTheDocument();
  });
});
