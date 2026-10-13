import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { WelcomeProgress, WelcomeStepProgress } from "@nexul/client-core/welcome";

import { getMeKey } from "@/hooks/AuthHooks";
import { welcomeFollower } from "@/hooks/WelcomeHooks";
import { WelcomeWizardPage } from "@/pages/WelcomeWizardPage";
import { useSessionStore } from "@/stores/sessionStore";
import type { MeResponse } from "@/models/User";
import { followFrame } from "@/test/followFrame";

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), toast: { success: vi.fn(), error: vi.fn() } }));

vi.mock("@/api/client", () => ({ api: { get: mocks.get, post: mocks.post }, errorMessage: () => "Something went wrong" }));
vi.mock("sonner", () => ({ toast: mocks.toast }));

const me: MeResponse = {
  user: { id: "u-alice", login: "alice", name: "Alice", avatar_url: "", first_login_done: false, created_at: "" },
  needs_owner_wizard: false,
  needs_first_login_wizard: true,
  instance_permissions: [],
  instance_owner: false,
};

const step = (key: WelcomeStepProgress["key"], state: WelcomeStepProgress["state"]): WelcomeStepProgress => ({ key, state, given: false });

let progress: WelcomeProgress;

const project = { id: "p1", name: "Northwind storefront", prefix: "NW", position: 0, workspace_id: "w1", icon: "", setup: { finished: true, steps: {} } };

beforeEach(() => {
  mocks.get.mockReset();
  mocks.post.mockReset();
  useSessionStore.setState({ token: "t", isLoggedIn: true });
  progress = { first_run_done: true, steps: [], dot: false };
  mocks.get.mockImplementation(async (url: string) => {
    if (url === "/api/me/welcome") return { data: progress };
    if (url === "/api/auth/me") return { data: me };
    if (url === "/api/auth/github-link") return { data: { state: "connected", login: "alice-gh" } };
    if (url === "/api/workspaces") return { data: [{ id: "w1", name: "Northwind", slug: "northwind" }] };
    if (url === "/api/projects") return { data: [project] };
    if (url === "/api/workspaces/w1/me") return { data: { role_name: "Member", permissions: [], projects: [{ project_id: "p1", actions: [] }] } };
    if (url === "/api/permissions/catalog") return { data: { permissions: [] } };
    throw new Error(`unexpected GET ${url}`);
  });
});

const renderWizard = (path = "/wizard/welcome") => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  client.setQueryData([getMeKey], me);
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[path]}>
        <Routes>
          <Route path="/wizard/welcome" element={<WelcomeWizardPage />} />
          <Route path="/" element={<div>home-page</div>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
  return client;
};

const railSteps = async () => {
  const rail = await screen.findByRole("navigation", { name: "Welcome steps" });
  return within(rail)
    .getAllByRole("button")
    .map((button) => button.textContent);
};

describe("WelcomeWizardPage", () => {
  it("lists Profile, then only the steps the person's roles give them", async () => {
    progress = { first_run_done: false, steps: [step("github", "done"), step("phone", "left")], dot: true };
    renderWizard();
    expect(await screen.findByRole("heading", { name: "Introduce yourself" })).toBeInTheDocument();
    expect(await railSteps()).toEqual(["Profile", "Connect GitHubDone", "Get the phone app"]);
  });

  it("opens a later visit on the first step not done", async () => {
    progress = { first_run_done: true, steps: [step("github", "done"), step("phone", "skipped")], dot: false };
    renderWizard();
    expect(await screen.findByRole("heading", { name: "Get the phone app" })).toBeInTheDocument();
  });

  it("shows what a step done already found, with only Continue", async () => {
    progress = { first_run_done: true, steps: [step("github", "done")], dot: false };
    renderWizard("/wizard/welcome?step=github");
    expect(await screen.findByText("Connected as @alice-gh")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Skip for now" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Continue" })).toBeEnabled();
  });

  it("skips a step through its route and moves on", async () => {
    progress = { first_run_done: true, steps: [step("phone", "left")], dot: true };
    mocks.post.mockResolvedValue({ data: { ...progress, steps: [step("phone", "skipped")], dot: false } });
    renderWizard("/wizard/welcome?step=phone");
    await userEvent.click(await screen.findByRole("button", { name: "Skip for now" }));
    expect(mocks.post).toHaveBeenCalledWith("/api/me/welcome/phone/skip");
    expect(await screen.findByRole("heading", { name: "One thing left for later" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Open Get the phone app" })).toBeInTheDocument();
  });

  it("sets the first run when the person finishes later, then leaves", async () => {
    progress = { first_run_done: false, steps: [step("phone", "left")], dot: true };
    mocks.post.mockResolvedValue({ data: { ...progress, first_run_done: true } });
    renderWizard();
    await userEvent.click(await screen.findByRole("button", { name: "Finish later" }));
    expect(mocks.post).toHaveBeenCalledWith("/api/me/welcome/first-run");
    expect(await screen.findByText("home-page")).toBeInTheDocument();
  });

  it.each([
    { key: "phone" as const, waiting: "Waiting for the app to scan the code", settled: "Your phone signed in" },
    { key: "projects" as const, waiting: "Your team hasn't added you to a project yet", settled: "Northwind storefront" },
  ])("settles the $key step live when its progress changes", async ({ key, waiting, settled }) => {
    progress = { first_run_done: true, steps: [step(key, "left")], dot: true };
    const client = renderWizard(`/wizard/welcome?step=${key}`);
    expect(await screen.findByText(waiting)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Continue" })?.hasAttribute("disabled") ?? true).toBe(true);

    progress = { first_run_done: true, steps: [step(key, "done")], dot: false };
    await followFrame(welcomeFollower, "welcome.progress_changed", { user_id: "u-alice" }, client);
    expect(await screen.findByText(settled)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Continue" })).toBeEnabled();
  });
});
