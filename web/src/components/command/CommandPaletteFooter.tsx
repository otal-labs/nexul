import type { ReactNode } from "react";
import { CornerDownLeftIcon } from "lucide-react";

const Key = ({ children }: { children: ReactNode }) => (
  <kbd className="flex h-5 min-w-5 items-center justify-center rounded-sm border border-border bg-muted/60 px-1 font-mono text-[11px] text-muted-foreground">
    {children}
  </kbd>
);

interface CommandPaletteFooterProps {
  // What the search found, or that it found more than it shows; empty while browsing.
  status: string;
  // Whether a row is there for Enter to open.
  open: boolean;
}

// What the search found, and the keys; hidden from screen readers, whose own commands and the live count already say this.
export const CommandPaletteFooter = ({ status, open }: CommandPaletteFooterProps) => (
  <div aria-hidden className="flex h-9 items-center gap-4 border-t border-border px-4 text-xs text-muted-foreground">
    <span className="min-w-0 flex-1 truncate tabular-nums">{status}</span>
    {open && (
      <span className="flex items-center gap-1.5">
        <Key>
          <CornerDownLeftIcon className="size-3" />
        </Key>
        open
      </span>
    )}
    <span className="flex items-center gap-1.5">
      <Key>esc</Key>
      close
    </span>
  </div>
);
