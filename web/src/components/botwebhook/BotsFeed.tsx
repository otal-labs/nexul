import { BotRow } from "@/components/botwebhook/BotRow";
import type { Botwebhook } from "@/models/Botwebhook";

interface BotsFeedProps {
  bots: Botwebhook[];
  onOpen?: ((bot: Botwebhook) => void) | undefined;
}

export const BotsFeed = ({ bots, onOpen }: BotsFeedProps) => (
  <ul aria-label="Bots" className="space-y-1.5">
    {bots.map((bot) => (
      <BotRow key={bot.id} bot={bot} onOpen={onOpen && (() => onOpen(bot))} />
    ))}
  </ul>
);
