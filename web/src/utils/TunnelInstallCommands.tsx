export const TunnelOs = {
  Linux: "linux",
  MacOS: "macos",
  Windows: "windows",
} as const;

export type TunnelOs = (typeof TunnelOs)[keyof typeof TunnelOs];

export interface TunnelInstallStep {
  title: string;
  shell: string;
  lines: string[];
}

export const TUNNEL_OS_LABELS: Record<TunnelOs, string> = {
  [TunnelOs.Linux]: "Linux",
  [TunnelOs.MacOS]: "macOS",
  [TunnelOs.Windows]: "Windows",
};

// One command per system: nexul.io's tunnel script installs cloudflared when it is missing and runs it as a service
// with this computer's token (website/public/tunnel.sh, tunnel.ps1).
export const tunnelInstallSteps = (os: TunnelOs, token: string): TunnelInstallStep[] => {
  if (os === TunnelOs.Windows) {
    return [
      {
        title: "Run this in PowerShell opened as administrator",
        shell: "powershell",
        lines: [`& ([scriptblock]::Create((irm https://nexul.io/tunnel.ps1))) ${token}`],
      },
    ];
  }
  // On a Mac it asks for no sudo: the service is a login item, which is when T3 Code runs too.
  return [
    {
      title: "Run this in a terminal",
      shell: os === TunnelOs.MacOS ? "zsh" : "bash",
      lines: [`curl -fsSL https://nexul.io/tunnel.sh | sh -s -- ${token}`],
    },
  ];
};

export const detectTunnelOs = (userAgent: string): TunnelOs => {
  if (/windows/i.test(userAgent)) return TunnelOs.Windows;
  if (/mac os|macintosh/i.test(userAgent)) return TunnelOs.MacOS;
  return TunnelOs.Linux;
};
