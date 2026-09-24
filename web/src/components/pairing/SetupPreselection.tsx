import { Checkbox } from "@/components/ui/checkbox";

import { cn } from "@/lib/utils";

const CHOICES = [
  { id: "setup-skill-set", label: "Default skill set", detail: "mattpocock/skills and nexul-memory, the whole set", mono: false },
  { id: "setup-claude-skills", label: "~/.claude/skills/", detail: "Read by Claude Code and opencode", mono: true },
  { id: "setup-agents-skills", label: "~/.agents/skills/", detail: "Read by Codex and opencode", mono: true },
] as const;

interface PreselectionChoiceProps {
  choice: (typeof CHOICES)[number];
}

const PreselectionChoice = ({ choice }: PreselectionChoiceProps) => (
  <div className="flex items-start gap-2.5">
    <Checkbox id={choice.id} checked disabled className="mt-0.5 disabled:opacity-100" />
    <label htmlFor={choice.id} className="min-w-0 space-y-0.5">
      <span className={cn("block text-sm break-words", choice.mono && "font-mono text-xs")}>{choice.label}</span>
      <span className="block text-xs text-muted-foreground">{choice.detail}</span>
    </label>
  </div>
);

// Coarse on purpose: the set is one loop, so every run installs all of it into both locations.
export const SetupPreselection = () => (
  <fieldset className="space-y-2.5">
    <legend className="mb-2.5 text-xs font-semibold">Installs</legend>
    {CHOICES.map((c) => (
      <PreselectionChoice key={c.id} choice={c} />
    ))}
  </fieldset>
);
