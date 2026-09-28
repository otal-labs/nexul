import { describe, expect, it } from "vitest";

import { movedSettingsTarget } from "@/utils/SettingsRedirects";

describe("movedSettingsTarget", () => {
  it.each([
    ["?section=connectors&connector=github&connected=1", "", "/configuration?section=connectors&connector=github&connected=1"],
    ["?section=instance", "#instance-version", "/configuration?section=instance#instance-version"],
    ["?section=instance&tab=sign-in", "", "/configuration?section=sign-in"],
    ["?section=connectors&tab=github-app", "", "/configuration?section=connectors&tab=github-app"],
    ["?section=roles", "", "/configuration?section=roles"],
    ["?section=tokens", "", "/settings?section=security&tab=tokens"],
    ["?section=tokens&tab=personal", "", "/settings?section=security&tab=tokens"],
  ])("sends %s%s to %s", (search, hash, target) => {
    expect(movedSettingsTarget(search, hash)).toBe(target);
  });

  it.each(["", "?section=profile", "?section=appearance", "?section=security&tab=tokens", "?section=pairing&setup=c1", "?section=nope"])(
    "leaves %s on Your settings",
    (search) => {
      expect(movedSettingsTarget(search, "")).toBeUndefined();
    },
  );
});
