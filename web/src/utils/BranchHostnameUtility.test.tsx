import { describe, expect, it } from "vitest";

import { exampleBranch, hostnameLabel, previewHostname } from "@/utils/BranchHostnameUtility";

describe("hostnameLabel", () => {
  it.each([
    ["feature/*", "feature/dot.test", "dot-test"],
    ["feature/*", "feature/security-test", "security-test"],
    ["feature/*", "feature/Fix.Login", "fix-login"],
    ["staging/*", "staging/eu/west", "eu-west"],
    ["feature/*", "feature/", "feature"],
    ["staging", "staging", "staging"],
    ["release/v1.2", "release/v1.2", "release-v1-2"],
    ["feature/*", `feature/${"a".repeat(70)}`, "a".repeat(63)],
  ])("%s on %s is %s", (pattern, branch, want) => {
    expect(hostnameLabel(pattern, branch)).toBe(want);
  });
});

describe("previewHostname", () => {
  it("fills * with the branch label", () => {
    expect(previewHostname("*.example.com", "feature/*", "feature/dot.test")).toBe("dot-test.example.com");
  });

  it("previews a wildcard row with its example branch", () => {
    expect(previewHostname("*.example.com", "feature/*", exampleBranch("feature/*"))).toBe(
      "security-test.example.com",
    );
  });

  it("uses the exact branch as its own example", () => {
    expect(exampleBranch("staging")).toBe("staging");
  });
});
