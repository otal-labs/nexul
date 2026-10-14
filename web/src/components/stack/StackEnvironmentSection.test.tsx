import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { StackEnvironmentSection } from "@/components/stack/StackEnvironmentSection";
import { DeployStrategy, type Stack } from "@/models/Stack";

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), patch: vi.fn(), areas: ["editStacks", "editDeploys"] as string[] }));

vi.mock("@/api/client", () => ({ api: { get: mocks.get, post: mocks.post, patch: mocks.patch }, errorMessage: () => "boom" }));
vi.mock("@/hooks/AccessHooks", () => ({ useAreaAccess: () => (area: string) => mocks.areas.includes(area) }));

const stack: Stack = {
  id: "stack-1",
  project_id: "proj-1",
  name: "api",
  slug: "api",
  machine: "instance",
  strategy: DeployStrategy.Compose,
  build_source: { repo_owner: "acme", repo_name: "api", branch: "main" },
  env: { DATABASE_URL: "postgres://acme:hunter2@db.example.com/app", LOG_LEVEL: "info", OLD_FLAG: "1" },
  managed: true,
  created_at: "2026-10-01T09:00:00Z",
  updated_at: "2026-10-01T09:00:00Z",
};

const renderSection = (value: Stack = stack) =>
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <StackEnvironmentSection stack={value} image={undefined} />
      </MemoryRouter>
    </QueryClientProvider>,
  );

describe("StackEnvironmentSection", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.areas = ["editStacks", "editDeploys"];
    mocks.patch.mockResolvedValue({ data: stack });
  });

  it.each([
    ["a role that cannot change stacks", { env: { DATABASE_URL: "", LOG_LEVEL: "" }, env_masked: true }, []],
    ["a stack rule that took the change away from a role that holds it", { env: { DATABASE_URL: "", LOG_LEVEL: "" }, env_masked: true }, ["editStacks"]],
    ["a branch deployment", { derived_from: "stack-0" }, ["editStacks"]],
  ])("lists only the names for %s", (_name, change, areas) => {
    mocks.areas = areas;
    renderSection({ ...stack, ...change });

    expect(screen.getByText("DATABASE_URL")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Add variable" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Reveal values/ })).not.toBeInTheDocument();
    expect(screen.queryByDisplayValue(/hunter2/)).not.toBeInTheDocument();
  });

  it("keeps the values hidden until revealed, then saves a removed key and a blanked value", async () => {
    const user = userEvent.setup();
    renderSection();

    const value = screen.getByLabelText("Value of DATABASE_URL");
    expect(value).toHaveAttribute("type", "password");
    await user.click(screen.getByRole("button", { name: "Reveal values" }));
    expect(value).toHaveAttribute("type", "text");
    expect(value).toHaveValue("postgres://acme:hunter2@db.example.com/app");

    await user.click(screen.getByRole("button", { name: "Remove OLD_FLAG" }));
    await user.clear(screen.getByLabelText("Value of LOG_LEVEL"));
    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(mocks.patch).toHaveBeenCalledWith("/api/stacks/stack-1", {
      ...stack,
      env: { DATABASE_URL: "postgres://acme:hunter2@db.example.com/app", LOG_LEVEL: "" },
    });
  });

  it("adds a variable and refuses a name that is already set", async () => {
    const user = userEvent.setup();
    renderSection();

    await user.click(screen.getByRole("button", { name: "Add variable" }));
    await user.type(screen.getByLabelText("Name of new variable"), "LOG_LEVEL");
    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(await screen.findByText("Already set above.")).toBeInTheDocument();
    expect(mocks.patch).not.toHaveBeenCalled();
  });

  it("deploys the branch once the changes are saved, and not before", async () => {
    mocks.post.mockResolvedValue({ data: { id: "d-1", stack_id: "stack-1" } });
    const user = userEvent.setup();
    renderSection();

    expect(screen.getByRole("button", { name: "Deploy now" })).toBeEnabled();
    await user.type(screen.getByLabelText("Value of LOG_LEVEL"), "x");
    expect(screen.getByRole("button", { name: "Deploy now" })).toBeDisabled();
    await user.click(screen.getByRole("button", { name: "Discard" }));
    await user.click(screen.getByRole("button", { name: "Deploy now" }));

    expect(mocks.post).toHaveBeenCalledWith("/api/deploys", { stack_id: "stack-1", ref: "main" });
  });
});
