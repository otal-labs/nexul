import { CheckIcon, PlusIcon, TicketIcon } from "lucide-react";

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

interface TokenButtonProps {
  token: (typeof TEMPLATE_TOKENS)[number];
  value: string;
  onChange: (value: string) => void;
}

// A field already in the format reads as added, a check in full ink, rather than fading out of reach.
const TokenButton = ({ token, value, onChange }: TokenButtonProps) => {
  const field = `{ticket.${token}}`;
  const used = value.includes(field);
  const Icon = used ? CheckIcon : PlusIcon;
  return (
    <button
      type="button"
      aria-label={used ? `${field} is in the format` : `Add ${field}`}
      disabled={used}
      onClick={() => onChange(`${value}${value && !value.endsWith(" ") ? " " : ""}${field}`)}
      className="inline-flex h-6 items-center gap-1 rounded-md border border-border bg-muted/40 px-2 text-xs text-muted-foreground transition-colors duration-150 ease-standard enabled:hover:bg-muted enabled:hover:text-foreground disabled:cursor-default disabled:text-foreground"
    >
      <Icon className="size-3" aria-hidden />
      {token}
    </button>
  );
};

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
          <TokenButton key={token} token={token} value={value} onChange={onChange} />
        ))}
      </div>
    </div>
    <MentionPreview template={value} />
  </div>
);
