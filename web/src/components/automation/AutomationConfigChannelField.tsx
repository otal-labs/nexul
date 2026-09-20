import { Controller, type Control } from "react-hook-form";

import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useFetchConversations } from "@/hooks/ChatHooks";
import { conversationLabel } from "@/models/Chat";
import type { AutomationConfigField } from "@/models/Automation";
import { useWorkspaceStore } from "@/stores/workspaceStore";

interface AutomationConfigChannelFieldProps {
  control: Control<Record<string, string>>;
  field: AutomationConfigField;
}

// Real control for a "channel" config knob: a dropdown of this
// workspace's chat channels — the channel source this instance already has.
export const AutomationConfigChannelField = ({ control, field }: AutomationConfigChannelFieldProps) => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: conversations } = useFetchConversations(workspaceId);
  const channels = (conversations ?? []).filter((c) => c.kind === "channel");

  return (
    <Controller
      control={control}
      name={field.key}
      render={({ field: rhf }) => (
        <Select value={rhf.value} onValueChange={rhf.onChange}>
          <SelectTrigger id={field.key}>
            <SelectValue placeholder="Select a channel" />
          </SelectTrigger>
          <SelectContent>
            {channels.map((channel) => (
              <SelectItem key={channel.id} value={channel.id}>
                {conversationLabel(channel)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      )}
    />
  );
};
