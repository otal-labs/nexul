import { describe, expect, it } from "vitest";

import { movedSettingsTarget } from "@/utils/SettingsRedirects";

describe("movedSettingsTarget", () => {
  it.each([
    ["connectors", "?connector=github&connected=1", "", "/configuration/connectors?connector=github&connected=1"],
    ["instance", "", "#instance-version", "/configuration/instance#instance-version"],
    ["instance", "?tab=sign-in", "", "/configuration/sign-in"],
    ["connectors", "?tab=github-app", "", "/configuration/connectors?tab=github-app"],
    ["roles", "", "", "/configuration/roles"],
    ["tokens", "", "", "/settings/security?tab=tokens"],
    ["tokens", "?tab=personal", "", "/settings/security?tab=tokens"],
  ])("sends /settings/%s%s%s to %s", (section, search, hash, target) => {
    expect(movedSettingsTarget(section, search, hash)).toBe(target);
  });

  it.each([
    [undefined, ""],
    ["profile", ""],
    ["appearance", ""],
    ["security", "?tab=tokens"],
    ["pairing", "?setup=c1"],
    ["nope", ""],
  ])("leaves /settings/%s%s on Your settings", (section, search) => {
    expect(movedSettingsTarget(section, search, "")).toBeUndefined();
  });
});
