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
vi.mock("@/hooks/AccessHooks", () => ({
  useAreaAccess: () => (area: string) => access.areas.includes(area),
  useCanAddMachines: () => false,
  useCanOpenSection: () => false,
  useVisibleSettingsSections: () => [],
}));

beforeEach(() => {
  access.areas = ["tickets"];
});

// Step components are exercised in full by their own test files (WizardSetupRepositoryStep.test.tsx,
// WizardServiceStep.test.tsx, WizardEnvStep.test.tsx, WizardReachStep.test.tsx); this file only cares about the
// stepper mechanics ProjectWizardPage owns: which rung the URL opens, what a skip records, what a step opened early
// says it needs, what Deploy and Service read from the attached repository, and the doors that arrive with a project.
const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn() }));

// The person's GitHub link is connected throughout, so each test's GET mock stays about the wizard.
vi.mock("@/api/client", async () => {
  const { withOwner } = await import("@/test/viewer");
  return {
    api: {
      get: withOwner((url: string, ...rest: unknown[]) =>
        url === "/api/auth/github-link" ? Promise.resolve({ data: { state: "connected" } }) : mocks.get(url, ...rest),
      ),
      post: mocks.post,
      put: mocks.put,
      patch: mocks.patch,
    },
    errorMessage: vi.fn(() => ""),
  };
});

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

// The machine stack runs on, where the viewer holds Deploy to.
const stackMachines = [{ id: "m-1", name: "prod", stack_root: "/data/nexul", first_seen: "", last_seen: "", actions: ["machines:see", "machines:deploy"] }];

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

const rungs = () => within(screen.getByRole("list", { name: "Project wizard steps" })).getAllByRole("listitem");
const rung = (label: string) => rungs().find((item) => item.textContent?.startsWith(label))!;
const railWord = (word: string) => within(screen.getByRole("navigation", { name: "Project wizard" })).getByText(word);

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
  it("goes back from the info rung to the page the wizard was opened from, creating nothing", async () => {
    const user = userEvent.setup();
    renderPage(["/acme/board/p-1", "/acme/wizard/project/project"]);

    await user.click(await screen.findByRole("button", { name: "Back" }));

    expect(await screen.findByText("project board")).toBeInTheDocument();
    expect(mocks.post).not.toHaveBeenCalled();
  });

  it.each([
    ["tickets", "board"],
    ["", "home"],
  ])("goes back from a cold-opened info rung to the board, or home when the viewer holds %j", async (area, landing) => {
    access.areas = area ? [area] : [];
    const user = userEvent.setup();
    renderPage("/acme/wizard/project/project");

    await user.click(await screen.findByRole("button", { name: "Back" }));

    expect(await screen.findByText(landing)).toBeInTheDocument();
  });

  it.each([
    { name: "Finish later from a project in setup", project: true, label: "Finish later", landing: "project board" },
    { name: "Exit before a project exists", project: false, label: "Exit", landing: "board" },
  ])("leaves mid-wizard by $name, landing on the board instead of the step before", async ({ project, label, landing }) => {
    if (project) {
      mockProjectInSetup({ project: "done" });
      useProjectWizardStore.getState().setProjectId("p-1", "Backend");
    }
    const user = userEvent.setup();
    renderPage(["/acme/wizard/project/project", "/acme/wizard/project/people"]);

    await user.click(await screen.findByRole("button", { name: label }));

    expect(await screen.findByText(landing)).toBeInTheDocument();
  });

  it("goes back from the service rung to the deploy rung while no stack exists yet", async () => {
    const store = useProjectWizardStore.getState();
    store.setProjectId("p-1", "Backend");
    store.setCandidate({ kind: "compose", path: "compose.yml", name: "api", services: [] });
    const user = userEvent.setup();
    renderPage("/acme/wizard/project/service");
    await screen.findByRole("heading", { name: "Where should it run?" });

    await user.click(screen.getByRole("button", { name: "Back" }));

    expect(rung("Deploy")).toHaveAttribute("data-state", "current");
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
    mocks.put.mockResolvedValue({ data: inSetup({ project: "done", people: "skipped" }) });
    const user = userEvent.setup();
    renderPage("/acme/wizard/project/project");

    await user.type(await screen.findByLabelText("Project name"), "Backend");
    await user.type(screen.getByLabelText("Prefix"), "BE");
    await user.click(screen.getByRole("button", { name: "Continue" }));
    await user.click(await screen.findByRole("button", { name: "Skip for now" }));

    expect(await screen.findByRole("heading", { name: "Where's the code?" })).toBeInTheDocument();
    expect(mocks.put).toHaveBeenCalledWith("/api/projects/p-1/setup", { finished: undefined, steps: { people: "skipped" } });
    expect(mocks.post).toHaveBeenCalledTimes(1);
  });

  it("opens a step ahead of its groundwork and says what it needs, with a jump there", async () => {
    const user = userEvent.setup();
    renderPage("/acme/wizard/project/reach?project=p-1");

    expect(await screen.findByText("Create the service first.")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Go to Service" }));

    expect(await screen.findByText("Pick a repository first.")).toBeInTheDocument();
    expect(rung("Service")).toHaveAttribute("data-state", "current");
  });

  it("keeps the step open when its skip cannot be saved", async () => {
    mockProjectInSetup();
    mocks.put.mockRejectedValue(new Error("offline"));
    useProjectWizardStore.getState().setProjectId("p-1", "Backend");
    const user = userEvent.setup();
    renderPage("/acme/wizard/project/service?project=p-1");

    await user.click(await screen.findByRole("button", { name: "Skip for now" }));

    expect(screen.getByRole("heading", { name: "Where should it run?" })).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Backend is ready" })).not.toBeInTheDocument();
  });

  describe("leaving Repository without a service", () => {
    const portal = { id: 7, owner: "acme", name: "portal", full_name: "acme/portal", default_branch: "main", html_url: "", provider: "github" };
    let attached: unknown[] = [];

    beforeEach(() => {
      attached = [];
      mocks.get.mockImplementation(async (url: string) => {
        if (url === "/api/projects/p-1") return { data: inSetup({ project: "done", people: "done" }) };
        if (url === "/api/projects/p-1/repos") return { data: attached };
        if (url === "/api/repositories") return { data: { repositories: [portal] } };
        if (url === "/api/memories") return { data: [{ id: "m-1", project_id: "p-1", kind: "interview", body: "## Stack" }] };
        return { data: [] };
      });
    });

    it("No repository yet skips the deploy steps, so Continue setup lands on Done, and a place to go next finishes setup", async () => {
      access.areas = ["tickets", "docs"];
      mocks.put.mockResolvedValue({ data: inSetup({ project: "done", people: "done", repository: "skipped" }) });
      const user = userEvent.setup();
      renderPage("/acme/wizard/project/repository?project=p-1");

      await screen.findByLabelText("Search repositories");
      await user.click(screen.getByRole("button", { name: /No repository yet/ }));

      expect(await screen.findByRole("heading", { name: "Backend is ready" })).toBeInTheDocument();
      expect(mocks.put).toHaveBeenCalledWith("/api/projects/p-1/setup", {
        finished: undefined,
        steps: { repository: "skipped", service: "skipped", reach: "skipped", branches: "skipped" },
      });
      expect(screen.getByText("needs a repository")).toBeInTheDocument();
      await user.click(screen.getByRole("button", { name: /^Board/ }));
      expect(await screen.findByText("project board")).toBeInTheDocument();
      expect(mocks.put).toHaveBeenLastCalledWith("/api/projects/p-1/setup", { finished: true });
    });

    it("Continue attaches the picked repository without scanning it, and Deploy then reads it, so Later ends on Done waiting to deploy it", async () => {
      mocks.post.mockImplementation(async (url: string) => {
        if (url === "/api/repositories/scan") {
          return { data: { default_branch: "main", env_keys: [], candidates: [{ kind: "compose", path: "compose.yaml", name: "portal", services: [] }] } };
        }
        attached = [{ owner: "acme", name: "portal", full_name: "acme/portal", connector_id: "github", role: "app" }];
        return { data: undefined };
      });
      mocks.put.mockResolvedValue({ data: inSetup({ project: "done", people: "done", repository: "done" }) });
      const user = userEvent.setup();
      renderPage("/acme/wizard/project/repository?project=p-1");

      await user.type(await screen.findByPlaceholderText("Search repositories…"), "portal");
      await user.click(await screen.findByRole("button", { name: /acme\/portal/ }));
      expect(mocks.post).not.toHaveBeenCalled();
      await user.click(screen.getByRole("button", { name: "Continue" }));

      expect(await screen.findByRole("radio", { name: /Deploy now/ })).toBeChecked();
      expect(mocks.post).toHaveBeenCalledWith("/api/projects/p-1/repos", { owner: "acme", name: "portal", connector_id: "github" });
      expect(mocks.post).toHaveBeenCalledWith("/api/repositories/scan", { owner: "acme", name: "portal", workspace_id: "ws-1" });
      await user.click(screen.getByRole("radio", { name: /Later/ }));
      await user.click(screen.getByRole("button", { name: "Continue" }));

      expect(await screen.findByRole("heading", { name: "Backend is ready" })).toBeInTheDocument();
      expect(mocks.put).toHaveBeenLastCalledWith("/api/projects/p-1/setup", {
        finished: undefined,
        steps: { repository: "done", service: "skipped", reach: "skipped", branches: "skipped" },
      });
      expect(screen.getByText("later")).toBeInTheDocument();
    });
  });

  it.each([
    ["service", { project: "done" }, { service: "skipped", reach: "skipped", branches: "skipped" }],
    ["env", { project: "done" }, { env: "skipped", service: "skipped", reach: "skipped", branches: "skipped" }],
  ])("skipping %s with no service made skips every deploy step with it and lands on Done", async (step, steps, recorded) => {
    mockProjectInSetup(steps);
    mocks.put.mockResolvedValue({ data: inSetup({ ...steps, ...recorded }) });
    const store = useProjectWizardStore.getState();
    store.setProjectId("p-1", "Backend");
    if (step === "env") store.setScanResult({ default_branch: "main", candidates: [], env_keys: ["PORT"] });
    const user = userEvent.setup();
    renderPage(`/acme/wizard/project/${step}?project=p-1`);

    await user.click(await screen.findByRole("button", { name: "Skip for now" }));

    expect(await screen.findByRole("heading", { name: "Backend is ready" })).toBeInTheDocument();
    expect(mocks.put).toHaveBeenCalledTimes(1);
    expect(mocks.put).toHaveBeenCalledWith("/api/projects/p-1/setup", { finished: undefined, steps: recorded });
    expect(screen.queryByText("Create the service first.")).not.toBeInTheDocument();
  });

  it.each([
    { finished: false, query: "" },
    { finished: true, query: "&revisit=1" },
  ])("opens a recorded service from another device with setup finished=$finished", async ({ finished, query }) => {
    mocks.get.mockImplementation(async (url: string) => {
      if (url === "/api/projects/p-1")
        return { data: { ...project, setup: { finished, stack_id: "stack-1", steps: { project: "done", repository: "done", service: "done" } } } };
      if (url === "/api/stacks") return { data: [stack, { ...stack, id: "stack-new", name: "other", created_at: "2026-10-10" }] };
      if (url === "/api/stacks/stack-1") return { data: stack };
      return { data: [] };
    });
    renderPage(`/acme/wizard/project/service?project=p-1${query}`);

    expect(await screen.findByText(/api runs on/)).toHaveTextContent("api runs on prod.");
    expect(railWord("Project setup")).toBeInTheDocument();
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
      if (url === "/api/machines/deployable") return { data: [{ id: "m-1", name: "prod", stack_root: "/data/nexul", first_seen: "", last_seen: "" }] };
      return { data: [] };
    });
    mocks.post.mockResolvedValueOnce({ data: stack });
    const user = userEvent.setup();
    renderPage("/acme/wizard/project/service?project=p-1&add=1");

    await pickOption(user, "Machine", "prod");
    await user.click(screen.getByRole("button", { name: "Create & deploy" }));

    expect(await screen.findByRole("heading", { name: "Give it a domain" })).toBeInTheDocument();
    expect(mocks.post).toHaveBeenCalledTimes(1);
    expect(mocks.put).not.toHaveBeenCalled();
  });

  it("retries a failed Service mark from the summary before advancing, without creating again", async () => {
    const store = useProjectWizardStore.getState();
    store.setProjectId("p-1", "Backend");
    store.setCandidate({ kind: "compose", path: "compose.yml", name: "api", services: [] });
    mocks.get.mockImplementation(async (url: string) => {
      if (url === "/api/projects/p-1") return { data: inSetup({ project: "done" }) };
      if (url === "/api/stacks/stack-1") return { data: stack };
      if (url === "/api/machines/deployable") return { data: [{ id: "m-1", name: "prod", stack_root: "/data/nexul", first_seen: "", last_seen: "" }] };
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
    await user.click(screen.getByRole("button", { name: "Continue" }));

    expect(screen.getByRole("heading", { name: "Where should it run?" })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Continue" }));
    expect(await screen.findByRole("heading", { name: "Give it a domain" })).toBeInTheDocument();
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
      if (url === "/api/machines") return { data: stackMachines };
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
    await user.click(await screen.findByRole("button", { name: "Save & deploy" }));

    expect(await screen.findByRole("heading", { name: "Give it a domain" })).toBeInTheDocument();
    expect(mocks.patch).toHaveBeenCalledWith("/api/stacks/stack-1", expect.objectContaining({ env: { PORT: "8080", EXTRA: "1" } }));
    expect(mocks.post).toHaveBeenCalledWith("/api/deploys", expect.objectContaining({ stack_id: "stack-1", ref: "release" }));
  });

  it("keeps restored service context when Repository picks another repository", async () => {
    const setup = { finished: false, stack_id: "stack-1", env_keys: ["PORT"], steps: { project: "done", service: "done" } };
    mocks.get.mockImplementation(async (url: string) => {
      if (url === "/api/projects/p-1") return { data: { ...project, setup } };
      if (url === "/api/machines") return { data: stackMachines };
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
    await user.type(await screen.findByPlaceholderText("Search repositories…"), "other");
    await user.click(await screen.findByRole("button", { name: /acme.*other/ }));
    await user.click(await screen.findByRole("button", { name: "Continue" }));
    expect(await screen.findByRole("heading", { name: "Deploy it now?" })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Continue" }));
    const port = await screen.findByLabelText("PORT");
    expect(port).toHaveValue("8000");
    expect(screen.queryByLabelText("OTHER_KEY")).not.toBeInTheDocument();
    await user.type(port, "1");
    await user.click(await screen.findByRole("button", { name: "Save & deploy" }));
    await screen.findByRole("heading", { name: "Give it a domain" });
    expect(mocks.post).toHaveBeenLastCalledWith("/api/deploys", expect.objectContaining({ stack_id: "stack-1", ref: "release" }));
    expect(mocks.post).not.toHaveBeenCalledWith("/api/repositories/scan", expect.anything());
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
      if (url === "/api/machines/deployable") return { data: [{ id: "m-1", name: "prod", stack_root: "/data/nexul", first_seen: "", last_seen: "" }] };
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
    const target = action === "Finish" ? "Done" : "Name";
    const heading = action === "Finish" ? "Backend is ready" : "Name your project";
    await user.click(within(rung(target)).getByRole("button"));
    await user.click(await screen.findByRole("button", { name: action }));
    await vi.waitFor(() => expect(mocks.put).toHaveBeenCalledTimes(2));
    expect(screen.getByRole("heading", { name: heading })).toBeInTheDocument();
    expect(useProjectWizardStore.getState().stackId).toBe("stack-1");
    await user.click(screen.getByRole("button", { name: action }));
    if (action === "Finish") {
      await screen.findByText("project board");
    } else {
      await screen.findByRole("heading", { name: "Where should it run?" });
    }
    expect(mocks.put).toHaveBeenLastCalledWith("/api/projects/p-1/setup", expect.objectContaining({
      stack_id: "stack-1", env_keys: ["PORT"], steps: { service: "done" },
      ...(action === "Finish" && { finished: true }),
    }));
    expect(mocks.post).toHaveBeenCalledTimes(1);
  });

  it("redirects an unknown step back to the project step", async () => {
    renderPage("/acme/wizard/project/nonsense");
    expect(await screen.findByRole("heading", { name: "Name your project" })).toBeInTheDocument();
    expect(rung("Name")).toHaveAttribute("data-state", "current");
  });

  it("opens the project rung by default with nothing after it visited, the deploy steps folded into Deploy, and no way to skip it", async () => {
    renderPage("/acme/wizard/project/project");
    await screen.findByRole("heading", { name: "Name your project" });
    expect(screen.queryByRole("button", { name: "Skip for now" })).not.toBeInTheDocument();
    expect(railWord("New project")).toBeInTheDocument();
    expect(rung("Name")).toHaveAttribute("data-state", "current");
    for (const label of ["People", "Repository", "Deploy", "Done"]) {
      expect(rung(label)).toHaveAttribute("data-state", "unvisited");
    }
    expect(rungs().map((item) => item.textContent)).toEqual(["Name", "People", "Repository", "Deploy", "Done"]);
  });

  it("door 2: preselects the project from ?project= and opens Repository, with a rail that shows only what this run passed", async () => {
    mocks.get.mockImplementation(async (url: string) => {
      if (url === "/api/projects/p-1")
        return { data: { ...project, setup: { finished: false, steps: { project: "done", people: "done", reach: "skipped" } } } };
      return { data: [] };
    });
    renderPage("/acme/wizard/project/repository?project=p-1&add=1");
    expect(await screen.findByRole("heading", { name: "Where's the code?" })).toBeInTheDocument();

    expect(await within(screen.getByRole("navigation", { name: "Project wizard" })).findByText("Backend")).toBeInTheDocument();
    expect(railWord("New service")).toBeInTheDocument();
    expect(rungs().map((item) => item.textContent)).toEqual(["Repository", "Service", "Domain", "Branches", "Done"]);
    expect(rung("Repository")).toHaveAttribute("data-state", "current");
    for (const label of ["Service", "Domain", "Branches", "Done"]) expect(rung(label)).toHaveAttribute("data-state", "unvisited");
  });

  describe("adding a service while a repository waits to deploy", () => {
    const attached = (full: string) => ({ owner: full.split("/")[0], name: full.split("/")[1], full_name: full, connector_id: "github", role: "app" });

    beforeEach(() => {
      access.areas = ["tickets", "projects", "stacks", "newProject", "editProjects", "editStacks", "editDeploys"];
      mocks.post.mockResolvedValue({
        data: { default_branch: "main", env_keys: [], candidates: [{ kind: "compose", path: "compose.yaml", name: "web", services: [] }] },
      });
    });

    const mockWaiting = (repos: string[]) =>
      mocks.get.mockImplementation(async (url: string) => {
        if (url === "/api/projects/p-1") return { data: project };
        if (url === "/api/projects/p-1/repos") return { data: repos.map(attached) };
        if (url === "/api/repositories") return { data: { repositories: [] } };
        return { data: [] };
      });

    it.each([
      ["the only one", ["acme/portal"], "", "acme/portal"],
      ["the one the link names", ["acme/portal", "acme/site"], "&repo=acme%2Fsite", "acme/site"],
    ])("scans %s and opens Service without asking for it", async (_, repos, query, picked) => {
      mockWaiting(repos);
      renderPage(`/acme/wizard/project/repository?project=p-1&add=1${query}`);

      expect(await screen.findByRole("heading", { name: "Where should it run?" })).toBeInTheDocument();
      const [owner, name] = picked.split("/");
      expect(mocks.post).toHaveBeenCalledWith("/api/repositories/scan", { owner, name, workspace_id: "ws-1" });
      expect(mocks.post).toHaveBeenCalledTimes(1);
      expect(useProjectWizardStore.getState().repository?.full_name).toBe(picked);
    });

    it("asks which one when several wait and the link names none", async () => {
      mockWaiting(["acme/portal", "acme/site"]);
      renderPage("/acme/wizard/project/repository?project=p-1&add=1");

      expect(await screen.findByPlaceholderText("Search repositories…")).toBeInTheDocument();
      await vi.waitFor(() => expect(mocks.get).toHaveBeenCalledWith("/api/projects/p-1/repos"));
      await vi.waitFor(() => expect(mocks.get).toHaveBeenCalledWith("/api/services", { params: { project_id: "p-1" } }));
      expect(mocks.post).not.toHaveBeenCalled();
      expect(screen.getByRole("heading", { name: "Where's the code?" })).toBeInTheDocument();
    });
  });

  describe("a project whose attached repository was never deployed", () => {
    const portal = { owner: "acme", name: "portal", full_name: "acme/portal", connector_id: "github", role: "app" };
    const setup = { finished: false, steps: { project: "done", people: "done", repository: "done", service: "skipped", reach: "skipped", branches: "skipped" } };

    beforeEach(() => {
      access.areas = ["tickets", "projects", "stacks"];
      mocks.get.mockImplementation(async (url: string) => {
        if (url === "/api/projects/p-1") return { data: { ...project, setup } };
        if (url === "/api/projects/p-1/repos") return { data: [portal] };
        if (url === "/api/machines/deployable") return { data: [{ id: "m-1", name: "prod", stack_root: "/data/nexul", first_seen: "", last_seen: "" }] };
        return { data: [] };
      });
      mocks.post.mockResolvedValue({
        data: { default_branch: "main", env_keys: [], candidates: [{ kind: "compose", path: "compose.yaml", name: "portal", services: [] }] },
      });
      mocks.put.mockResolvedValue({ data: { ...project, setup } });
    });

    it("scans the attached repository when Service opens, with no scan held by this browser", async () => {
      renderPage("/acme/wizard/project/service?project=p-1&revisit=1");

      expect(await screen.findByRole("button", { name: "Create & deploy" })).toBeInTheDocument();
      expect(mocks.post).toHaveBeenCalledWith("/api/repositories/scan", { owner: "acme", name: "portal", workspace_id: "ws-1" });
      expect(screen.queryByText(/scan isn't loaded/)).not.toBeInTheDocument();
    });

    it("folds Service, Domain and Branches under Deploy while it is open, and shows Deploy as later once it is not", async () => {
      renderPage("/acme/wizard/project/deploy?project=p-1&revisit=1");

      await screen.findByRole("radio", { name: /Deploy now/ });
      expect(rungs().map((item) => item.textContent)).toEqual([
        "Namedone", "Peopledone", "Repositorydone", "Deploy", "Serviceskipped", "Domainskipped", "Branchesskipped", "Done",
      ]);
    });

    it("shows Deploy as later on the rail, without the sub-steps, once the wizard is on Done", async () => {
      renderPage("/acme/wizard/project/done?project=p-1&revisit=1");

      await vi.waitFor(() => expect(rungs().map((item) => item.textContent)).toEqual(["Namedone", "Peopledone", "Repositorydone", "Deploylater", "Done"]));
    });

    it("says why the read failed and leaves Go to Repository and Later as the ways on", async () => {
      mocks.post.mockRejectedValue(new Error("offline"));
      const user = userEvent.setup();
      renderPage("/acme/wizard/project/deploy?project=p-1&revisit=1");

      expect(await screen.findByRole("alert")).toBeInTheDocument();
      expect(screen.getByRole("button", { name: "Go to Repository" })).toBeInTheDocument();
      expect(mocks.post).toHaveBeenCalledTimes(1);
      await user.click(screen.getByRole("button", { name: "Later" }));

      expect(await screen.findByRole("heading", { name: "Backend is ready" })).toBeInTheDocument();
    });

    it("offers Deploy now when Deploy opens, and Later lands on Done without a service", async () => {
      const user = userEvent.setup();
      renderPage("/acme/wizard/project/deploy?project=p-1&revisit=1");

      expect(await screen.findByRole("radio", { name: /Deploy now/ })).toBeChecked();
      expect(mocks.post).toHaveBeenCalledTimes(1);
      await user.click(screen.getByRole("radio", { name: /Later/ }));
      await user.click(screen.getByRole("button", { name: "Continue" }));

      expect(await screen.findByRole("heading", { name: "Backend is ready" })).toBeInTheDocument();
      expect(mocks.put).toHaveBeenCalledWith("/api/projects/p-1/setup", {
        finished: undefined,
        steps: { repository: "done", service: "skipped", reach: "skipped", branches: "skipped" },
      });
    });
  });

  describe("a project whose repository already deploys from a service made outside setup", () => {
    const portal = { owner: "acme", name: "portal", full_name: "acme/portal", connector_id: "github", role: "app" };
    const setup = { finished: true, steps: { project: "done", people: "done", repository: "done", service: "skipped", reach: "skipped", branches: "skipped" } };
    const staging = { ...stack, id: "stack-9", name: "portal-staging", build_source: { repo_owner: "acme", repo_name: "portal", branch: "main" } };

    beforeEach(() => {
      access.areas = ["tickets", "projects", "stacks"];
      mocks.get.mockImplementation(async (url: string) => {
        if (url === "/api/projects/p-1") return { data: { ...project, setup } };
        if (url === "/api/projects/p-1/repos") return { data: [portal] };
        if (url === "/api/services") return { data: [staging] };
        return { data: [] };
      });
    });

    it.each(["deploy", "service"])("names that service on %s instead of asking for a repository, and scans nothing", async (step) => {
      const user = userEvent.setup();
      renderPage(`/acme/wizard/project/${step}?project=p-1&revisit=1`);

      expect(await screen.findByRole("link", { name: "portal-staging" })).toHaveAttribute("href", "/acme/stacks/stack-9");
      expect(screen.getByText(/already deploys as/)).toBeInTheDocument();
      expect(screen.queryByText("Pick a repository first.")).not.toBeInTheDocument();
      expect(mocks.post).not.toHaveBeenCalled();
      await user.click(screen.getByRole("button", { name: "Continue" }));

      expect(await screen.findByRole("heading", { name: "Backend is ready" })).toBeInTheDocument();
    });
  });

  it("door 2: Service has no Skip for now, since the door exists to make one, and Back is the way out", async () => {
    const store = useProjectWizardStore.getState();
    store.setProjectId("p-1", "Backend");
    store.setCandidate({ kind: "compose", path: "compose.yml", name: "api", services: [] });
    renderPage("/acme/wizard/project/service?project=p-1&add=1");

    await screen.findByRole("heading", { name: "Where should it run?" });
    expect(screen.getByRole("button", { name: "Back" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Skip for now" })).not.toBeInTheDocument();
    expect(rung("Repository")).toHaveAttribute("data-state", "done");
  });

  it("door 3: preselects the project from ?stack= and opens the repository rung", async () => {
    mocks.get.mockImplementation(async (url: string) => {
      if (url === "/api/stacks/stack-1") return { data: stack };
      if (url === "/api/projects/p-1") return { data: project };
      if (url === "/api/repositories") return { data: { repositories: [] } };
      return { data: [] };
    });
    renderPage("/acme/wizard/project/repository?stack=stack-1");
    expect(await screen.findByRole("heading", { name: "Where's the code?" })).toBeInTheDocument();
    expect(railWord("Attach repository")).toBeInTheDocument();
    expect(rungs().map((item) => item.textContent)).toEqual(["Repository", "Attach", "Branches", "Done"]);
    expect(rung("Repository")).toHaveAttribute("data-state", "current");
  });
});
