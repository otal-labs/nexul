import { ArrowUpRight } from "lucide-react";
import { useState } from "react";

import { CopyButton } from "@/components/settings/CopyButton";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { cn } from "@/lib/utils";
import { detectTunnelOs, TUNNEL_OS_LABELS, TunnelOs, type OsCommand } from "@/utils/TunnelInstallCommands";

interface CommandSnippetProps {
  commands: Record<TunnelOs, OsCommand>;
  // Names what the lines do, for the copy button and the screen reader.
  label: string;
  guide: { href: string; label: string };
}

// The site's install snippet: a system switch, one raised row whose long line scrolls under a faded edge, a note and a guide link.
export const CommandSnippet = ({ commands, label, guide }: CommandSnippetProps) => {
  const [picked, setPicked] = useState<TunnelOs>();
  const os = picked ?? detectTunnelOs(navigator.userAgent);
  const command = commands[os];
  return (
    <div className="min-w-0 space-y-2.5">
      <ToggleGroup
        type="single"
        variant="segmented"
        size="xs"
        aria-label="Operating system"
        value={os}
        onValueChange={(next) => next && setPicked(next as TunnelOs)}
      >
        {Object.values(TunnelOs).map((value) => (
          <ToggleGroupItem key={value} value={value}>
            {TUNNEL_OS_LABELS[value]}
          </ToggleGroupItem>
        ))}
      </ToggleGroup>
      <div
        key={os}
        className={cn(
          "canvas-card flex min-w-0 items-center gap-1 py-1 pr-1 pl-3.5 has-[pre:focus-visible]:outline-2 has-[pre:focus-visible]:outline-offset-2 has-[pre:focus-visible]:outline-(--focus)",
          picked && "motion-safe:animate-in motion-safe:fade-in-0 motion-safe:duration-[160ms] motion-safe:ease-out",
          picked === TunnelOs.Windows && "motion-safe:slide-in-from-right-[6px]",
          picked === TunnelOs.Unix && "motion-safe:slide-in-from-left-[6px]",
        )}
      >
        <pre
          tabIndex={0}
          aria-label={`${label}, ${TUNNEL_OS_LABELS[os]}`}
          className="min-w-0 flex-1 overflow-x-auto py-1.5 font-mono text-xs leading-6 [scrollbar-width:none] [mask-image:linear-gradient(to_left,transparent,#000_2rem)] quiet-focus [&::-webkit-scrollbar]:hidden"
        >
          {command.lines.map((line) => (
            <code key={line} className="block w-max pr-8">
              {line}
            </code>
          ))}
        </pre>
        <CopyButton
          value={command.lines.join("\n")}
          label={`Copy ${label.toLowerCase()}`}
          iconOnly
          variant="default"
          className="size-8 text-brand-foreground hover:text-brand-foreground"
        />
      </div>
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
    </div>
  );
};
