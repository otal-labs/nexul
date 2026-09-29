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

  it("carries the typed values into Paste, and edits made there back into the fields plus extra keys that keep being saved", async () => {
    mocks.patch.mockResolvedValue({ data: stack });
    const onDone = renderStep();
    const user = userEvent.setup();

    await user.type(await screen.findByLabelText("API_KEY"), "secret");
    await user.click(screen.getByRole("radio", { name: "Paste .env" }));
    const paste = screen.getByLabelText("Environment file");
    expect(paste).toHaveValue("API_KEY=secret\nDB_URL=");

    await user.clear(paste);
    await user.click(paste);
    await user.paste("# db\nexport DB_URL='postgres://x'\nEXTRA=1\n");
    await user.click(screen.getByRole("radio", { name: "Fields" }));

    expect(screen.getByLabelText("API_KEY")).toHaveValue("");
    expect(screen.getByLabelText("DB_URL")).toHaveValue("postgres://x");
    expect(screen.getByText("Also saving: EXTRA")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /save & deploy/i }));
    await vi.waitFor(() => expect(onDone).toHaveBeenCalled());
    expect(mocks.patch).toHaveBeenCalledWith(
      "/api/stacks/stack-1",
      expect.objectContaining({ env: { DB_URL: "postgres://x", EXTRA: "1" } }),
    );
  });

  it("saves what the paste box holds when Save is pressed there", async () => {
    mocks.patch.mockResolvedValue({ data: stack });
    const onDone = renderStep();
    const user = userEvent.setup();

    await user.click(await screen.findByRole("radio", { name: "Paste .env" }));
    const paste = screen.getByLabelText("Environment file");
    await user.clear(paste);
    await user.click(paste);
    await user.paste("API_KEY=from-paste");
    await user.click(screen.getByRole("button", { name: /save & deploy/i }));

    await vi.waitFor(() => expect(onDone).toHaveBeenCalled());
    expect(mocks.patch).toHaveBeenCalledWith(
      "/api/stacks/stack-1",
      expect.objectContaining({ env: { API_KEY: "from-paste" } }),
    );
  });

  it("blocks Save and leaving Paste while a line is not KEY=value, and names the line", async () => {
    const user = userEvent.setup();
    renderStep();

    await user.click(await screen.findByRole("radio", { name: "Paste .env" }));
    const paste = screen.getByLabelText("Environment file");
    await user.clear(paste);
    await user.click(paste);
    await user.paste("API_KEY=1\noops");

    expect(screen.getByText("Line 2 isn't KEY=value")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /save & deploy/i })).toBeDisabled();
    expect(screen.getByRole("radio", { name: "Fields" })).toBeDisabled();
    expect(mocks.patch).not.toHaveBeenCalled();
  });
});
