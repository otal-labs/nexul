import { describe, expect, it } from "vitest";

import { detectTunnelOs, TunnelOs, tunnelInstallSteps } from "@/utils/TunnelInstallCommands";

describe("tunnelInstallSteps", () => {
  it.each([
    [TunnelOs.Linux, "bash", "sudo cloudflared service install tok"],
    [TunnelOs.MacOS, "zsh", "cloudflared service install tok"],
    [TunnelOs.Windows, "powershell", "cloudflared.exe service install tok"],
  ])("ends %s's steps by installing the service with the computer's token", (os, shell, last) => {
    const steps = tunnelInstallSteps(os, "tok");
    expect(steps).toHaveLength(2);
    expect(steps.every((s) => s.shell === shell)).toBe(true);
    expect(steps.at(-1)?.lines).toEqual([last]);
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
