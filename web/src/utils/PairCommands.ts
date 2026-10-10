import { TunnelOs, type OsCommand } from "@/utils/TunnelInstallCommands";

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
  commands: Record<TunnelOs, OsCommand>;
}

const DESKTOP_UNIX = "~/.t3/bin/t3 pair";
const DESKTOP_WINDOWS = '& "$HOME\\.t3\\bin\\t3.cmd" pair';
const CLI_PAIR = "~/.local/bin/t3 pair";

// Paths are spelled out because neither the desktop launcher (~/.t3/bin) nor the install script's ~/.local/bin is reliably on PATH.
// viaTunnel: the tunnel command already installed T3 Code where it was missing, so only pairing is left.
export const pairPlan = (install: T3Install, viaTunnel: boolean): PairPlan => {
  if (install === T3Install.CommandLine) {
    return {
      lead: "T3 Code's server has to be running on the computer. Run this there.",
      commands: {
        [TunnelOs.Unix]: { lines: ["t3 pair"], note: "No server running? Start one with t3 service install" },
        [TunnelOs.Windows]: { lines: ["t3 pair"], note: "No server running? Start one with t3 serve" },
      },
    };
  }
  const desktopNote = "Install t3 command in Settings → General → About for plain t3 pair";
  if (install === T3Install.Desktop) {
    return {
      lead: "Keep T3 Code open on the computer and run this there.",
      commands: {
        [TunnelOs.Unix]: { lines: [DESKTOP_UNIX], note: desktopNote },
        [TunnelOs.Windows]: { lines: [DESKTOP_WINDOWS], note: desktopNote },
      },
    };
  }
  if (viaTunnel) {
    return {
      lead: "The tunnel command installed T3 Code, so only pairing is left.",
      commands: {
        [TunnelOs.Unix]: { lines: [CLI_PAIR], note: "The tunnel command prints this line when it finishes" },
        [TunnelOs.Windows]: { lines: [DESKTOP_WINDOWS], note: "Open T3 Code once first, so its server is running" },
      },
    };
  }
  return {
    lead: "Install T3 Code on the computer, then pair it.",
    commands: {
      [TunnelOs.Unix]: {
        lines: ["curl -fsSL https://t3.codes/install.sh | sh", "~/.local/bin/t3 service install", CLI_PAIR],
        note: "If it finds no server, wait a few seconds and run the last line again",
      },
      [TunnelOs.Windows]: {
        lines: ["winget install T3Tools.T3Code"],
        note: "Open T3 Code once, then choose Desktop app",
      },
    },
  };
};
