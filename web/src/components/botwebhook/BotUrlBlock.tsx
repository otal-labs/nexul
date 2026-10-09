import { personLabel } from "@nexul/client-core/person";

import { BotUrlField } from "@/components/botwebhook/BotUrlField";
import { usePerson } from "@/hooks/PeopleHooks";
import { botActivityLine, type Botwebhook } from "@/models/Botwebhook";
import { formatRelativeTime } from "@/utils/TimeUtility";

interface BotUrlBlockProps {
  bot: Botwebhook;
  /** Says what a regenerate or restore just did; nothing shows under the URL otherwise. */
  note?: string | undefined;
}

export const BotUrlBlock = ({ bot, note }: BotUrlBlockProps) => {
  const madeBy = personLabel(usePerson(bot.created_by));
  return (
    <>
      <div className="space-y-1.5">
        <p className="text-sm font-medium">Webhook URL</p>
        {bot.url && <BotUrlField name={bot.name} url={bot.url} />}
        {note && (
          <p role="status" className="text-xs text-muted-foreground">
            {note}
          </p>
        )}
      </div>
      <p className="text-xs break-words text-muted-foreground">
        {botActivityLine(bot)} · made by {madeBy} · created {formatRelativeTime(bot.created_at)}
      </p>
    </>
  );
};
