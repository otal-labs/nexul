import { personLabel, type Person } from "@nexul/client-core/person";

import { PersonAvatar } from "@/components/PersonAvatar";
import { cn } from "@/lib/utils";

interface PanelTargetRowProps {
  person: Person;
  selected: boolean;
  onSelect: () => void;
  subline: string;
}

// Wide: a list row with the 2px brand edge when selected. At 768px: the same person as a compact tab in a scrolling row.
export const PanelTargetRow = ({ person, selected, onSelect, subline }: PanelTargetRowProps) => (
  <li className="shrink-0">
    <button
      type="button"
      aria-current={selected ? "true" : undefined}
      onClick={onSelect}
      className={cn(
        "flex w-full items-center gap-2.5 rounded-md px-2.5 py-1.5 text-left transition-colors duration-150 ease-standard @3xl/panel:rounded-none @3xl/panel:border-l-2 @3xl/panel:px-3 @3xl/panel:py-2",
        selected && "bg-accent ring-1 ring-border @3xl/panel:border-l-brand @3xl/panel:ring-0",
        !selected && "hover:bg-accent/40 @3xl/panel:border-l-transparent",
      )}
    >
      <PersonAvatar login={person.login} src={person.avatar_url} className="size-6" label={personLabel(person)} />
      <span className="min-w-0 flex-1">
        <span title={personLabel(person)} className="block truncate text-[13px] font-medium">
          {personLabel(person)}
        </span>
        <span className="hidden truncate font-mono text-[11px] text-muted-foreground @3xl/panel:block">{subline}</span>
      </span>
    </button>
  </li>
);
