import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AddRunnerDialog } from "@/components/runner/AddRunnerDialog";

const mocks = vi.hoisted(() => ({ get: vi.fn() }));

vi.mock("@/api/client", () => ({ api: { get: mocks.get } }));

const install = {
  ws_url: "wss://deploy.example.com:8081/ws/runner",
  secret: "abc123def456",
  download_url: "https://deploy.example.com/api/runners/download",
};

const renderDialog = () => {
  const client = new QueryClient();
  return render(
    <QueryClientProvider client={client}>
      <AddRunnerDialog />
    </QueryClientProvider>,
  );
};

describe("AddRunnerDialog", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.get.mockResolvedValue({ data: install });
  });

  it("does not fetch install details until opened", () => {
    renderDialog();
    expect(mocks.get).not.toHaveBeenCalled();
  });

  it("shows the install command with the real ws_url and secret once opened", async () => {
    const user = userEvent.setup();
    renderDialog();

    await user.click(screen.getByRole("button", { name: "Add runner" }));

    expect(mocks.get).toHaveBeenCalledWith("/api/runners/install");
    const pre = await screen.findByText(/curl -fsSL/);
    expect(pre.textContent).toContain(install.ws_url);
    expect(pre.textContent).toContain(install.secret);
    expect(pre.textContent).toContain(install.download_url);
    expect(pre.textContent).toContain("<github-token>");
  });

  it("switches to the Windows PowerShell command", async () => {
    const user = userEvent.setup();
    renderDialog();
    await user.click(screen.getByRole("button", { name: "Add runner" }));
    await screen.findByText(/curl -fsSL/);

    await user.click(screen.getByRole("button", { name: "Windows" }));

    const pre = await screen.findByText(/Invoke-WebRequest/);
    expect(pre.textContent).toContain(install.ws_url);
  });

  it("replaces the placeholder once a GitHub token is typed", async () => {
    const user = userEvent.setup();
    renderDialog();
    await user.click(screen.getByRole("button", { name: "Add runner" }));
    await screen.findByText(/curl -fsSL/);

    await user.type(screen.getByLabelText("GitHub token"), "ghp_abc");

    const pre = await screen.findByText(/curl -fsSL/);
    expect(pre.textContent).toContain('NEXUL_GIT_TOKEN="ghp_abc"');
    expect(pre.textContent).not.toContain("<github-token>");
  });

  it("copies the command to the clipboard", async () => {
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
    const pre = await screen.findByText(/curl -fsSL/);
    const command = pre.textContent;

    await user.click(screen.getByRole("button", { name: "Copy" }));

    expect(writeText).toHaveBeenCalledWith(command);
  });
});
