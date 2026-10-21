// @vitest-environment node
import { describe, expect, it } from "vitest";

import { detectCommandOs, CommandOs } from "@/utils/InstallCommands";

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
