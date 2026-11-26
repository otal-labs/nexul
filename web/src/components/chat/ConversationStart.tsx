import type { LucideIcon } from "lucide-react";

import type { Conversation } from "@nexul/client-core/chat";

import { PersonAvatar } from "@/components/PersonAvatar";
import { useDMPeer } from "@/hooks/useDMPeer";

interface ConversationStartProps {
  conversation: Conversation;
  label: string;
  icon: LucideIcon;
}

// Heads a history that is all loaded, at the stream's foot, so a short conversation sits above the composer rather than under the header.
export const ConversationStart = ({ conversation, label, icon: Icon }: ConversationStartProps) => {
  const peer = useDMPeer(conversation);
  const channel = conversation.kind === "channel" || conversation.kind === "voice_channel";
  const subject = channel ? `#${label}` : label;
  const sentence = peer ? `This is the start of your conversation with ${label}.` : `This is the start of ${subject}.`;
  return (
    <div className="mt-auto flex flex-col items-start gap-3 px-3 pt-8 pb-2">
      {peer && <PersonAvatar login={peer.login} name={peer.display_name} src={peer.avatar_url} label={label} className="size-12" />}
      {!peer && (
        <span className="flex size-12 items-center justify-center rounded-xl bg-foreground/5 text-muted-foreground">
          <Icon className="size-5" aria-hidden />
        </span>
      )}
      <div className="space-y-0.5">
        <p dir="auto" className="text-base font-semibold tracking-tight text-foreground">{subject}</p>
        <p className="text-sm text-muted-foreground">{sentence}</p>
      </div>
    </div>
  );
};
