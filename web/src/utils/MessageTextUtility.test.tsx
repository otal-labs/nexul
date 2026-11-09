// @vitest-environment node
import { describe, expect, it } from "vitest";

import { parseInstanceLink } from "@/utils/MessageTextUtility";

describe("parseInstanceLink", () => {
  const origin = "https://nexul.example.com";

  it("reads the workspace and section of a same-origin URL", () => {
    expect(parseInstanceLink(`${origin}/acme/chat/c-1?x=1#m`, origin)).toEqual({
      path: "/acme/chat/c-1?x=1#m",
      workspace: "acme",
      section: "chat",
      rest: ["c-1"],
    });
  });

  it("treats an unprefixed page as having no workspace", () => {
    expect(parseInstanceLink(`${origin}/settings/devices`, origin)).toMatchObject({ workspace: undefined, section: "settings" });
  });

  it("returns null for another origin", () => {
    expect(parseInstanceLink("https://evil.dev/acme/chat/c-1", origin)).toBeNull();
  });
});
