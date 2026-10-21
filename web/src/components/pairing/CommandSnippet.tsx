import { ArrowUpRight } from "lucide-react";
import { useState } from "react";

import { CommandRow } from "@/components/pairing/CommandRow";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { cn } from "@/lib/utils";
import { detectCommandOs, COMMAND_OS_LABELS, CommandOs, type OsCommand } from "@/utils/InstallCommands";

interface CommandSnippetProps {
  commands: Record<CommandOs, OsCommand>;
  // Names what the lines do, for the copy button and the screen reader.
  label: string;
  guide: { href: string; label: string };
}

// The site's install snippet: a system switch, one raised row whose long line scrolls under a faded edge, a note and a guide link,
// then a system's second command with its own title when it has one.
export const CommandSnippet = ({ commands, label, guide }: CommandSnippetProps) => {
  const [picked, setPicked] = useState<CommandOs>();
  const os = picked ?? detectCommandOs(navigator.userAgent);
  const command = commands[os];
  return (
    <div className="min-w-0 space-y-2.5">
      <ToggleGroup
        type="single"
        variant="segmented"
        size="xs"
        aria-label="Operating system"
        value={os}
        onValueChange={(next) => next && setPicked(next as CommandOs)}
      >
        {Object.values(CommandOs).map((value) => (
          <ToggleGroupItem key={value} value={value}>
            {COMMAND_OS_LABELS[value]}
          </ToggleGroupItem>
        ))}
      </ToggleGroup>
      <CommandRow
        key={os}
        lines={command.lines}
        ariaLabel={`${label}, ${COMMAND_OS_LABELS[os]}`}
        copyLabel={label.toLowerCase()}
        className={cn(
          picked && "motion-safe:animate-in motion-safe:fade-in-0 motion-safe:duration-[160ms] motion-safe:ease-out",
          picked === CommandOs.Windows && "motion-safe:slide-in-from-right-[6px]",
          picked === CommandOs.Unix && "motion-safe:slide-in-from-left-[6px]",
        )}
      />
      <div className="flex min-w-0 items-center justify-between gap-4 text-xs text-muted-foreground">
        <span className="min-w-0">{command.note}</span>
        <a
          href={guide.href}
          target="_blank"
          rel="noreferrer"
          className="group inline-flex shrink-0 items-center gap-1 font-medium text-foreground"
        >
          {guide.label}
          <ArrowUpRight
            className="size-3.5 text-muted-foreground transition-transform duration-150 ease-out group-hover:translate-x-0.5 group-hover:-translate-y-0.5"
            aria-hidden
          />
        </a>
      </div>
      {command.alternative && (
        <div className="min-w-0 space-y-2 pt-1">
          <p className="text-xs">
            <span className="font-medium text-foreground">{command.alternative.title}</span>
            <span className="text-muted-foreground"> · {command.alternative.note}</span>
          </p>
          <CommandRow
            lines={command.alternative.lines}
            ariaLabel={`${label}, ${command.alternative.title.toLowerCase()}`}
            copyLabel={`${command.alternative.title.toLowerCase()} command`}
          />
        </div>
      )}
    </div>
  );
};
