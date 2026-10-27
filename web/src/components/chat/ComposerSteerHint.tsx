import { queueShortcut } from "@/lib/shortcut";
import { useAgentStreamStore } from "@/stores/agentStreamStore";

interface ComposerSteerHintProps {
  conversationId: string;
}

// While the Agent works in the thread a sent message steers into its turn; the shortcut queues it behind instead.
export const ComposerSteerHint = ({ conversationId }: ComposerSteerHintProps) => {
  const working = useAgentStreamStore((s) => s.streams[conversationId]?.streaming === true);

  return (
    working && (
      <p data-slot="steer-hint" className="settle-in px-1 pt-1.5 text-xs text-muted-foreground">
        Enter steers the Agent ·{" "}
        <kbd className="rounded-sm border border-border px-1.5 py-px font-mono text-[11px]">{queueShortcut()}</kbd> queues
      </p>
    )
  );
};
