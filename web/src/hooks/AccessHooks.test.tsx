import { describe, expect, it } from "vitest";

import { workspaceAccess } from "@/hooks/AccessHooks";

describe("workspaceAccess", () => {
  it("opens the Interview template in Configuration to whoever may edit it, with no project's memories:read", () => {
    const editor = workspaceAccess([], ["memories:template"]);
    expect(editor.configurationSections).toContain("interview");
    expect(editor.canOpen("configuration")).toBe(true);
    expect(workspaceAccess([], []).configurationSections).not.toContain("interview");
  });
});
