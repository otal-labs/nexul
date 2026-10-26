import { MessageSquareIcon } from "lucide-react";

import { conversationLabel } from "@nexul/client-core/chat";

import { SettingsCard } from "@/components/settings/SettingsCard";
import { SettingsRow, SettingsRows } from "@/components/settings/SettingsRow";
import { StackNoticeBotSelect } from "@/components/stack/StackNoticeBotSelect";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useFetchConversations } from "@/hooks/ChatHooks";
import { useUpdateStack } from "@/hooks/StackHooks";
import { useActionState } from "@/hooks/useActionState";
import { useStackActions } from "@/hooks/useStackActions";
import type { Stack } from "@/models/Stack";
import { useWorkspaceStore } from "@/stores/workspaceStore";

const NOWHERE = "none";

interface StackCrashNoticesCardProps {
  stack: Stack;
  // Why the stack can't be changed here (no project, withheld variables), shown in place of the hint.
  off: string | undefined;
}

// Picks apply at once, like the crash ticket switch beside it; a new channel starts posting as the system.
export const StackCrashNoticesCard = ({ stack, off }: StackCrashNoticesCardProps) => {
  const answer = useStackActions(stack);
  const update = useUpdateStack();
  const action = useActionState({ permission: "stacks:write", projectId: stack.project_id || undefined, answer: answer("stacks:write"), verb: "change this stack's alerts" });
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: conversations } = useFetchConversations(workspaceId);
  const channels = (conversations ?? []).filter((c) => c.kind === "channel");
  const current = stack.crash_notice_conversation_id ?? "";
  const unseen = current !== "" && conversations !== undefined && !channels.some((c) => c.id === current);
  const disabled = !!off || action.state !== "live" || update.isPending;
  const reason = action.state === "locked" ? action.reason : undefined;

  const aim = (conversationId: string, botId = "") =>
    update.mutate({ ...stack, crash_notice_conversation_id: conversationId, crash_notice_bot_id: botId });

  return (
    <SettingsCard
      id="crash-notices"
      title="Crash notices"
      icon={MessageSquareIcon}
      description={off ?? reason ?? "Post each crash, error-rate alert and recovery in a channel, with a link to its crash ticket. A restart loop is one post."}
    >
      <SettingsRows>
        <SettingsRow label="Channel" htmlFor="crash-notice-channel">
          <Select value={current || NOWHERE} onValueChange={(id) => aim(id === NOWHERE ? "" : id)} disabled={disabled}>
            <SelectTrigger id="crash-notice-channel" className="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={NOWHERE}>Off</SelectItem>
              {unseen && <SelectItem value={current}>A channel you can't see</SelectItem>}
              {channels.map((channel) => (
                <SelectItem key={channel.id} value={channel.id}>
                  #{conversationLabel(channel)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </SettingsRow>
        {current !== "" && (
          <SettingsRow label="Post as" description="A bot of the channel, or a system message" htmlFor="crash-notice-bot">
            <StackNoticeBotSelect
              conversationId={current}
              value={stack.crash_notice_bot_id ?? ""}
              disabled={disabled}
              onChange={(botId) => aim(current, botId)}
            />
          </SettingsRow>
        )}
      </SettingsRows>
    </SettingsCard>
  );
};
