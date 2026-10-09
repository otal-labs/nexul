import { TicketIcon } from "lucide-react";

const TEMPLATE_TOKENS = ["Project", "Ticket", "Status", "Type", "Developer", "Due"] as const;

// Fake preview data only — this never renders a live chip, so no API call is needed.
const PREVIEW_VALUES: Record<(typeof TEMPLATE_TOKENS)[number], string> = {
  Project: "ERF-1",
  Ticket: "Fix login redirect loop",
  Status: "In progress",
  Type: "bug",
  Developer: "Sam",
  Due: "Aug 30",
};

// Mirrors MentionChip.tsx's {ticket.Field} substitution (spec.md §6).
const renderPreview = (template: string) =>
  template.replace(/\{ticket\.(\w+)\}/g, (match, key: string) => PREVIEW_VALUES[key as keyof typeof PREVIEW_VALUES] ?? match);

interface MentionChipFieldProps {
  id: string;
  value: string;
  onChange: (value: string) => void;
}

export const MentionChipField = ({ id, value, onChange }: MentionChipFieldProps) => (
  <div className="space-y-5">
    <div>
      <label htmlFor={id} className="mb-2 block text-xs font-medium text-muted-foreground">
        Format
      </label>
      <input
        id={id}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="w-full min-w-0 rounded-md border border-input bg-background px-3 py-2 font-mono text-sm shadow-xs outline-none focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/30"
        spellCheck={false}
      />
      <div className="mt-2 flex flex-wrap items-center gap-1.5">
        <span className="mr-1 text-xs text-muted-foreground">Add</span>
        {TEMPLATE_TOKENS.map((token) => (
          <button
            key={token}
            type="button"
            aria-label={`Add {ticket.${token}}`}
            onClick={() => onChange(`${value}{ticket.${token}}`)}
            className="rounded-md border border-border bg-muted/40 px-1.5 py-0.5 font-mono text-xs text-muted-foreground transition-colors duration-150 ease-standard hover:bg-muted hover:text-foreground"
          >
            {token}
          </button>
        ))}
      </div>
    </div>
    <div className="rounded-md bg-surface-2 px-4 py-5">
      <p className="mb-3 text-xs font-medium text-muted-foreground">Preview</p>
      <p className="text-sm text-muted-foreground">
        Can you take{" "}
        <span className="mention-chip inline-flex max-w-full items-center gap-1.5 rounded-md border border-border bg-muted/40 px-2 py-0.5 align-middle text-sm text-foreground">
          <TicketIcon className="h-3.5 w-3.5 shrink-0" aria-hidden />
          <span className="min-w-0 truncate">{renderPreview(value)}</span>
        </span>{" "}
        after standup?
      </p>
    </div>
  </div>
);
