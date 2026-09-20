import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { WizardEnvStep } from "@/components/wizard/WizardEnvStep";
import { useProjectWizardStore } from "@/stores/projectWizardStore";

const mocks = vi.hoisted(() => ({ get: vi.fn(), patch: vi.fn(), post: vi.fn(), errorMessage: vi.fn(() => "") }));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, patch: mocks.patch, post: mocks.post },
  errorMessage: mocks.errorMessage,
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const stack = {
  id: "stack-1",
  project_id: "p-1",
  name: "api",
  slug: "api",
  machine: "prod",
  strategy: "compose" as const,
  managed: true,
  created_at: "",
  updated_at: "",
};

const renderStep = (onDone = vi.fn()) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <WizardEnvStep onDone={onDone} />
    </QueryClientProvider>,
  );
  return onDone;
};

beforeEach(() => {
  mocks.get.mockReset();
  mocks.patch.mockReset();
  mocks.post.mockReset();
  mocks.post.mockResolvedValue({ data: { id: "d1", stack_id: "stack-1", status: "pending", kind: "build" } });
  useProjectWizardStore.getState().reset();
  useProjectWizardStore.getState().setStackId("stack-1");
  useProjectWizardStore.getState().setScanResult({ default_branch: "main", candidates: [], env_keys: ["API_KEY", "DB_URL"] });
  mocks.get.mockResolvedValue({ data: stack });
});

describe("WizardEnvStep", () => {
  it("saves only the filled keys onto the stack's env map, leaving blanks out, then starts the first deploy", async () => {
    mocks.patch.mockResolvedValue({ data: { ...stack, env: { API_KEY: "secret" } } });
    const onDone = renderStep();
    const user = userEvent.setup();

    expect(await screen.findByLabelText("API_KEY")).toBeInTheDocument();
    expect(screen.getByLabelText("DB_URL")).toBeInTheDocument();
    await user.type(screen.getByLabelText("API_KEY"), "secret");
    await user.click(screen.getByRole("button", { name: /save & deploy/i }));

    await vi.waitFor(() => expect(onDone).toHaveBeenCalled());
    expect(mocks.patch).toHaveBeenCalledWith(
      "/api/stacks/stack-1",
      expect.objectContaining({ env: { API_KEY: "secret" } }),
    );
    expect(useProjectWizardStore.getState().envValues).toEqual({ API_KEY: "secret" });
    expect(mocks.post).toHaveBeenCalledWith("/api/deploys", expect.objectContaining({ stack_id: "stack-1", ref: "main" }));
  });
});
