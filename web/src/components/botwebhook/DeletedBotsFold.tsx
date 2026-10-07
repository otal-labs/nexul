import { useState } from "react";
import { ChevronRight } from "lucide-react";

import { DeletedBotRow } from "@/components/botwebhook/DeletedBotRow";
import { useFetchBotwebhooks } from "@/hooks/BotwebhookHooks";
import { useBotActions } from "@/hooks/useBotActions";
import type { Botwebhook } from "@/models/Botwebhook";
import { cn } from "@/lib/utils";

interface DeletedBotsFoldProps {
  conversationId: string;
  full: boolean;
  onRestored: (bot: Botwebhook) => void;
}

// Only an editor mounts it: the deleted list needs botwebhook:write.
export const DeletedBotsFold = ({ conversationId, full, onRestored }: DeletedBotsFoldProps) => {
  const [open, setOpen] = useState(false);
  const { data: deleted } = useFetchBotwebhooks(conversationId, true);
  const { restore } = useBotActions();
  if (!deleted || deleted.length === 0) return null;

  const onRestore = async (bot: Botwebhook) => {
    if (await restore(bot)) onRestored(bot);
  };

  return (
    <div className="space-y-1.5">
      <button
        type="button"
        onClick={() => setOpen(!open)}
        aria-expanded={open}
        className="flex items-center gap-1 rounded-md text-xs text-muted-foreground hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none"
      >
        <ChevronRight className={cn("size-3.5 transition-transform duration-150 ease-standard", open && "rotate-90")} aria-hidden />
        Deleted ({deleted.length})
      </button>
      {open && (
        <ul aria-label="Deleted bots" className="space-y-1.5">
          {deleted.map((bot) => (
            <DeletedBotRow key={bot.id} bot={bot} full={full} onRestore={() => void onRestore(bot)} />
          ))}
        </ul>
      )}
    </div>
  );
};
