import { BotAvatar } from "@/components/botwebhook/BotAvatar";
import { Button } from "@/components/ui/button";
import { usePerson } from "@/hooks/PeopleHooks";
import type { Botwebhook } from "@/models/Botwebhook";
import { personLabel } from "@/models/Person";
import { formatRelativeTime } from "@/utils/TimeUtility";

interface DeletedBotRowProps {
  bot: Botwebhook;
  /** Off while the conversation already holds its ten bots. */
  full: boolean;
  onRestore: () => void;
}

export const DeletedBotRow = ({ bot, full, onRestore }: DeletedBotRowProps) => {
  const deletedBy = personLabel(usePerson(bot.deleted_by ?? ""));
  return (
    <li className="flex items-center gap-3 rounded-lg border border-dashed border-border px-3 py-2">
      <BotAvatar avatar={bot.avatar} className="size-6 opacity-60" />
      <span className="min-w-0 flex-1">
        <span className="block truncate text-sm text-muted-foreground" title={bot.name}>
          {bot.name}
        </span>
        <span className="block truncate text-xs text-muted-foreground">
          Deleted {formatRelativeTime(bot.deleted_at ?? bot.updated_at)} by {deletedBy}
        </span>
      </span>
      <Button
        type="button"
        variant="ghost"
        size="sm"
        disabled={full}
        title={full ? "Delete a bot to make room for this one" : undefined}
        aria-label={`Restore ${bot.name}`}
        onClick={onRestore}
      >
        Restore
      </Button>
    </li>
  );
};
