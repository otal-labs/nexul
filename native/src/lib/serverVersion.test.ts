import { compareVersions, serverIsSupported } from "@/lib/serverVersion";

describe("serverIsSupported", () => {
  const min = "v0.2.0-beta.9";
  const cases: { name: string; version: string; want: boolean }[] = [
    { name: "malformed version is refused", version: "latest", want: false },
    { name: "empty version is refused", version: "", want: false },
    { name: "two-part version is refused", version: "v0.2", want: false },
    { name: "older beta is refused", version: "v0.2.0-beta.8", want: false },
    { name: "older release is refused", version: "v0.1.9", want: false },
    { name: "same beta passes", version: "v0.2.0-beta.9", want: true },
    { name: "betas compare numerically, not as strings", version: "v0.2.0-beta.10", want: true },
    { name: "dash-separated beta counts", version: "v0.2.0-beta-010", want: true },
    { name: "release beats its betas", version: "v0.2.0", want: true },
    { name: "newer release passes", version: "v0.3.0", want: true },
    { name: "newer major passes", version: "v1.0.0-beta.1", want: true },
    { name: "leading v is optional", version: "0.2.1", want: true },
    { name: "dev always passes", version: "dev", want: true },
  ];

  test.each(cases)("$name", ({ version, want }) => {
    expect(serverIsSupported(version, min)).toBe(want);
  });

  test("a release minimum refuses that release's betas", () => {
    expect(serverIsSupported("v0.2.0-beta.99", "v0.2.0")).toBe(false);
    expect(serverIsSupported("v0.2.0", "v0.2.0")).toBe(true);
  });
});

describe("compareVersions", () => {
  test("returns null when either side is malformed", () => {
    expect(compareVersions("v0.2.0", "nope")).toBeNull();
    expect(compareVersions("nope", "v0.2.0")).toBeNull();
  });

  test("orders a numeric pre-release part before a word", () => {
    expect(compareVersions("v0.2.0-1", "v0.2.0-beta")).toBeLessThan(0);
    expect(compareVersions("v0.2.0-alpha", "v0.2.0-beta")).toBeLessThan(0);
    expect(compareVersions("v0.2.0-beta", "v0.2.0-beta.1")).toBeLessThan(0);
  });
});
