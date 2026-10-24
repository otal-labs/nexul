import { FormInput } from "@/components/FormInput";
import { useFormDialogContext } from "@/components/dialogs/FormDialogContext";
import { FormSelect } from "@/components/ticket/FormSelect";
import { useFetchConversations } from "@/hooks/ChatHooks";
import { WHERE_OPTIONS, type MeetingFormData } from "@/models/Meeting";
import { groupConversations } from "@/models/Chat";
import { useWorkspaceStore } from "@/stores/workspaceStore";

// Where it takes place: nowhere, one of the workspace's voice channels, or a link. Only the picked kind is sent.
export const MeetingWhereFields = () => {
  const { control, watch } = useFormDialogContext<MeetingFormData>();
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: conversations } = useFetchConversations(workspaceId);
  const where = watch("where");
  const channels = groupConversations(conversations ?? []).voiceChannels.map((c) => ({ value: c.id, label: c.name ?? "Voice channel" }));
  return (
    <div className="grid grid-cols-2 items-start gap-3">
      <FormSelect control={control} name="where" label="Where" options={WHERE_OPTIONS} />
      {where === "channel" && <FormSelect control={control} name="voice_channel_id" label="Channel" options={channels} placeholder="Pick one…" />}
      {where === "link" && <FormInput control={control} name="link" type="url" label="Link" placeholder="https://" autoComplete="off" />}
    </div>
  );
};
