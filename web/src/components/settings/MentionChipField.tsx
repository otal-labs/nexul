import { TicketIcon } from "lucide-react";

import { mentionChipText, type MentionChipData } from "@nexul/client-core/mention";

import { Textarea } from "@/components/ui/textarea";

const TEMPLATE_TOKENS = ["Project", "Ticket", "Status", "Type", "Developer", "Due"] as const;

// A made-up ticket run through the same mentionChipText every chip uses, so the preview never drifts from a real chip.
const PREVIEW_TICKET: MentionChipData = {
  type: "ticket",
  id: "preview",
  title: "Fix login redirect loop",
  status_label: "In progress",
  can_open: false,
  project_prefix: "ERF",
  project_number: 1,
  type_label: "bug",
  developer_label: "Sam",
  due_label: "Aug 30",
};

// The same markup and classes as a live chip (MentionChip), so the preview sits on the text baseline exactly as one does.
const Chip = ({ template }: { template: string }) => (
  <span className="mention-chip mention-chip--inert">
    <TicketIcon className="size-[1.1em] shrink-0" aria-hidden />
    <span className="mention-chip__label">{mentionChipText("ticket", template, PREVIEW_TICKET, PREVIEW_TICKET.title)}</span>
  </span>
);

// The chip in a chat line and in a doc line, the two places people meet it.
const MentionPreview = ({ template }: { template: string }) => (
  <div className="space-y-3 border-t border-border pt-4">
    <p className="text-sm font-medium">Preview</p>
    <p className="text-sm text-pretty text-foreground">
      Can you take <Chip template={template} /> after standup?
    </p>
    <p className="text-sm text-pretty text-foreground">
      Blocked by <Chip template={template} /> until it ships
    </p>
  </div>
);

interface MentionChipFieldProps {
  id: string;
  value: string;
  onChange: (value: string) => void;
}

export const MentionChipField = ({ id, value, onChange }: MentionChipFieldProps) => (
  <div className="space-y-5">
    <div>
      <label htmlFor={id} className="mb-2 block text-sm font-medium">
        Format
      </label>
      {/* A one-line format that wraps instead of clipping, so every field in a long one stays readable. */}
      <Textarea
        id={id}
        value={value}
        rows={1}
        onChange={(e) => onChange(e.target.value.replace(/\n/g, " "))}
        onKeyDown={(e) => {
          if (e.key !== "Enter") return;
          e.preventDefault();
          e.currentTarget.form?.requestSubmit();
        }}
        className="field-sizing-content min-h-0 resize-none bg-surface-2 font-mono"
        spellCheck={false}
      />
      <div className="mt-2 flex flex-wrap items-center gap-1.5">
        {TEMPLATE_TOKENS.map((token) => (
          <button
            key={token}
            type="button"
            aria-label={`Add {ticket.${token}}`}
            disabled={value.includes(`{ticket.${token}}`)}
            onClick={() => onChange(`${value}${value && !value.endsWith(" ") ? " " : ""}{ticket.${token}}`)}
            className="rounded-md border border-border bg-muted/40 px-1.5 py-0.5 font-mono text-xs text-muted-foreground transition-colors duration-150 ease-standard enabled:hover:bg-muted enabled:hover:text-foreground disabled:opacity-50"
          >
            + {token}
          </button>
        ))}
      </div>
    </div>
    <MentionPreview template={value} />
  </div>
);
