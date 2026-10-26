import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useFetchBotwebhooks } from "@/hooks/BotwebhookHooks";

const SYSTEM = "system";

interface StackNoticeBotSelectProps {
  conversationId: string;
  value: string;
  disabled: boolean;
  onChange: (botId: string) => void;
}

// A viewer who can't read the channel's bots still gets the system choice; a deleted bot stays named until it is changed.
export const StackNoticeBotSelect = ({ conversationId, value, disabled, onChange }: StackNoticeBotSelectProps) => {
  const { data: bots } = useFetchBotwebhooks(conversationId);
  const gone = value !== "" && bots !== undefined && !bots.some((b) => b.id === value);
  return (
    <Select value={value || SYSTEM} onValueChange={(id) => onChange(id === SYSTEM ? "" : id)} disabled={disabled}>
      <SelectTrigger id="crash-notice-bot" className="w-full">
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        <SelectItem value={SYSTEM}>System message</SelectItem>
        {gone && <SelectItem value={value}>A deleted bot · posts nothing</SelectItem>}
        {(bots ?? []).map((bot) => (
          <SelectItem key={bot.id} value={bot.id}>
            {bot.name}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
};
