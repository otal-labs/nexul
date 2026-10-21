export const CommandOs = {
  Unix: "unix",
  Windows: "windows",
} as const;

export type CommandOs = (typeof CommandOs)[keyof typeof CommandOs];

export const COMMAND_OS_LABELS: Record<CommandOs, string> = {
  [CommandOs.Unix]: "macOS / Linux",
  [CommandOs.Windows]: "Windows",
};

// What a command snippet shows for one system: the lines to run there and one muted line under them.
export interface OsCommand {
  lines: string[];
  note: string;
}

export const PAIRING_GUIDE = { href: "https://nexul.io/docs/guide/paired-computers/", label: "Pairing guide" };

// The commands Add a computer shows: the server renders both with this computer's one-time token.
export const computerCommands = (commands: { unix: string; windows: string }): Record<CommandOs, OsCommand> => ({
  [CommandOs.Unix]: { lines: [commands.unix], note: "On a Mac, T3 Code answers only while you're logged in" },
  [CommandOs.Windows]: { lines: [commands.windows], note: "PowerShell, no admin; runs while you're logged on" },
});

export const detectCommandOs = (userAgent: string): CommandOs =>
  /windows/i.test(userAgent) ? CommandOs.Windows : CommandOs.Unix;
