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

// Every `service install <token>` writes the token to a file and runs cloudflared with --token-file, so it never shows in ps.
export const tunnelInstallSteps = (os: TunnelOs, token: string): TunnelInstallStep[] => {
  if (os === TunnelOs.MacOS) {
    return [
      { title: "Install cloudflared", shell: "zsh", lines: ["brew install cloudflared"] },
      // Without sudo it is a login item: no tunnel before sign-in, when T3 Code isn't running either.
      { title: "Run it as a service with this computer's token", shell: "zsh", lines: [`cloudflared service install ${token}`] },
    ];
  }
  if (os === TunnelOs.Windows) {
    return [
      { title: "Install cloudflared in a terminal opened as administrator", shell: "powershell", lines: ["winget install --id Cloudflare.cloudflared"] },
      { title: "Open a new administrator terminal and run it as a service", shell: "powershell", lines: [`cloudflared.exe service install ${token}`] },
    ];
  }
  return [
    {
      title: "Add Cloudflare's package repository and install cloudflared (Debian, Ubuntu)",
      shell: "bash",
      lines: [
        "sudo mkdir -p --mode=0755 /usr/share/keyrings",
        "curl -fsSL https://pkg.cloudflare.com/cloudflare-main.gpg | sudo tee /usr/share/keyrings/cloudflare-main.gpg >/dev/null",
        'echo "deb [signed-by=/usr/share/keyrings/cloudflare-main.gpg] https://pkg.cloudflare.com/cloudflared any main" | sudo tee /etc/apt/sources.list.d/cloudflared.list',
        "sudo apt-get update && sudo apt-get install cloudflared",
      ],
    },
    { title: "Run it as a service with this computer's token", shell: "bash", lines: [`sudo cloudflared service install ${token}`] },
  ];
};

export const detectTunnelOs = (userAgent: string): TunnelOs => {
  if (/windows/i.test(userAgent)) return TunnelOs.Windows;
  if (/mac os|macintosh/i.test(userAgent)) return TunnelOs.MacOS;
  return TunnelOs.Linux;
};
