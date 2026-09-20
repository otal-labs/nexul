export const RunnerPlatform = {
  Unix: "unix",
  Windows: "windows",
} as const;

export type RunnerPlatform = (typeof RunnerPlatform)[keyof typeof RunnerPlatform];

interface BuildRunnerInstallCommandParams {
  platform: RunnerPlatform;
  wsUrl: string;
  secret: string;
  downloadUrl: string;
  gitToken?: string;
  name?: string;
  // Machine the runner reports on connect (NEXUL_MACHINE); set when adding a runner to a known machine.
  machine?: string | undefined;
}

const escapeForBash = (value: string): string =>
  value.replace(/"/g, '\\"').replace(/\$/g, "\\$").replace(/`/g, "\\`");

const escapeForPowerShell = (value: string): string =>
  value.replace(/`/g, "``").replace(/\$/g, "`$").replace(/"/g, '`"');

const buildUnixCommand = ({
  wsUrl,
  secret,
  downloadUrl,
  gitToken,
  name,
  machine,
}: BuildRunnerInstallCommandParams): string => {
  const token = gitToken?.trim() ? escapeForBash(gitToken) : "<github-token>";
  const nameSegment = name?.trim() ? `NEXUL_RUNNER_NAME="${escapeForBash(name)}" ` : "";
  const machineSegment = machine?.trim() ? `NEXUL_MACHINE="${escapeForBash(machine)}" ` : "";
  return (
    `S="${escapeForBash(secret)}" && curl -fsSL -H "Authorization: Bearer $S" ` +
    `"${escapeForBash(downloadUrl)}/$(uname -s | tr '[:upper:]' '[:lower:]')` +
    `-$(uname -m | sed -e 's/x86_64/amd64/' -e 's/aarch64/arm64/')" -o nexul-runner && ` +
    `chmod +x nexul-runner && NEXUL_SERVER_WS="${escapeForBash(wsUrl)}" ` +
    `NEXUL_RUNNER_SECRET="$S" NEXUL_GIT_TOKEN="${token}" ` +
    `${nameSegment}${machineSegment}./nexul-runner`
  );
};

const buildWindowsCommand = ({
  wsUrl,
  secret,
  downloadUrl,
  gitToken,
  name,
  machine,
}: BuildRunnerInstallCommandParams): string => {
  const token = gitToken?.trim() ? escapeForPowerShell(gitToken) : "<github-token>";
  const nameSegment = name?.trim() ? `$env:NEXUL_RUNNER_NAME="${escapeForPowerShell(name)}"; ` : "";
  const machineSegment = machine?.trim() ? `$env:NEXUL_MACHINE="${escapeForPowerShell(machine)}"; ` : "";
  return (
    `$s="${escapeForPowerShell(secret)}"; Invoke-WebRequest -Headers @{Authorization="Bearer $s"} ` +
    `"${escapeForPowerShell(downloadUrl)}/windows-amd64" -OutFile nexul-runner.exe; ` +
    `$env:NEXUL_SERVER_WS="${escapeForPowerShell(wsUrl)}"; ` +
    `$env:NEXUL_RUNNER_SECRET=$s; ` +
    `$env:NEXUL_GIT_TOKEN="${token}"; ` +
    `${nameSegment}${machineSegment}.\\nexul-runner.exe`
  );
};

export const buildRunnerInstallCommand = (params: BuildRunnerInstallCommandParams): string => {
  if (params.platform === RunnerPlatform.Windows) return buildWindowsCommand(params);
  return buildUnixCommand(params);
};
