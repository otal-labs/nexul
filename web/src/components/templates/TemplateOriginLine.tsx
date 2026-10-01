import { Button } from "@/components/ui/button";
import { useResetTemplate } from "@/hooks/TemplateHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import type { TemplateKind, TemplateLocation } from "@/models/Template";

// Following and edited for the kinds that follow the instance live; matches and differs for the ones copied at creation.
export type TemplateOrigin = "following" | "edited" | "matches" | "differs";

const ORIGIN_TEXT: Record<TemplateOrigin, string> = {
  following: "Following the instance template",
  edited: "Edited for this workspace",
  matches: "Matches the instance template",
  differs: "Differs from the instance template",
};

interface TemplateOriginLineProps {
  kind: TemplateKind;
  templateKey: string;
  at: TemplateLocation;
  state: TemplateOrigin;
  canReset: boolean;
}

export const TemplateOriginLine = ({ kind, templateKey, at, state, canReset }: TemplateOriginLineProps) => {
  const reset = useResetTemplate();
  const { open: confirm } = useConfirmationDialog();
  const ownText = state === "edited" || state === "differs";

  const onReset = async () => {
    const ok = await confirm({
      title: "Reset to the instance template?",
      message: "This text is replaced with the instance template's.",
      confirmLabel: "Reset",
    });
    if (ok) reset.mutate({ kind, key: templateKey, at });
  };

  return (
    <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground">
      <span>{ORIGIN_TEXT[state]}</span>
      {canReset && ownText && (
        <Button type="button" variant="link" size="sm" className="h-auto p-0 text-xs" loading={reset.isPending} onClick={() => void onReset()}>
          Reset to instance template
        </Button>
      )}
    </div>
  );
};
