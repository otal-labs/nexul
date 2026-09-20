import { describe, expect, it } from "vitest";

import { buildRunnerInstallCommand, RunnerPlatform } from "@/utils/RunnerInstallCommand";

const baseParams = {
  wsUrl: "wss://deploy.example.com:8081/ws/runner",
  secret: "abc123def456",
  downloadUrl: "https://deploy.example.com/api/runners/download",
};

describe("buildRunnerInstallCommand", () => {
  it("builds a one-line curl command for Unix", () => {
    const command = buildRunnerInstallCommand({ platform: RunnerPlatform.Unix, ...baseParams });
    expect(command).toContain(baseParams.downloadUrl);
    expect(command).toContain('-H "Authorization: Bearer $S"');
    expect(command).toContain(`NEXUL_SERVER_WS="${baseParams.wsUrl}"`);
    expect(command).toContain("./nexul-runner");
    expect(command).not.toContain("github.com");
    expect(command).not.toContain("\n");
  });

  it("assigns the secret to a shell variable exactly once as a literal", () => {
    const command = buildRunnerInstallCommand({ platform: RunnerPlatform.Unix, ...baseParams });
    const literalOccurrences = command.split(`"${baseParams.secret}"`).length - 1;
    expect(literalOccurrences).toBe(1);
    expect(command).toContain(`S="${baseParams.secret}"`);
    expect(command).toContain('NEXUL_RUNNER_SECRET="$S"');
  });

  it("builds a one-line PowerShell command for Windows", () => {
    const command = buildRunnerInstallCommand({ platform: RunnerPlatform.Windows, ...baseParams });
    expect(command).toContain("Invoke-WebRequest");
    expect(command).toContain(`"${baseParams.downloadUrl}/windows-amd64"`);
    expect(command).toContain('-Headers @{Authorization="Bearer $s"}');
    expect(command).toContain(`$env:NEXUL_SERVER_WS="${baseParams.wsUrl}"`);
    expect(command).toContain(".\\nexul-runner.exe");
    expect(command).not.toContain("github.com");
    expect(command).not.toContain("\n");
  });

  it("assigns the secret to a PowerShell variable exactly once as a literal", () => {
    const command = buildRunnerInstallCommand({ platform: RunnerPlatform.Windows, ...baseParams });
    const literalOccurrences = command.split(`"${baseParams.secret}"`).length - 1;
    expect(literalOccurrences).toBe(1);
    expect(command).toContain(`$s="${baseParams.secret}"`);
    expect(command).toContain("$env:NEXUL_RUNNER_SECRET=$s");
  });

  it("uses the literal placeholder when no git token is given", () => {
    const unix = buildRunnerInstallCommand({ platform: RunnerPlatform.Unix, ...baseParams });
    expect(unix).toContain('NEXUL_GIT_TOKEN="<github-token>"');

    const windows = buildRunnerInstallCommand({
      platform: RunnerPlatform.Windows,
      ...baseParams,
      gitToken: "   ",
    });
    expect(windows).toContain('$env:NEXUL_GIT_TOKEN="<github-token>"');
  });

  it("interpolates a supplied git token", () => {
    const command = buildRunnerInstallCommand({
      platform: RunnerPlatform.Unix,
      ...baseParams,
      gitToken: "ghp_abc",
    });
    expect(command).toContain('NEXUL_GIT_TOKEN="ghp_abc"');
  });

  it("omits the runner name variable when name is empty", () => {
    const command = buildRunnerInstallCommand({ platform: RunnerPlatform.Unix, ...baseParams, name: "" });
    expect(command).not.toContain("NEXUL_RUNNER_NAME");
  });

  it("includes the runner name variable when name is given", () => {
    const unix = buildRunnerInstallCommand({
      platform: RunnerPlatform.Unix,
      ...baseParams,
      name: "build-box-1",
    });
    expect(unix).toContain('NEXUL_RUNNER_NAME="build-box-1" ./nexul-runner');

    const windows = buildRunnerInstallCommand({
      platform: RunnerPlatform.Windows,
      ...baseParams,
      name: "build-box-1",
    });
    expect(windows).toContain('$env:NEXUL_RUNNER_NAME="build-box-1"; .\\nexul-runner.exe');
  });

  it("escapes double quotes, dollar signs, and backticks for bash", () => {
    const command = buildRunnerInstallCommand({
      platform: RunnerPlatform.Unix,
      wsUrl: 'wss://x"$(`)y',
      secret: baseParams.secret,
      downloadUrl: baseParams.downloadUrl,
    });
    expect(command).toContain('NEXUL_SERVER_WS="wss://x\\"\\$(\\`)y"');
  });

  it("escapes double quotes, dollar signs, and backticks for PowerShell", () => {
    const command = buildRunnerInstallCommand({
      platform: RunnerPlatform.Windows,
      wsUrl: 'wss://x"$(`)y',
      secret: baseParams.secret,
      downloadUrl: baseParams.downloadUrl,
    });
    expect(command).toContain('$env:NEXUL_SERVER_WS="wss://x`"`$(``)y"');
  });
});
