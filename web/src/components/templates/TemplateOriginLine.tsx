import { ActionButton } from "@/components/action/ActionButton";
import { useResetTemplate } from "@/hooks/TemplateHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { cn } from "@/lib/utils";
import type { TemplateKind, TemplateLocation } from "@/models/Template";

// Following and edited for the kinds that follow the instance live; matches and differs for the ones copied at creation.
export type TemplateOrigin = "following" | "edited" | "matches" | "differs";

const ORIGIN_TEXT: Record<TemplateOrigin, string> = {
  following: "Following the instance template",
  edited: "Edited for this workspace",
  matches: "Matches the instance template",
  differs: "Differs from the instance template",
};

interface TemplateResetButtonProps {
  kind: TemplateKind;
  templateKey: string;
  at: TemplateLocation;
  className?: string;
}

// Puts the instance template's text back here, after asking.
export const TemplateResetButton = ({ kind, templateKey, at, className }: TemplateResetButtonProps) => {
  const reset = useResetTemplate();
  const { open: confirm } = useConfirmationDialog();

  const onReset = async () => {
    const ok = await confirm({
      title: "Reset to the instance template?",
      message: "This text is replaced with the instance template's.",
      confirmLabel: "Reset",
    });
    if (ok) reset.mutate({ kind, key: templateKey, at });
  };

  return (
    <ActionButton
      permission="templates:write"
      verb="reset templates"
      type="button"
      variant="link"
      size="sm"
      className={cn("h-auto p-0 text-xs", className)}
      loading={reset.isPending}
      onClick={() => void onReset()}
    >
      Reset to instance template
    </ActionButton>
  );
};

interface TemplateOriginLineProps {
  kind: TemplateKind;
  templateKey: string;
  at: TemplateLocation;
  state: TemplateOrigin;
}

export const TemplateOriginLine = ({ kind, templateKey, at, state }: TemplateOriginLineProps) => (
  <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground">
    <span>{ORIGIN_TEXT[state]}</span>
    {(state === "edited" || state === "differs") && <TemplateResetButton kind={kind} templateKey={templateKey} at={at} />}
  </div>
);
