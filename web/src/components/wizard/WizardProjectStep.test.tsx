import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { WizardProjectStep } from "@/components/wizard/WizardProjectStep";

const mocks = vi.hoisted(() => ({ post: vi.fn(), errorMessage: vi.fn(() => "") }));

vi.mock("@/api/client", () => ({ api: { post: mocks.post }, errorMessage: mocks.errorMessage }));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const renderStep = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <WizardProjectStep onDone={vi.fn()} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  mocks.post.mockReset();
  mocks.post.mockResolvedValue({ data: { id: "p-1", name: "Phase 1", prefix: "P1" } });
});

describe("WizardProjectStep", () => {
  it("uppercases the prefix as it is typed and creates a project whose prefix has a digit", async () => {
    const user = userEvent.setup();
    renderStep();
    await user.type(screen.getByLabelText("Project name"), "Phase 1");
    await user.type(screen.getByLabelText("Prefix"), "p1");
    expect(screen.getByLabelText("Prefix")).toHaveValue("P1");
    await user.click(screen.getByRole("button", { name: "Continue" }));
    expect(mocks.post).toHaveBeenCalledWith("/api/projects", expect.objectContaining({ name: "Phase 1", prefix: "P1" }));
  });

  it("refuses a prefix that starts with a digit and says what is allowed", async () => {
    const user = userEvent.setup();
    renderStep();
    await user.type(screen.getByLabelText("Project name"), "Phase 1");
    await user.type(screen.getByLabelText("Prefix"), "1p");
    await user.click(screen.getByRole("button", { name: "Continue" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Prefix is 2–5 letters or digits, starting with a letter");
    expect(mocks.post).not.toHaveBeenCalled();
  });
});
