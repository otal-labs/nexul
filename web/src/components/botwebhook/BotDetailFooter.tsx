import { RotateCcw } from "lucide-react";

import type { BotView } from "@/components/botwebhook/BotForm";
import { Button } from "@/components/ui/button";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useBotActions } from "@/hooks/useBotActions";
import type { Botwebhook } from "@/models/Botwebhook";

interface BotDetailFooterProps {
  /** Undefined in the create view, which has only its submit. */
  bot?: Botwebhook | undefined;
  submitLabel: string;
  showSubmit: boolean;
  submitting: boolean;
  onNavigate: (view: BotView | null) => void;
}

export const BotDetailFooter = ({ bot, submitLabel, showSubmit, submitting, onNavigate }: BotDetailFooterProps) => {
  const can = useAreaAccess();
  const actions = useBotActions();

  const regenerate = async (target: Botwebhook) => {
    if (await actions.regenerate(target)) onNavigate({ id: target.id, note: "New URL. The old one no longer works." });
  };
  const remove = async (target: Botwebhook) => {
    if (await actions.remove(target)) onNavigate(null);
  };

  return (
    <div className="flex flex-wrap items-center justify-between gap-2 border-t border-border pt-3">
      <span className="flex flex-wrap gap-1">
        {bot && (
          <Button type="button" variant="outline" size="sm" onClick={() => void regenerate(bot)}>
            <RotateCcw className="size-4" aria-hidden />
            Regenerate URL
          </Button>
        )}
        {bot && can?.("deleteBots") && (
          <Button type="button" variant="ghost" size="sm" className="text-destructive hover:text-destructive" onClick={() => void remove(bot)}>
            Delete bot
          </Button>
        )}
      </span>
      {showSubmit && (
        <Button type="submit" size="sm" loading={submitting}>
          {submitLabel}
        </Button>
      )}
    </div>
  );
};
