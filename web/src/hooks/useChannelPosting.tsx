import type { Conversation } from "@nexul/client-core/chat";

import { useFetchExplanation } from "@/hooks/PermissionHooks";
import type { PermissionSource } from "@/models/Permission";

// What keeps the viewer from posting in a channel, or undefined: elsewhere, and until the answer lands, the composer stays.
export const useChannelPostRefusal = (conversation: Conversation): PermissionSource | undefined => {
  const channel = conversation.kind === "channel" || conversation.kind === "voice_channel";
  const { data } = useFetchExplanation("channel", conversation.id, "", channel);
  const post = data?.effective.find((answer) => answer.action === "channels:post");
  return post && !post.allowed ? post.source : undefined;
};
