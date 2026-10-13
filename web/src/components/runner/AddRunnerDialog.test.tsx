import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AddRunnerDialog } from "@/components/runner/AddRunnerDialog";
import { runnerFollower } from "@/hooks/RunnerHooks";
import type { Runner } from "@/models/Runner";
import { followFrame } from "@/test/followFrame";

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn() }));

vi.mock("@/api/client", () => ({ api: { get: mocks.get, post: mocks.post }, errorMessage: () => "Something went wrong" }));

const enrollment = {
  code: "nxe_abc",
  expires_at: "2026-09-27T13:00:00Z",
  commands: {
    unix: "curl -fsSL https://nexul.io/runner.sh | sh -s -- --server https://nexul.example.com --name build-box --code nxe_abc",
    windows: "& ([scriptblock]::Create((irm https://nexul.io/runner.ps1))) --server https://nexul.example.com --name build-box --code nxe_abc",
  },
};

const runner = (name: string, over: Partial<Runner> = {}): Runner => ({
  id: name,
  name,
  connected: true,
  last_seen: "2026-09-27T12:00:00Z",
  running_job: null,
  version: "v0.3.0",
  ...over,
});

const renderDialog = (machineName?: string) => {
  const client = new QueryClient();
  render(
    <QueryClientProvider client={client}>
      {machineName && <AddRunnerDialog machineName={machineName} />}
      {!machineName && <AddRunnerDialog />}
    </QueryClientProvider>,
  );
  return client;
};

describe("AddRunnerDialog", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.get.mockResolvedValue({ data: [] });
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

  it("asks for a name only and shows both one-liners exactly as the server made them", async () => {
    const user = userEvent.setup();
    renderDialog();
    await user.click(screen.getByRole("button", { name: "Add runner" }));

    expect(screen.queryByLabelText("Machine")).not.toBeInTheDocument();
    expect(screen.queryByLabelText("GitHub token")).not.toBeInTheDocument();
    await user.type(screen.getByLabelText("Runner name"), "build-box");
    await user.click(screen.getByRole("button", { name: "Create install command" }));

    expect(mocks.post).toHaveBeenCalledWith("/api/runners/enrollments", { name: "build-box", machine: undefined });
    expect(await screen.findByText(enrollment.commands.unix)).toBeInTheDocument();
    expect(screen.getByText(enrollment.commands.windows)).toBeInTheDocument();
  });

  it("adds to a known machine without asking for it", async () => {
    const user = userEvent.setup();
    renderDialog("prod");
    await user.click(screen.getByRole("button", { name: "Add a runner to this machine" }));

    expect(screen.queryByLabelText("Machine")).not.toBeInTheDocument();
    await user.type(screen.getByLabelText("Runner name"), "build-box");
    await user.click(screen.getByRole("button", { name: "Create install command" }));

    expect(mocks.post).toHaveBeenCalledWith("/api/runners/enrollments", { name: "build-box", machine: "prod" });
    expect(await screen.findByText(enrollment.commands.unix)).toBeInTheDocument();
  });

  it("waits for a runner under the typed name, then shows its name and machine once it connects", async () => {
    mocks.get.mockResolvedValue({ data: [runner("other-box", { machine: "elsewhere" })] });
    const user = userEvent.setup();
    const client = renderDialog();
    await user.click(screen.getByRole("button", { name: "Add runner" }));
    await user.type(screen.getByLabelText("Runner name"), "build-box");
    await user.click(screen.getByRole("button", { name: "Create install command" }));
    await screen.findByText(enrollment.commands.unix);

    const row = screen.getByRole("status");
    expect(await within(row).findByText("Waiting")).toBeInTheDocument();
    expect(within(row).getByText("Server connected")).toBeInTheDocument();

    mocks.get.mockResolvedValue({ data: [runner("other-box"), runner("build-box", { machine: "prod" })] });
    await act(() => followFrame(runnerFollower, "runner.connected", { name: "build-box" }, client));

    expect(await within(row).findByText("Connected")).toBeInTheDocument();
    expect(within(row).getByText("build-box · prod")).toBeInTheDocument();
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
