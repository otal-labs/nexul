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
// stepper mechanics ProjectWizardPage owns: which rung the URL opens, what a skip records, what a step opened early
// says it needs, and the doors that arrive with a project.
const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), put: vi.fn() }));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, post: mocks.post, put: mocks.put },
  errorMessage: vi.fn(() => ""),
}));

const project = {
  id: "p-1",
  name: "Backend",
  prefix: "BE",
  position: 0,
  workspace_id: "ws-1",
  setup: { finished: true, steps: {} },
  created_at: "",
  updated_at: "",
};

const inSetup = (steps: Record<string, string>) => ({ ...project, setup: { finished: false, steps } });

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
  });

  it("shows the service a revisited Service step made, instead of making a second", async () => {
    const store = useProjectWizardStore.getState();
    store.setProjectId("p-1", "Backend");
    store.setCandidate({ kind: "compose", path: "compose.yml", name: "api", services: [] });
    store.setName("api");
    store.setMachine("prod");
    store.setStackId("stack-1");
    renderPage("/acme/wizard/project/service");

    expect(await screen.findByText(/api runs on/)).toHaveTextContent("api runs on prod.");
    expect(screen.queryByRole("button", { name: /Create/ })).not.toBeInTheDocument();
  });

  it("records a skipped step on the project and moves on to the next step, keeping the project it made", async () => {
    mocks.post.mockResolvedValueOnce({ data: inSetup({ project: "done" }) });
    mocks.put.mockResolvedValue({ data: inSetup({ project: "done", repository: "skipped" }) });
    const user = userEvent.setup();
    renderPage("/acme/wizard/project/project");

    await user.type(await screen.findByLabelText("Project name"), "Backend");
    await user.type(screen.getByLabelText("Prefix"), "BE");
    await user.click(screen.getByRole("button", { name: "Continue to Repository" }));
    await user.click(await screen.findByRole("button", { name: "Skip for now" }));

    expect(await screen.findByRole("heading", { name: "Service" })).toBeInTheDocument();
    expect(mocks.put).toHaveBeenCalledWith("/api/projects/p-1/setup", { finished: undefined, steps: { repository: "skipped" } });
    expect(mocks.post).toHaveBeenCalledTimes(1);
  });

  it("opens a step ahead of its groundwork and says what it needs, with a jump there", async () => {
    const user = userEvent.setup();
    renderPage("/acme/wizard/project/reach?project=p-1");

    expect(await screen.findByText("Reach gives a service a hostname. There's no service yet.")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Go to Service" }));

    expect(await screen.findByText("A service builds from a repository. Pick one first.")).toBeInTheDocument();
    expect(rung("Service")).toHaveAttribute("data-state", "current");
  });

  it("keeps the step open when its skip cannot be saved", async () => {
    mocks.put.mockRejectedValue(new Error("offline"));
    useProjectWizardStore.getState().setProjectId("p-1", "Backend");
    const user = userEvent.setup();
    renderPage("/acme/wizard/project/repository?project=p-1");

    await user.click(await screen.findByRole("button", { name: "Skip for now" }));

    expect(screen.getByRole("heading", { name: "Repository" })).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Service" })).not.toBeInTheDocument();
  });

  it("can skip Environment before a service exists and advances to Reach", async () => {
    mocks.put.mockResolvedValue({ data: inSetup({ project: "done", env: "skipped" }) });
    const store = useProjectWizardStore.getState();
    store.setProjectId("p-1", "Backend");
    store.setScanResult({ default_branch: "main", candidates: [], env_keys: ["PORT"] });
    const user = userEvent.setup();
    renderPage("/acme/wizard/project/env?project=p-1");

    await user.click(await screen.findByRole("button", { name: "Skip for now" }));

    expect(await screen.findByRole("heading", { name: "Reach" })).toBeInTheDocument();
    expect(mocks.put).toHaveBeenCalledWith("/api/projects/p-1/setup", { finished: undefined, steps: { env: "skipped" } });
  });

  it.each([
    { finished: false, query: "", title: "Continue setup" },
    { finished: true, query: "&revisit=1", title: "Project setup" },
  ])("opens a recorded service from another device with setup finished=$finished", async ({ finished, query, title }) => {
    mocks.get.mockImplementation(async (url: string) => {
      if (url === "/api/projects/p-1")
        return { data: { ...project, setup: { finished, steps: { project: "done", repository: "done", service: "done" } } } };
      if (url === "/api/stacks") return { data: [stack] };
      return { data: [] };
    });
    renderPage(`/acme/wizard/project/service?project=p-1${query}`);

    expect(await screen.findByText(/api runs on/)).toHaveTextContent("api runs on prod.");
    expect(screen.getByRole("heading", { name: title })).toBeInTheDocument();
    expect(rung("Repository")).toHaveAttribute("data-state", "done");
  });

  it("redirects an unknown step back to the project step", async () => {
    renderPage("/acme/wizard/project/nonsense");
    expect(await screen.findByRole("heading", { name: "Info" })).toBeInTheDocument();
    expect(rung("Info")).toHaveAttribute("data-state", "current");
  });

  it("opens the project rung by default with nothing after it visited", async () => {
    renderPage("/acme/wizard/project/project");
    await screen.findByRole("heading", { name: "Info" });
    expect(rung("Info")).toHaveAttribute("data-state", "current");
    for (const label of ["Repository", "Service", "Reach", "Deploy branches", "Done"]) {
      expect(rung(label)).toHaveAttribute("data-state", "unvisited");
    }
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

  it("door 2: preselects the project from ?project= and opens the repository rung with Info as the project records it", async () => {
    mocks.get.mockImplementation(async (url: string) => {
      if (url === "/api/projects/p-1") return { data: { ...project, setup: { finished: true, steps: { project: "done" } } } };
      return { data: [] };
    });
    renderPage("/acme/wizard/project/repository?project=p-1");
    expect(await screen.findByRole("heading", { name: "Repository" })).toBeInTheDocument();
    expect(await screen.findByRole("heading", { name: /^add a service$/i })).toBeInTheDocument();

    await vi.waitFor(() => expect(rung("Info")).toHaveAttribute("data-state", "done"));
    expect(rung("Repository")).toHaveAttribute("data-state", "current");
  });

  it("door 3: preselects the project from ?stack= and opens the repository rung", async () => {
    mocks.get.mockImplementation(async (url: string) => {
      if (url === "/api/stacks/stack-1") return { data: stack };
      if (url === "/api/projects/p-1") return { data: project };
      if (url === "/api/repositories") return { data: { repositories: [] } };
      return { data: [] };
    });
    renderPage("/acme/wizard/project/repository?stack=stack-1");
    expect(await screen.findByRole("heading", { name: "Repository" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: /^attach a repository$/i })).toBeInTheDocument();

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
