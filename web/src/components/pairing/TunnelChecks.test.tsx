import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { CommandBlock } from "@/components/pairing/CommandBlock";
import { TunnelChecks } from "@/components/pairing/TunnelChecks";
import type { TunnelStatus } from "@/models/Pairing";

const rows = (status: TunnelStatus) => {
  render(<TunnelChecks status={status} />);
  return screen.getAllByRole("listitem").map((li) => li.textContent);
};

describe("TunnelChecks", () => {
  it.each([
    [{ tunnel: "inactive", harness_reachable: false }, /Waiting for cloudflared.*Waiting/, /Checked once the tunnel is online.*Waiting/, "Waiting"],
    [{ tunnel: "down", harness_reachable: false }, /lost its connections.*Down/, /Checked once/, "Waiting"],
    [{ tunnel: "degraded", harness_reachable: false }, /unhealthy.*Degraded/, /Start T3 Code/, "Waiting"],
    [{ tunnel: "healthy", harness_reachable: true, harness_version: "0.0.40" }, /Online/, /T3 Code 0\.0\.40.*Answering/, "Both passed"],
  ])("renders %j", (status, tunnel, harness, summary) => {
    const [tunnelRow, harnessRow] = rows(status as TunnelStatus);
    expect(tunnelRow).toMatch(tunnel);
    expect(harnessRow).toMatch(harness);
    expect(screen.getByRole("status").firstElementChild).toHaveTextContent(`Checks${summary}`);
  });
});

describe("CommandBlock", () => {
  it("copies every line, one per row", async () => {
    const user = userEvent.setup();
    const writeText = vi.spyOn(navigator.clipboard, "writeText").mockResolvedValue();
    render(<CommandBlock shell="bash" lines={["sudo apt-get update", "sudo apt-get install cloudflared"]} />);

    await user.click(screen.getByRole("button", { name: /copy bash commands/i }));
    expect(writeText).toHaveBeenCalledWith("sudo apt-get update\nsudo apt-get install cloudflared");
    expect(await screen.findByRole("button", { name: "Copied" })).toBeInTheDocument();
  });

  it("keeps the lines on screen when the clipboard is unavailable", async () => {
    const user = userEvent.setup();
    vi.spyOn(navigator.clipboard, "writeText").mockRejectedValue(new Error("denied"));
    render(<CommandBlock shell="zsh" lines={["brew install cloudflared"]} />);

    await user.click(screen.getByRole("button", { name: /copy zsh commands/i }));
    expect(screen.getByText("brew install cloudflared")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /copy zsh commands/i })).toBeInTheDocument();
  });
});
