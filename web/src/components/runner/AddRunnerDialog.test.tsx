import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AddRunnerDialog } from "@/components/runner/AddRunnerDialog";

const mocks = vi.hoisted(() => ({ post: vi.fn() }));

vi.mock("@/api/client", () => ({ api: { post: mocks.post }, errorMessage: () => "Something went wrong" }));

const enrollment = {
  code: "nxe_abc",
  expires_at: "2026-09-27T13:00:00Z",
  commands: {
    unix: "curl -fsSL https://nexul.io/runner.sh | sh -s -- --server https://nexul.example.com --name build-box --code nxe_abc",
    windows: "& ([scriptblock]::Create((irm https://nexul.io/runner.ps1))) --server https://nexul.example.com --name build-box --code nxe_abc",
  },
};

const renderDialog = (machineName?: string) =>
  render(
    <QueryClientProvider client={new QueryClient()}>
      {machineName && <AddRunnerDialog machineName={machineName} />}
      {!machineName && <AddRunnerDialog />}
    </QueryClientProvider>,
  );

describe("AddRunnerDialog", () => {
  beforeEach(() => {
    mocks.post.mockReset();
    mocks.post.mockResolvedValue({ data: enrollment });
  });

  it("rejects a name the installer cannot use, without asking the server", async () => {
    const user = userEvent.setup();
    renderDialog();
    await user.click(screen.getByRole("button", { name: "Add runner" }));

    await user.type(screen.getByLabelText("Runner name"), "Build Box");
    await user.click(screen.getByRole("button", { name: "Create install command" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("lowercase letters, digits or dashes");
    expect(mocks.post).not.toHaveBeenCalled();
  });

  it("keeps the form when the server refuses, so the name can be changed", async () => {
    mocks.post.mockRejectedValue(new Error("conflict"));
    const user = userEvent.setup();
    renderDialog();
    await user.click(screen.getByRole("button", { name: "Add runner" }));

    await user.type(screen.getByLabelText("Runner name"), "build-box");
    await user.click(screen.getByRole("button", { name: "Create install command" }));

    expect(await screen.findByRole("button", { name: "Create install command" })).toBeEnabled();
    expect(screen.getByLabelText("Runner name")).toHaveValue("build-box");
  });

  it("enrolls the runner and shows both one-liners, carrying the git token", async () => {
    const user = userEvent.setup();
    renderDialog();
    await user.click(screen.getByRole("button", { name: "Add runner" }));

    await user.type(screen.getByLabelText("Runner name"), "build-box");
    await user.type(screen.getByLabelText("GitHub token"), "ghp_abc");
    await user.click(screen.getByRole("button", { name: "Create install command" }));

    expect(mocks.post).toHaveBeenCalledWith("/api/runners/enrollments", { name: "build-box", machine: undefined });
    expect(await screen.findByText(`${enrollment.commands.unix} --git-token 'ghp_abc'`)).toBeInTheDocument();
    expect(screen.getByText(`${enrollment.commands.windows} --git-token 'ghp_abc'`)).toBeInTheDocument();
  });

  it("adds to a known machine without asking for it", async () => {
    const user = userEvent.setup();
    renderDialog("prod");
    await user.click(screen.getByRole("button", { name: "Add a runner to this machine" }));

    expect(screen.getByLabelText("Machine")).toHaveValue("prod");
    await user.type(screen.getByLabelText("Runner name"), "build-box");
    await user.click(screen.getByRole("button", { name: "Create install command" }));

    expect(mocks.post).toHaveBeenCalledWith("/api/runners/enrollments", { name: "build-box", machine: "prod" });
    expect(await screen.findByText(enrollment.commands.unix)).toBeInTheDocument();
  });

  it("copies a command to the clipboard", async () => {
    const user = userEvent.setup();
    // userEvent.setup() installs its own navigator.clipboard stub, so the
    // spy must be attached after setup() runs or it gets clobbered.
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", {
      value: { writeText },
      configurable: true,
      writable: true,
    });
    renderDialog();
    await user.click(screen.getByRole("button", { name: "Add runner" }));
    await user.type(screen.getByLabelText("Runner name"), "build-box");
    await user.click(screen.getByRole("button", { name: "Create install command" }));
    await screen.findByText(enrollment.commands.windows);

    await user.click(screen.getByRole("button", { name: "Copy the Windows command" }));

    expect(writeText).toHaveBeenCalledWith(enrollment.commands.windows);
  });
});
