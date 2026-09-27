export const RunnerPlatform = {
  Unix: "unix",
  Windows: "windows",
} as const;

export type RunnerPlatform = (typeof RunnerPlatform)[keyof typeof RunnerPlatform];

const quoteForSh = (value: string): string => `'${value.replace(/'/g, "'\\''")}'`;

const quoteForPowerShell = (value: string): string => `'${value.replace(/'/g, "''")}'`;

// Appends the installer's --git-token flag, quoted for the shell the command runs in.
export const withGitToken = (command: string, platform: RunnerPlatform, gitToken: string): string => {
  if (!gitToken) return command;
  const quoted = platform === RunnerPlatform.Windows ? quoteForPowerShell(gitToken) : quoteForSh(gitToken);
  return `${command} --git-token ${quoted}`;
};
