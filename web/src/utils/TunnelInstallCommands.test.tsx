import { describe, expect, it } from "vitest";

import { detectTunnelOs, TunnelOs } from "@/utils/TunnelInstallCommands";

describe("detectTunnelOs", () => {
  it.each([
    ["Mozilla/5.0 (Windows NT 10.0; Win64; x64)", TunnelOs.Windows],
    ["Mozilla/5.0 (Macintosh; Intel Mac OS X 14_5)", TunnelOs.Unix],
    ["Mozilla/5.0 (X11; Linux x86_64)", TunnelOs.Unix],
    ["", TunnelOs.Unix],
  ])("reads %j as %s", (ua, os) => {
    expect(detectTunnelOs(ua)).toBe(os);
  });
});
