export const TunnelOs = {
  Unix: "unix",
  Windows: "windows",
} as const;

export type TunnelOs = (typeof TunnelOs)[keyof typeof TunnelOs];

export const TUNNEL_OS_LABELS: Record<TunnelOs, string> = {
  [TunnelOs.Unix]: "macOS / Linux",
  [TunnelOs.Windows]: "Windows",
};

// What a command snippet shows for one system: the lines to run there and one muted line under them.
export interface OsCommand {
  lines: string[];
  note: string;
}

export const PAIRING_GUIDE = { href: "https://nexul.io/docs/guide/paired-computers/", label: "Pairing guide" };

// nexul.io's tunnel script runs cloudflared as a service with this computer's token, then makes sure T3 Code answers
// (website/public/tunnel.sh, tunnel.ps1).
export const tunnelCommands = (token: string): Record<TunnelOs, OsCommand> => ({
  [TunnelOs.Unix]: {
    lines: [`curl -fsSL https://nexul.io/tunnel.sh | sh -s -- ${token}`],
    note: "Also installs T3 Code if it's missing",
  },
  [TunnelOs.Windows]: {
    lines: [`& ([scriptblock]::Create((irm https://nexul.io/tunnel.ps1))) ${token}`],
    note: "Admin PowerShell; installs T3 Code if it's missing",
  },
});

// The command Add a computer shows, the same on Linux and macOS: the server renders it with this computer's one-time token.
export const computerCommands = (commands: { unix: string; windows: string }): Record<TunnelOs, OsCommand> => ({
  [TunnelOs.Unix]: { lines: [commands.unix], note: "On a Mac, T3 Code answers only while you're logged in" },
  [TunnelOs.Windows]: { lines: commands.windows ? [commands.windows] : [], note: "Windows is coming soon" },
});

export const detectTunnelOs = (userAgent: string): TunnelOs =>
  /windows/i.test(userAgent) ? TunnelOs.Windows : TunnelOs.Unix;
