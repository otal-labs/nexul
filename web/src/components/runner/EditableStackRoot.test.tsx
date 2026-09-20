import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { EditableStackRoot } from "@/components/runner/EditableStackRoot";

const mocks = vi.hoisted(() => ({ patch: vi.fn(), toast: { success: vi.fn(), error: vi.fn() } }));

vi.mock("@/api/client", () => ({ api: { patch: mocks.patch }, errorMessage: () => "boom" }));
vi.mock("sonner", () => ({ toast: mocks.toast }));

const renderField = () =>
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <EditableStackRoot machineId="m-1" stackRoot="/data/nexul" />
    </QueryClientProvider>,
  );

describe("EditableStackRoot", () => {
  beforeEach(() => {
    mocks.patch.mockReset();
    mocks.patch.mockResolvedValue({ data: { ID: "m-1", Name: "prod", StackRoot: "/srv/data" } });
  });

  it("shows the current stack root as plain text until clicked", () => {
    renderField();
    expect(screen.getByText("/data/nexul")).toBeInTheDocument();
    expect(screen.queryByLabelText("Stack root")).not.toBeInTheDocument();
  });

  it("saves a changed value on blur", async () => {
    const user = userEvent.setup();
    renderField();

    await user.click(screen.getByText("/data/nexul"));
    const input = screen.getByLabelText("Stack root");
    await user.clear(input);
    await user.type(input, "/srv/data");
    await user.tab();

    expect(mocks.patch).toHaveBeenCalledWith("/api/machines/m-1", { stack_root: "/srv/data" });
  });

  it("cancels without saving", async () => {
    const user = userEvent.setup();
    renderField();

    await user.click(screen.getByText("/data/nexul"));
    await user.type(screen.getByLabelText("Stack root"), "garbage");
    await user.click(screen.getByRole("button", { name: "Cancel" }));

    expect(mocks.patch).not.toHaveBeenCalled();
    expect(screen.getByText("/data/nexul")).toBeInTheDocument();
  });
});
