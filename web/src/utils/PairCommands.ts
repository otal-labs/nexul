import { TunnelOs } from "@/utils/TunnelInstallCommands";

export const T3Install = {
  Desktop: "desktop",
  CommandLine: "command-line",
  NotInstalled: "not-installed",
} as const;

export type T3Install = (typeof T3Install)[keyof typeof T3Install];

export const T3_INSTALL_LABELS: Record<T3Install, string> = {
  [T3Install.Desktop]: "Desktop app",
  [T3Install.CommandLine]: "Command line",
  [T3Install.NotInstalled]: "Not installed yet",
};

export interface PairPlan {
  lead: string;
  shell: string;
  lines: string[];
  // The same step on the other kind of system, for a computer that isn't the one this browser runs on.
  other?: { label: string; line: string };
}

const WINDOWS_LAUNCHER = '& "$HOME\\.t3\\bin\\t3.cmd" pair';
const WINDOWS_WINGET = "winget install T3Tools.T3Code";
const POSIX_INSTALL = "curl -fsSL https://t3.codes/install.sh | sh";

// Paths are spelled out because neither the desktop launcher (~/.t3/bin) nor the install script's ~/.local/bin is reliably on PATH.
export const pairPlan = (install: T3Install, os: TunnelOs): PairPlan => {
  const windows = os === TunnelOs.Windows;
  const shell = windows ? "PowerShell" : "Terminal";
  if (install === T3Install.CommandLine) {
    return { lead: "T3 Code's server has to be running on the computer. Run this there.", shell, lines: ["t3 pair"] };
  }
  if (install === T3Install.Desktop) {
    return {
      lead: "Keep T3 Code open on the computer and run this there. Press Install next to t3 command under Settings → General → About to type just t3 pair.",
      shell,
      lines: [windows ? WINDOWS_LAUNCHER : "~/.t3/bin/t3 pair"],
      other: windows
        ? { label: "On macOS or Linux:", line: "~/.t3/bin/t3 pair" }
        : { label: "On Windows, in PowerShell:", line: WINDOWS_LAUNCHER },
    };
  }
  if (windows) {
    return {
      lead: "Install the desktop app and open it once, then come back and choose Desktop app.",
      shell,
      lines: [WINDOWS_WINGET],
      other: { label: "On macOS or Linux, install the command line with:", line: POSIX_INSTALL },
    };
  }
  return {
    lead: "Installs the t3 command, keeps its server running in the background, then prints the token. If it finds no server, wait a few seconds and run the last line again.",
    shell,
    lines: [POSIX_INSTALL, "~/.local/bin/t3 service install", "~/.local/bin/t3 pair"],
    other: { label: "On Windows, install the desktop app instead:", line: WINDOWS_WINGET },
  };
};
