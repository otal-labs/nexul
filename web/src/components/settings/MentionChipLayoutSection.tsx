import { useState, type FormEvent } from "react";
import { TicketIcon } from "lucide-react";

import { SettingsCard } from "@/components/settings/SettingsCard";
import { Button } from "@/components/ui/button";
import { useUpdateMentionChipTemplate } from "@/hooks/AuthHooks";
import type { InstanceSettings } from "@/models/User";

interface MentionChipLayoutSectionProps {
  settings: InstanceSettings;
}

const TEMPLATE_TOKENS = ["Project", "Ticket", "Status", "Type", "Assignee", "Due"] as const;

// Fake preview data only — this never renders a live chip, so no API call is needed.
const PREVIEW_VALUES: Record<(typeof TEMPLATE_TOKENS)[number], string> = {
  Project: "ERF-1",
  Ticket: "Fix login redirect loop",
  Status: "In progress",
  Type: "bug",
  Assignee: "Sam",
  Due: "Aug 30",
};

// Mirrors MentionChip.tsx's {ticket.Field} substitution (spec.md §6).
const renderPreview = (template: string) =>
  template.replace(/\{ticket\.(\w+)\}/g, (match, key: string) => PREVIEW_VALUES[key as keyof typeof PREVIEW_VALUES] ?? match);

// Gated on workspaces:write, same gate-in-parent pattern as RoleSettingsSection.
export const MentionChipLayoutSection = ({ settings }: MentionChipLayoutSectionProps) => {
  const [template, setTemplate] = useState(settings.mention_chip_template);
  const updateTemplate = useUpdateMentionChipTemplate();

  const onSubmit = async (event: FormEvent) => {
    event.preventDefault();
    await updateTemplate.mutateAsync(template);
  };

  const insertToken = (token: (typeof TEMPLATE_TOKENS)[number]) =>
    setTemplate((prev) => `${prev}{ticket.${token}}`);

  return (
    <SettingsCard
      id="mention-layout"
      title="Mention chip layout"
      description="What a @-mention ticket chip shows across the workspace. The icon stays fixed — everything else comes from this format string."
    >
      <form onSubmit={onSubmit} className="space-y-4">
        <div>
          <label htmlFor="mention-chip-template" className="mb-2 block text-xs font-medium text-muted-foreground">
            Format
          </label>
          <div className="flex flex-wrap items-center gap-2">
            <input
              id="mention-chip-template"
              value={template}
              onChange={(e) => setTemplate(e.target.value)}
              className="w-full min-w-0 flex-1 rounded-md border border-input bg-background px-3 py-2 font-mono text-sm shadow-xs outline-none focus-visible:ring-[3px] focus-visible:ring-ring/30 focus-visible:border-ring sm:w-96"
              spellCheck={false}
            />
            <Button type="submit" disabled={updateTemplate.isPending}>
              {updateTemplate.isPending ? "Saving…" : "Save"}
            </Button>
          </div>
          <div className="mt-2 flex flex-wrap gap-1.5">
            {TEMPLATE_TOKENS.map((token) => (
              <button
                key={token}
                type="button"
                onClick={() => insertToken(token)}
                className="rounded border border-border bg-muted/40 px-1.5 py-0.5 font-mono text-xs text-muted-foreground hover:bg-muted"
              >
                {`{ticket.${token}}`}
              </button>
            ))}
          </div>
        </div>
        <div>
          <p className="mb-2 text-xs font-medium text-muted-foreground">Preview</p>
          <span className="mention-chip inline-flex w-fit items-center gap-1.5 rounded-md border border-border bg-muted/40 px-2 py-1 text-sm">
            <TicketIcon className="h-3.5 w-3.5 shrink-0" />
            <span className="min-w-0 truncate">{renderPreview(template)}</span>
          </span>
        </div>
      </form>
    </SettingsCard>
  );
};
