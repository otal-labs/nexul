import { describe, expect, it } from "vitest";

import { RunnerPlatform, withGitToken } from "@/utils/RunnerInstallCommand";

const unix = "curl -fsSL https://nexul.io/runner.sh | sh -s -- --server https://x --name a --code nxe_1";
const windows = "& ([scriptblock]::Create((irm https://nexul.io/runner.ps1))) --server https://x --name a --code nxe_1";

describe("withGitToken", () => {
  it("leaves the command alone without a token", () => {
    expect(withGitToken(unix, RunnerPlatform.Unix, "")).toBe(unix);
    expect(withGitToken(windows, RunnerPlatform.Windows, "")).toBe(windows);
  });

  it("single-quotes the token for sh, escaping embedded quotes", () => {
    expect(withGitToken(unix, RunnerPlatform.Unix, "ghp_a'b$c")).toBe(`${unix} --git-token 'ghp_a'\\''b$c'`);
  });

  it("single-quotes the token for PowerShell, doubling embedded quotes", () => {
    expect(withGitToken(windows, RunnerPlatform.Windows, "ghp_a'b$c")).toBe(`${windows} --git-token 'ghp_a''b$c'`);
  });
});
