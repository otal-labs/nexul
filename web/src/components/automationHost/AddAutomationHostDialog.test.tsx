import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AddAutomationHostDialog } from "@/components/automationHost/AddAutomationHostDialog";

const mocks = vi.hoisted(() => ({ post: vi.fn() }));

vi.mock("@/api/client", () => ({ api: { post: mocks.post }, errorMessage: () => "Something went wrong" }));

const enrollment = {
  code: "nxe_abc",
  expires_at: "2026-09-27T13:00:00Z",
  commands: {
    unix: "curl -fsSL https://nexul.io/automations.sh | sh -s -- --server https://nexul.example.com --name jobs-1 --code nxe_abc",
    windows: "& ([scriptblock]::Create((irm https://nexul.io/automations.ps1))) --server https://nexul.example.com --name jobs-1 --code nxe_abc",
  },
};

const renderDialog = () =>
  render(
    <QueryClientProvider client={new QueryClient()}>
      <AddAutomationHostDialog />
    </QueryClientProvider>,
  );

describe("AddAutomationHostDialog", () => {
  beforeEach(() => {
    mocks.post.mockReset();
    mocks.post.mockResolvedValue({ data: enrollment });
  });

  it("rejects a name the installer cannot use, without asking the server", async () => {
    const user = userEvent.setup();
    renderDialog();
    await user.click(screen.getByRole("button", { name: "Add automations host" }));

    await user.type(screen.getByLabelText("Host name"), "Jobs One");
    await user.click(screen.getByRole("button", { name: "Create install command" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("lowercase letters, digits or dashes");
    expect(mocks.post).not.toHaveBeenCalled();
  });

  it("keeps the form when the server refuses, so the name can be changed", async () => {
    mocks.post.mockRejectedValue(new Error("conflict"));
    const user = userEvent.setup();
    renderDialog();
    await user.click(screen.getByRole("button", { name: "Add automations host" }));

    await user.type(screen.getByLabelText("Host name"), "jobs-1");
    await user.click(screen.getByRole("button", { name: "Create install command" }));

    expect(await screen.findByRole("button", { name: "Create install command" })).toBeEnabled();
    expect(screen.getByLabelText("Host name")).toHaveValue("jobs-1");
  });

  it("enrolls the host and shows both one-liners", async () => {
    const user = userEvent.setup();
    renderDialog();
    await user.click(screen.getByRole("button", { name: "Add automations host" }));

    await user.type(screen.getByLabelText("Host name"), "jobs-1");
    await user.type(screen.getByLabelText("Machine"), "prod");
    await user.click(screen.getByRole("button", { name: "Create install command" }));

    expect(mocks.post).toHaveBeenCalledWith("/api/automation-hosts/enrollments", { name: "jobs-1", machine: "prod" });
    expect(await screen.findByText(enrollment.commands.unix)).toBeInTheDocument();
    expect(screen.getByText(enrollment.commands.windows)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Copy the Windows command" })).toBeInTheDocument();
  });
});
