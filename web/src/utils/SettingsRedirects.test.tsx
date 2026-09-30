import { describe, expect, it } from "vitest";

import { movedConfigurationTarget, movedSettingsTarget } from "@/utils/SettingsRedirects";

describe("movedConfigurationTarget", () => {
  it.each([
    ["connectors", "?connector=github&connected=1", "", "/settings/connectors?connector=github&connected=1"],
    ["connectors", "?tab=github-app", "", "/settings/connectors?tab=github-app"],
    ["instance", "", "#instance-version", "/settings/instance#instance-version"],
    ["instance", "?tab=sign-in", "", "/settings/sign-in"],
    ["sign-in", "?tab=google", "", "/settings/sign-in?tab=google"],
    ["dns", "", "", "/settings/dns"],
    ["members", "", "", "/settings/team"],
    ["access", "?person=u-1", "", "/settings/team?person=u-1"],
  ])("sends /configuration/%s%s%s to %s", (section, search, hash, target) => {
    expect(movedConfigurationTarget(section, search, hash)).toBe(target);
  });

  // Team is the viewer's call (instance-wide on Settings, scoped in Configuration), so it never redirects by path alone.
  it.each([[undefined], ["roles"], ["plays"], ["team"], ["danger"], ["nope"]])(
    "leaves /configuration/%s where it is",
    (section) => {
      expect(movedConfigurationTarget(section, "?x=1", "")).toBeUndefined();
    },
  );
});

describe("movedSettingsTarget", () => {
  it.each([
    ["roles", "", "", "/configuration/roles"],
    ["mentions", "?x=1", "", "/configuration/mentions?x=1"],
    ["instance", "?tab=sign-in", "", "/settings/sign-in"],
    ["members", "", "", "/settings/team"],
    ["tokens", "", "", "/settings/security?tab=tokens"],
    ["tokens", "?tab=personal", "", "/settings/security?tab=tokens"],
  ])("sends /settings/%s%s%s to %s", (section, search, hash, target) => {
    expect(movedSettingsTarget(section, search, hash)).toBe(target);
  });

  it.each([
    [undefined, ""],
    ["profile", ""],
    ["security", "?tab=tokens"],
    ["pairing", "?setup=c1"],
    ["instance", ""],
    ["connectors", "?connector=github&connected=1"],
    ["team", "?person=u-1"],
    ["nope", ""],
  ])("leaves /settings/%s%s on Settings", (section, search) => {
    expect(movedSettingsTarget(section, search, "")).toBeUndefined();
  });
});
