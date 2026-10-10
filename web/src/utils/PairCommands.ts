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

// T3 Code's own labels, in order (Settings → Connections); Local network only shows once Network access is on.
export const desktopSteps = (viaTunnel: boolean): string[] => [
  "In T3 Code on the computer, open Settings → Connections.",
  ...(viaTunnel ? [] : ["Switch on Network access, so T3 Code answers on the local network."]),
  "Under Authorized clients choose Create link, then Create link again in the dialog.",
  viaTunnel
    ? "On the new link choose Share, then Copy link, and paste it below."
    : "On the new link choose Share, pick Local network under Reach this machine via, then Copy link and paste it below.",
];

const DESKTOP_WINDOWS = '& "$HOME\\.t3\\bin\\t3.cmd" pair';
const CLI_PAIR = "~/.local/bin/t3 pair";
const UNIT_DIR = "~/.config/systemd/user/t3code.service.d";
// The background service binds 127.0.0.1 unless T3CODE_HOST says otherwise, and Nexul on another machine can't reach that.
const LISTEN_ON_NETWORK = [`mkdir -p ${UNIT_DIR}`, `printf '[Service]\\nEnvironment=T3CODE_HOST=0.0.0.0\\n' > ${UNIT_DIR}/nexul-host.conf`];

type CommandInstall = Exclude<T3Install, typeof T3Install.Desktop>;

// Paths are spelled out because the install script's ~/.local/bin is not reliably on PATH.
// viaTunnel: the tunnel command already installed T3 Code where it was missing, so only pairing is left.
export const pairPlan = (install: CommandInstall, viaTunnel: boolean): PairPlan => {
  if (install === T3Install.CommandLine) {
    return {
      lead: "T3 Code's server has to be running on the computer. Run this there and paste the Pairing URL it prints.",
      commands: {
        [TunnelOs.Unix]: { lines: ["t3 pair"], note: "No server running? Start one with t3 service install" },
        [TunnelOs.Windows]: { lines: ["t3 pair"], note: "No server running? Start one with t3 serve" },
      },
    };
  }
  if (viaTunnel) {
    return {
      lead: "The tunnel command installed T3 Code, so only pairing is left. Run this there and paste the Pairing URL it prints.",
      commands: {
        [TunnelOs.Unix]: { lines: [CLI_PAIR], note: "The tunnel command prints this line when it finishes" },
        [TunnelOs.Windows]: { lines: [DESKTOP_WINDOWS], note: "Open T3 Code once first, so its server is running" },
      },
    };
  }
  return {
    lead: "Install T3 Code on the computer, then pair it. Pairing prints a URL; paste it below.",
    commands: {
      [TunnelOs.Unix]: {
        lines: ["curl -fsSL https://t3.codes/install.sh | sh", ...LISTEN_ON_NETWORK, "~/.local/bin/t3 service install", CLI_PAIR],
        note: "Linux: the two middle lines make T3 Code listen on the network. If it finds no server, wait a few seconds and run the last line again",
      },
      [TunnelOs.Windows]: {
        lines: ["winget install T3Tools.T3Code"],
        note: "Open T3 Code once, then choose Desktop app",
      },
    },
  };
};
