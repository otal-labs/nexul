import { CommandOs, type OsCommand } from "@/utils/InstallCommands";

// What pairing through a tunnel installed: cloudflared as a system service, removed with its own command and the person's sudo.
export const cloudflaredRemovalCommands: Record<CommandOs, OsCommand> = {
  [CommandOs.Unix]: { lines: ["sudo cloudflared service uninstall"], note: "Asks for your password, for sudo" },
  [CommandOs.Windows]: { lines: ["cloudflared service uninstall"], note: "Administrator PowerShell" },
};

export const MOVE_GUIDE = { href: "https://nexul.io/docs/guide/paired-computers/#move-a-computer-to-the-nexul-app", label: "Moving guide" };
