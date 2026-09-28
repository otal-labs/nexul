import { describe, expect, it } from "vitest";

import { detectTunnelOs, TunnelOs, tunnelInstallSteps } from "@/utils/TunnelInstallCommands";

describe("tunnelInstallSteps", () => {
  it.each([
    [TunnelOs.Linux, "bash", "curl -fsSL https://nexul.io/tunnel.sh | sh -s -- tok"],
    [TunnelOs.MacOS, "zsh", "curl -fsSL https://nexul.io/tunnel.sh | sh -s -- tok"],
    [TunnelOs.Windows, "powershell", "& ([scriptblock]::Create((irm https://nexul.io/tunnel.ps1))) tok"],
  ])("gives %s one command that carries the computer's token", (os, shell, command) => {
    const steps = tunnelInstallSteps(os, "tok");
    expect(steps).toHaveLength(1);
    expect(steps[0]?.shell).toBe(shell);
    expect(steps[0]?.lines).toEqual([command]);
  });
});

describe("detectTunnelOs", () => {
  it.each([
    ["Mozilla/5.0 (Windows NT 10.0; Win64; x64)", TunnelOs.Windows],
    ["Mozilla/5.0 (Macintosh; Intel Mac OS X 14_5)", TunnelOs.MacOS],
    ["Mozilla/5.0 (X11; Linux x86_64)", TunnelOs.Linux],
    ["", TunnelOs.Linux],
  ])("reads %j as %s", (ua, os) => {
    expect(detectTunnelOs(ua)).toBe(os);
  });
});
