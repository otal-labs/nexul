import type { ComponentType } from "react";

import { EmptyState } from "@/components/EmptyState";
import { commandPaletteShortcut } from "@/lib/shortcut";

interface ListDetailPlaceholderProps {
  icon: ComponentType<{ className?: string }>;
  title: string;
  /** What the list holds, in a few words: "27 docs in Atlas Platform". */
  summary?: string;
  /** The command palette finds these records, so the open pane can teach its shortcut. */
  searchable?: boolean;
}

// What the open pane shows before a row is picked: the list's own mark, what it holds, and the quick way to find one.
export const ListDetailPlaceholder = ({ icon, title, summary, searchable = false }: ListDetailPlaceholderProps) => (
  <EmptyState
    role="status"
    icon={icon}
    title={title}
    size="compact"
    className="border-0"
    {...(summary ? { message: summary } : {})}
    action={
      searchable && (
        <p className="text-xs text-muted-foreground">
          <kbd className="rounded-sm border border-border px-1.5 py-px font-mono text-[11px]">{commandPaletteShortcut()}</kbd> to search
        </p>
      )
    }
  />
);
