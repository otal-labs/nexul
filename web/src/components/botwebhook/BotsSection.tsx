import { useState } from "react";
import { Plus } from "lucide-react";

import { BotForm, type BotView } from "@/components/botwebhook/BotForm";
import { BotsFeed } from "@/components/botwebhook/BotsFeed";
import { DeletedBotsFold } from "@/components/botwebhook/DeletedBotsFold";
import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { Button } from "@/components/ui/button";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useFetchBotwebhooks } from "@/hooks/BotwebhookHooks";
import { BOT_CAP, capNote } from "@/models/Botwebhook";
import type { Conversation } from "@/models/Chat";
import { cn } from "@/lib/utils";

interface BotsSectionProps {
  conversation: Conversation;
  className?: string;
}

// The list, or one bot's detail view in its place; only an editor opens a bot, creates one, or sees the deleted ones.
export const BotsSection = ({ conversation, className }: BotsSectionProps) => {
  const can = useAreaAccess();
  const editor = can?.("editBots") ?? false;
  const { data: bots, error, isPending } = useFetchBotwebhooks(conversation.id);
  const [view, setView] = useState<BotView | null>(null);
  const creating = editor && view?.id === "new";
  const open = editor ? bots?.find((bot) => bot.id === view?.id) : undefined;
  const full = (bots?.length ?? 0) >= BOT_CAP;

  return (
    <div className={cn("min-w-0", className)}>
      {(creating || open) && (
        <BotForm key={view?.id} conversationId={conversation.id} bot={open} note={view?.note} onNavigate={setView} />
      )}
      {!creating && !open && (
        <section aria-labelledby="bots-heading" className="space-y-2.5">
          <div className="flex items-baseline justify-between gap-2">
            <h3 id="bots-heading" className="text-sm font-medium">
              Bots
            </h3>
            {bots && (
              <span className="font-mono text-xs text-muted-foreground tabular-nums">
                {bots.length} of {BOT_CAP}
              </span>
            )}
          </div>
          <p className="text-sm text-muted-foreground">
            Outside tools post here through a webhook URL. Anything that already posts to a Discord webhook works.
          </p>
          {isPending && <LoadingDisplay label="Loading bots…" className="p-4" />}
          {error && <ErrorDisplay error={error} title="Failed to load bots." className="p-4" />}
          {bots && bots.length === 0 && <EmptyRow>No bots yet</EmptyRow>}
          {bots && bots.length > 0 && <BotsFeed bots={bots} onOpen={editor ? (bot) => setView({ id: bot.id }) : undefined} />}
          {editor && bots && (
            <div className="space-y-1.5">
              <Button type="button" variant="outline" size="sm" className="w-full" disabled={full} onClick={() => setView({ id: "new" })}>
                <Plus className="size-4" aria-hidden />
                New bot
              </Button>
              {full && <p className="text-xs text-muted-foreground">{capNote(conversation.kind)}</p>}
            </div>
          )}
          {editor && (
            <DeletedBotsFold
              conversationId={conversation.id}
              full={full}
              onRestored={(bot) => setView({ id: bot.id, note: "Restored with a new URL." })}
            />
          )}
        </section>
      )}
    </div>
  );
};
