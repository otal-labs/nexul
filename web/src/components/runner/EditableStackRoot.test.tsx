import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { EditableStackRoot } from "@/components/runner/EditableStackRoot";
import { resetViewer, viewer } from "@/test/viewerActions";

const mocks = vi.hoisted(() => ({ patch: vi.fn(), toast: { success: vi.fn(), error: vi.fn() } }));

vi.mock("@/api/client", () => ({ api: { patch: mocks.patch }, errorMessage: () => "boom" }));
vi.mock("sonner", () => ({ toast: mocks.toast }));
vi.mock("@/hooks/useActionState", async () => (await import("@/test/viewerActions")).mockedActionState);

const renderField = (actions = ["machines:see", "machines:write"]) =>
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <EditableStackRoot machine={{ id: "m-1", stack_root: "/data/nexul", actions }} />
    </QueryClientProvider>,
  );

describe("EditableStackRoot", () => {
  beforeEach(() => {
    resetViewer();
    viewer.showLocked = true;
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

  it("saves once when the check button is clicked", async () => {
    const user = userEvent.setup();
    renderField();

    await user.click(screen.getByText("/data/nexul"));
    const input = screen.getByLabelText("Stack root");
    await user.clear(input);
    await user.type(input, "/srv/data");
    await user.click(screen.getByRole("button", { name: "Save stack root" }));

    expect(mocks.patch).toHaveBeenCalledTimes(1);
  });

  it("keeps editing when the save fails", async () => {
    mocks.patch.mockRejectedValue(new Error("Request failed with status code 400"));
    const user = userEvent.setup();
    renderField();

    await user.click(screen.getByText("/data/nexul"));
    const input = screen.getByLabelText("Stack root");
    await user.clear(input);
    await user.type(input, "/srv/data{Enter}");

    expect(await screen.findByLabelText("Stack root")).toHaveValue("/srv/data");
    expect(mocks.toast.error).toHaveBeenCalledWith("boom");
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

  it("keeps the root as text a viewer who can't manage the machine can't edit, locked or not", async () => {
    const user = userEvent.setup();
    const { unmount } = renderField(["machines:see"]);
    const locked = screen.getByRole("button", { name: "Edit stack root /data/nexul" });
    await user.click(locked);
    expect(locked).toHaveAttribute("aria-disabled", "true");
    expect(screen.queryByLabelText("Stack root")).not.toBeInTheDocument();
    unmount();

    viewer.showLocked = false;
    renderField(["machines:see"]);
    expect(screen.getByText("/data/nexul")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Edit stack root/ })).not.toBeInTheDocument();
  });
});
