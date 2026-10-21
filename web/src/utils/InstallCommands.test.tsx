// @vitest-environment node
import { describe, expect, it } from "vitest";

import { computerCommands, detectCommandOs, CommandOs, foregroundCommand } from "@/utils/InstallCommands";

describe("detectCommandOs", () => {
  it.each([
    ["Mozilla/5.0 (Windows NT 10.0; Win64; x64)", CommandOs.Windows],
    ["Mozilla/5.0 (Macintosh; Intel Mac OS X 14_5)", CommandOs.Unix],
    ["Mozilla/5.0 (X11; Linux x86_64)", CommandOs.Unix],
    ["", CommandOs.Unix],
  ])("reads %j as %s", (ua, os) => {
    expect(detectCommandOs(ua)).toBe(os);
  });
});

describe("foregroundCommand", () => {
  it("is the same command with no sudo and --foreground at the end", () => {
    expect(foregroundCommand("curl -fsSL https://nexul.io/computer.sh | sudo sh -s -- eyJ.token.sig")).toBe(
      "curl -fsSL https://nexul.io/computer.sh | sh -s -- eyJ.token.sig --foreground",
    );
  });

  it("is offered on macOS / Linux only", () => {
    const commands = computerCommands({ unix: "curl -fsSL https://nexul.io/computer.sh | sudo sh -s -- t", windows: "ps" });
    expect(commands[CommandOs.Unix].alternative?.lines).toEqual(["curl -fsSL https://nexul.io/computer.sh | sh -s -- t --foreground"]);
    expect(commands[CommandOs.Windows].alternative).toBeUndefined();
  });
});
