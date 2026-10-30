import { ChevronRight } from "lucide-react";

import { personLabel } from "@nexul/client-core/person";

import { BotAvatar } from "@/components/botwebhook/BotAvatar";
import { usePerson } from "@/hooks/PeopleHooks";
import { botActivityLine, type Botwebhook } from "@/models/Botwebhook";

interface BotRowProps {
  bot: Botwebhook;
  /** Opens the bot's detail view; a viewer who can't edit bots gets the row alone. */
  onOpen?: (() => void) | undefined;
}

export const BotRow = ({ bot, onOpen }: BotRowProps) => {
  const creator = personLabel(usePerson(bot.created_by));
  const meta = bot.stopped ?? `${botActivityLine(bot)} · made by ${creator}`;
  const content = (
    <>
      <BotAvatar avatar={bot.avatar} />
      <span className="min-w-0 flex-1">
        <span className="block truncate text-sm font-medium" title={bot.name}>
          {bot.name}
        </span>
        <span className="flex min-w-0 items-center gap-1.5 text-xs text-muted-foreground" title={meta}>
          {bot.stopped && <span className="size-1.5 shrink-0 rounded-full bg-warning-mark" aria-hidden />}
          <span className="truncate">{meta}</span>
        </span>
      </span>
    </>
  );

  return (
    <li className="rounded-lg border border-border bg-card">
      {onOpen && (
        <button
          type="button"
          onClick={onOpen}
          className="flex w-full min-w-0 items-center gap-3 rounded-lg py-2.5 pr-2 pl-3 text-left transition-colors duration-150 ease-standard hover:bg-accent focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none"
        >
          {content}
          <ChevronRight className="size-4 shrink-0 text-muted-foreground" aria-hidden />
        </button>
      )}
      {!onOpen && <div className="flex min-w-0 items-center gap-3 py-2.5 pr-2 pl-3">{content}</div>}
    </li>
  );
};
