import { CopyButton } from "@/components/settings/CopyButton";
import { cn } from "@/lib/utils";

interface CommandRowProps {
  lines: string[];
  // Names the command for the screen reader on the text, and for the copy button as "Copy <copyLabel>".
  ariaLabel: string;
  copyLabel: string;
  // A line read beside a form: a well with a quiet copy button, so it never reads as a field or a second primary action.
  quiet?: boolean;
  className?: string;
}

// One raised row: the command's long lines scroll under a faded edge, and a copy button sits at its end.
export const CommandRow = ({ lines, ariaLabel, copyLabel, quiet = false, className }: CommandRowProps) => (
  <div
    className={cn(
      quiet ? "rounded-lg bg-surface-2 ring-1 ring-border" : "canvas-card",
      quiet ? "py-0.5" : "py-1",
      "flex min-w-0 items-center gap-1 pr-1 pl-3.5 has-[pre:focus-visible]:outline-2 has-[pre:focus-visible]:outline-offset-2 has-[pre:focus-visible]:outline-(--focus)",
      className,
    )}
  >
    <pre
      tabIndex={0}
      aria-label={ariaLabel}
      className={cn("min-w-0 flex-1 overflow-x-auto font-mono text-xs leading-6 [scrollbar-width:none] [mask-image:linear-gradient(to_left,transparent,#000_2rem)] quiet-focus [&::-webkit-scrollbar]:hidden", quiet ? "py-1" : "py-1.5")}
    >
      {lines.map((line) => (
        <code key={line} className="block w-max pr-8">
          {line}
        </code>
      ))}
    </pre>
    <CopyButton
      value={lines.join("\n")}
      label={`Copy ${copyLabel}`}
      iconOnly
      variant={quiet ? "ghost" : "default"}
      className={cn("size-8", !quiet && "text-brand-foreground hover:text-brand-foreground")}
    />
  </div>
);
