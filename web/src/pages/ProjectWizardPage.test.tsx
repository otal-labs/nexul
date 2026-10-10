import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { RouterProvider, createMemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ProjectWizardPage } from "@/pages/ProjectWizardPage";
import { useProjectWizardStore } from "@/stores/projectWizardStore";
import { pickOption } from "@/test/pickOption";
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
const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn() }));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, post: mocks.post, put: mocks.put, patch: mocks.patch },
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

// The server's answer for a project the wizard made and has not finished.
const mockProjectInSetup = (steps: Record<string, string> = { project: "done" }) =>
  mocks.get.mockImplementation(async (url: string) => {
    if (url === "/api/projects/p-1") return { data: inSetup(steps) };
    if (url === "/api/repositories") return { data: { repositories: [] } };
    return { data: [] };
  });

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
  mocks.post.mockReset();
  mocks.put.mockReset();
  mocks.patch.mockReset();
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
    store.setStackId("stack-1");
    mocks.get.mockImplementation(async (url: string) => {
      if (url === "/api/projects/p-1") return { data: project };
      if (url === "/api/stacks/stack-1") return { data: stack };
      return { data: [] };
    });
    renderPage("/acme/wizard/project/service");

    expect(await screen.findByText(/api runs on/)).toHaveTextContent("api runs on prod.");
    expect(screen.queryByRole("button", { name: /Create/ })).not.toBeInTheDocument();
  });

  it("records a skipped step on the project and moves on to the next step, keeping the project it made", async () => {
    mockProjectInSetup();
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
    mockProjectInSetup();
    mocks.put.mockRejectedValue(new Error("offline"));
    useProjectWizardStore.getState().setProjectId("p-1", "Backend");
    const user = userEvent.setup();
    renderPage("/acme/wizard/project/repository?project=p-1");

    await user.click(await screen.findByRole("button", { name: "Skip for now" }));

    expect(screen.getByRole("heading", { name: "Repository" })).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Service" })).not.toBeInTheDocument();
  });

  it("can skip Environment before a service exists and advances to Reach", async () => {
    mockProjectInSetup();
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
        return { data: { ...project, setup: { finished, stack_id: "stack-1", steps: { project: "done", repository: "done", service: "done" } } } };
      if (url === "/api/stacks") return { data: [stack, { ...stack, id: "stack-new", name: "other", created_at: "2026-10-10" }] };
      if (url === "/api/stacks/stack-1") return { data: stack };
      return { data: [] };
    });
    renderPage(`/acme/wizard/project/service?project=p-1${query}`);

    expect(await screen.findByText(/api runs on/)).toHaveTextContent("api runs on prod.");
    expect(screen.getByRole("heading", { name: title })).toBeInTheDocument();
    expect(rung("Repository")).toHaveAttribute("data-state", "done");
  });

  it.each([
    { name: "a project still in setup", finished: false },
    { name: "a finished project", finished: true },
  ])("the Add a service door on $name makes a new service beside the recorded one and leaves setup alone", async ({ finished }) => {
    const store = useProjectWizardStore.getState();
    store.setProjectId("p-1", "Backend");
    store.setCandidate({ kind: "compose", path: "compose.yml", name: "api", services: [] });
    mocks.get.mockImplementation(async (url: string) => {
      if (url === "/api/projects/p-1")
        return { data: { ...project, setup: { finished, stack_id: "stack-old", env_keys: ["PORT"], steps: { project: "done", repository: "done", service: "done" } } } };
      if (url === "/api/stacks/stack-old") return { data: { ...stack, id: "stack-old", name: "old" } };
      if (url === "/api/stacks/stack-1") return { data: stack };
      if (url === "/api/machines") return { data: [{ id: "m-1", name: "prod", stack_root: "/data/nexul", first_seen: "", last_seen: "" }] };
      return { data: [] };
    });
    mocks.post.mockResolvedValueOnce({ data: stack });
    const user = userEvent.setup();
    renderPage("/acme/wizard/project/service?project=p-1&add=1");

    await pickOption(user, "Machine", "prod");
    await user.click(screen.getByRole("button", { name: "Create & deploy" }));

    expect(await screen.findByRole("heading", { name: "Reach" })).toBeInTheDocument();
    expect(mocks.post).toHaveBeenCalledTimes(1);
    expect(mocks.put).not.toHaveBeenCalled();
  });

  it("says the repository needs scanning again when setup recorded it but this browser holds no scan, and jumps there without undoing it", async () => {
    mockProjectInSetup({ project: "done", repository: "done" });
    const user = userEvent.setup();
    renderPage("/acme/wizard/project/service?project=p-1");

    expect(await screen.findByText(/scan isn't loaded on this device/)).toBeInTheDocument();
    expect(screen.queryByText(/Pick one first/)).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Go to Repository" }));

    expect(await screen.findByRole("heading", { name: "Repository" })).toBeInTheDocument();
    expect(rung("Repository")).toHaveAttribute("data-state", "current");
    expect(mocks.put).not.toHaveBeenCalled();
  });

  it("retries a failed Service mark from the summary before advancing, without creating again", async () => {
    const store = useProjectWizardStore.getState();
    store.setProjectId("p-1", "Backend");
    store.setCandidate({ kind: "compose", path: "compose.yml", name: "api", services: [] });
    mocks.get.mockImplementation(async (url: string) => {
      if (url === "/api/projects/p-1") return { data: inSetup({ project: "done" }) };
      if (url === "/api/stacks/stack-1") return { data: stack };
      if (url === "/api/machines") return { data: [{ id: "m-1", name: "prod", stack_root: "/data/nexul", first_seen: "", last_seen: "" }] };
      return { data: [] };
    });
    mocks.post.mockResolvedValueOnce({ data: stack });
    mocks.put.mockRejectedValueOnce(new Error("offline")).mockRejectedValueOnce(new Error("offline"));
    mocks.put.mockResolvedValueOnce({ data: inSetup({ project: "done", service: "done" }) });
    const user = userEvent.setup();
    renderPage("/acme/wizard/project/service?project=p-1");

    await pickOption(user, "Machine", "prod");
    await user.click(screen.getByRole("button", { name: "Create & deploy" }));
    await screen.findByText(/api runs on/);
    await vi.waitFor(() => expect(mocks.put).toHaveBeenCalledTimes(1));
    await user.click(screen.getByRole("button", { name: "Continue to Reach" }));

    expect(screen.getByRole("heading", { name: "Service" })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Continue to Reach" }));
    expect(await screen.findByRole("heading", { name: "Reach" })).toBeInTheDocument();
    expect(mocks.put).toHaveBeenCalledTimes(3);
    expect(mocks.put).toHaveBeenLastCalledWith("/api/projects/p-1/setup", expect.objectContaining({
      steps: { service: "done" }, stack_id: "stack-1", env_keys: [],
    }));
    expect(mocks.post).toHaveBeenCalledTimes(1);
  });

  it("restores detected Environment fields on a fresh device, saves to the recorded stack, and deploys its branch", async () => {
    mocks.get.mockImplementation(async (url: string) => {
      if (url === "/api/projects/p-1") return { data: { ...project,
        setup: { finished: false, steps: { project: "done", repository: "done", service: "done" }, stack_id: "stack-1", env_keys: ["PORT"] },
      } };
      if (url === "/api/stacks") return { data: [stack] };
      if (url === "/api/stacks/stack-1") return { data: { ...stack, env: { PORT: "8000", EXTRA: "1" }, build_source: { branch: "release" } } };
      return { data: [] };
    });
    mocks.put.mockResolvedValue({ data: inSetup({ project: "done", service: "done", env: "done" }) });
    mocks.post.mockResolvedValue({ data: { id: "deploy-1" } });
    mocks.patch.mockResolvedValue({ data: stack });
    const user = userEvent.setup();
    renderPage("/acme/wizard/project/env?project=p-1");

    const port = await screen.findByLabelText("PORT");
    expect(port).toHaveValue("8000");
    await user.clear(port);
    await user.type(port, "8080");
    expect(rung("Environment")).toHaveAttribute("data-state", "current");
    await user.click(screen.getByRole("button", { name: "Save & deploy" }));

    expect(await screen.findByRole("heading", { name: "Reach" })).toBeInTheDocument();
    expect(mocks.patch).toHaveBeenCalledWith("/api/stacks/stack-1", expect.objectContaining({ env: { PORT: "8080", EXTRA: "1" } }));
    expect(mocks.post).toHaveBeenCalledWith("/api/deploys", expect.objectContaining({ stack_id: "stack-1", ref: "release" }));
  });

  it("keeps restored service context when Repository scans another repository", async () => {
    const setup = { finished: false, stack_id: "stack-1", env_keys: ["PORT"], steps: { project: "done", service: "done" } };
    mocks.get.mockImplementation(async (url: string) => {
      if (url === "/api/projects/p-1") return { data: { ...project, setup } };
      if (url === "/api/stacks/stack-1") return { data: { ...stack, env: { PORT: "8000" }, build_source: { branch: "release" } } };
      if (url === "/api/repositories") return { data: { repositories: [{ id: 2, owner: "acme", name: "other", full_name: "acme/other", provider: "github", default_branch: "develop" }] } };
      return { data: [] };
    });
    mocks.post.mockImplementation(async (url: string) => {
      if (url.includes("scan")) return { data: { default_branch: "develop", env_keys: ["OTHER_KEY"], candidates: [{ kind: "compose", path: "compose.yml", name: "other", services: [] }] } };
      return { data: { id: "deploy-1" } };
    });
    mocks.put.mockResolvedValue({ data: { ...project, setup } });
    mocks.patch.mockResolvedValue({ data: stack });
    const user = userEvent.setup();
    renderPage("/acme/wizard/project/service?project=p-1");
    await screen.findByText(/api runs on/);
    await user.click(within(rung("Repository")).getByRole("button"));
    await user.type(screen.getByPlaceholderText("Search repositories…"), "other");
    await user.click(await screen.findByRole("button", { name: /acme.*other/ }));
    await screen.findByRole("heading", { name: "Service" });
    await user.click(screen.getByRole("button", { name: "Continue to Environment" }));
    expect(await screen.findByLabelText("PORT")).toHaveValue("8000");
    expect(screen.queryByLabelText("OTHER_KEY")).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Save & deploy" }));
    await screen.findByRole("heading", { name: "Reach" });
    expect(mocks.post).toHaveBeenLastCalledWith("/api/deploys", expect.objectContaining({ stack_id: "stack-1", ref: "release" }));
    for (const [, body] of mocks.put.mock.calls) expect(body.env_keys ?? ["PORT"]).toEqual(["PORT"]);
  });

  it.each(["Finish", "Back"])("%s retries pending service context after free navigation away from a failed Service save", async (action) => {
    const store = useProjectWizardStore.getState();
    store.setProjectId("p-1", "Backend");
    store.setScanResult({ default_branch: "main", candidates: [], env_keys: ["PORT"] });
    store.setCandidate({ kind: "compose", path: "compose.yml", name: "api", services: [] });
    mocks.get.mockImplementation(async (url: string) => {
      if (url === "/api/projects/p-1") return { data: inSetup({ project: "done" }) };
      if (url === "/api/projects") return { data: [project] };
      if (url === "/api/stacks/stack-1") return { data: stack };
      if (url === "/api/memories") return { data: [{ id: "m-1", project_id: "p-1", kind: "interview", body: "## Stack" }] };
      if (url === "/api/machines") return { data: [{ id: "m-1", name: "prod", stack_root: "/data/nexul", first_seen: "", last_seen: "" }] };
      return { data: [] };
    });
    mocks.post.mockResolvedValue({ data: stack });
    mocks.put.mockRejectedValueOnce(new Error("offline")).mockRejectedValueOnce(new Error("offline"));
    mocks.put.mockResolvedValue({ data: project });
    const user = userEvent.setup();
    renderPage("/acme/wizard/project/service?project=p-1");
    await pickOption(user, "Machine", "prod");
    await user.click(screen.getByRole("button", { name: "Create" }));
    await screen.findByText(/api runs on/);
    await vi.waitFor(() => expect(mocks.put).toHaveBeenCalledTimes(1));
    const target = action === "Finish" ? "Done" : "Info";
    await user.click(within(rung(target)).getByRole("button"));
    await user.click(await screen.findByRole("button", { name: action }));
    await vi.waitFor(() => expect(mocks.put).toHaveBeenCalledTimes(2));
    expect(screen.getByRole("heading", { name: target })).toBeInTheDocument();
    expect(useProjectWizardStore.getState().stackId).toBe("stack-1");
    await user.click(screen.getByRole("button", { name: action }));
    if (action === "Finish") {
      await screen.findByText("project board");
    } else {
      await screen.findByRole("heading", { name: "Service" });
    }
    expect(mocks.put).toHaveBeenLastCalledWith("/api/projects/p-1/setup", expect.objectContaining({
      stack_id: "stack-1", env_keys: ["PORT"], steps: { service: "done" },
      ...(action === "Finish" && { finished: true }),
    }));
    expect(mocks.post).toHaveBeenCalledTimes(1);
  });

  it("skips a directly opened Environment URL with no scan to Reach rather than Info", async () => {
    mocks.put.mockResolvedValue({ data: inSetup({ env: "skipped" }) });
    const user = userEvent.setup();
    renderPage("/acme/wizard/project/env?project=p-1");

    await user.click(await screen.findByRole("button", { name: "Skip for now" }));

    expect(await screen.findByRole("heading", { name: "Reach" })).toBeInTheDocument();
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
