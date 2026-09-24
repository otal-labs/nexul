import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { StackBranchDeploySection } from "@/components/stack/StackBranchDeploySection";
import type { StackWithBranches } from "@/models/Stack";

const mocks = vi.hoisted(() => ({ patch: vi.fn(), _delete: vi.fn() }));

vi.mock("@/api/client", () => ({
  api: { patch: mocks.patch, delete: mocks._delete },
  errorMessage: () => "boom",
}));

const stack: StackWithBranches = {
  id: "stack-1",
  project_id: "proj-1",
  name: "api",
  slug: "api",
  machine: "instance",
  strategy: "compose",
  managed: true,
  created_at: "2026-08-12T09:00:00Z",
  updated_at: "2026-08-12T09:00:00Z",
  branch_deploy_rules: [
    { pattern: "main", docker_network: "prod-net" },
    { pattern: "feature/*", docker_network: "qa-net", overrides: { DATABASE_URL: "postgres://qa", REDIS_URL: "redis://qa" } },
  ],
};

const renderSection = (s: StackWithBranches = stack) =>
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <StackBranchDeploySection stack={s} />
    </QueryClientProvider>,
  );

const patchedRules = () => mocks.patch.mock.calls[0]?.[1].branch_deploy_rules;

describe("StackBranchDeploySection overrides", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.patch.mockImplementation((_url: string, body: unknown) => Promise.resolve({ data: body }));
  });

  it("lists override keys on the rule row, never their values", () => {
    renderSection();
    expect(screen.getByText("overrides DATABASE_URL, REDIS_URL")).toBeInTheDocument();
    expect(screen.queryByText(/postgres:\/\/qa/)).not.toBeInTheDocument();
  });

  it("offers overrides only on a rule that deploys its own copy", () => {
    renderSection();
    expect(screen.getAllByRole("button", { name: "Overrides" })).toHaveLength(1);
  });

  it("removing an override line saves the rule without it", async () => {
    const user = userEvent.setup();
    renderSection();
    await user.click(screen.getByRole("button", { name: "Overrides" }));
    const field = screen.getByRole("textbox", { name: "Overrides" });
    expect(field).toHaveValue("DATABASE_URL=postgres://qa\nREDIS_URL=redis://qa");
    await user.clear(field);
    await user.type(field, "DATABASE_URL=postgres://staging");
    await user.click(screen.getByRole("button", { name: "Save overrides" }));

    expect(mocks.patch).toHaveBeenCalledWith("/api/stacks/stack-1", expect.anything());
    expect(patchedRules()).toEqual([
      { pattern: "main", docker_network: "prod-net" },
      { pattern: "feature/*", docker_network: "qa-net", overrides: { DATABASE_URL: "postgres://staging" } },
    ]);
  });

  it("clearing every override drops the overrides from the rule", async () => {
    const user = userEvent.setup();
    renderSection();
    await user.click(screen.getByRole("button", { name: "Overrides" }));
    await user.clear(screen.getByRole("textbox", { name: "Overrides" }));
    await user.click(screen.getByRole("button", { name: "Save overrides" }));

    expect(patchedRules()?.[1]).toEqual({ pattern: "feature/*", docker_network: "qa-net" });
  });

  it("a new wildcard rule saves its overrides; an in-place one never shows the field", async () => {
    const user = userEvent.setup();
    renderSection({ ...stack, branch_deploy_rules: [] });
    await user.click(screen.getByRole("button", { name: "Add rule" }));
    await user.type(screen.getByLabelText("Branch pattern"), "main");
    expect(screen.queryByLabelText("Overrides (optional)")).not.toBeInTheDocument();

    await user.clear(screen.getByLabelText("Branch pattern"));
    await user.type(screen.getByLabelText("Branch pattern"), "feature/*");
    await user.type(screen.getByLabelText("Docker network"), "qa-net");
    await user.type(screen.getByLabelText("Overrides (optional)"), "DATABASE_URL=postgres://qa");
    await user.click(screen.getByRole("button", { name: "Save rule" }));

    expect(patchedRules()).toEqual([
      { pattern: "feature/*", docker_network: "qa-net", overrides: { DATABASE_URL: "postgres://qa" } },
    ]);
  });
});
