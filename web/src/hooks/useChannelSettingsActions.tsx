import { toast } from "sonner";

import { errorMessage } from "@/api/client";
import { AddChannelPeopleForm } from "@/components/chat/AddChannelPeopleForm";
import { WhoStaysForm } from "@/components/chat/WhoStaysForm";
import { useLeaveChannel, useSetChannelPrivate } from "@/hooks/ChannelHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { useFormDialog } from "@/hooks/useFormDialog";
import {
  AddChannelPeopleFormSchema,
  ChannelPeopleFormSchema,
  channelMention,
  type ChannelPeopleFormData,
  type Conversation,
} from "@/models/Chat";

// The private switch, Add people, and Leave of a channel's settings, each behind its own dialog.
export const useChannelSettingsActions = (channel: Conversation) => {
  const { open: openForm } = useFormDialog();
  const { open: confirm } = useConfirmationDialog();
  const setPrivate = useSetChannelPrivate();
  const leaveChannel = useLeaveChannel();
  const label = channelMention(channel);

  const makePrivate = () =>
    openForm<ChannelPeopleFormData>({
      title: `Who stays in ${label}?`,
      description: "Anyone left unchecked loses access right away.",
      schema: ChannelPeopleFormSchema,
      okLabel: "Make private",
      form: <WhoStaysForm channel={channel} />,
      formOptions: { defaultValues: { user_ids: [] } },
    });

  const makePublic = async () => {
    const ok = await confirm({
      title: `Make ${label} public?`,
      message: "Everyone in the workspace will see it and read its whole history.",
      confirmLabel: "Make public",
    });
    if (!ok) return;
    setPrivate.mutate({ id: channel.id, isPrivate: false, memberIds: [] }, { onError: (error) => toast.error(errorMessage(error)) });
  };

  const addPeople = () =>
    openForm<ChannelPeopleFormData>({
      title: `Add people to ${label}`,
      schema: AddChannelPeopleFormSchema,
      okLabel: "Add people",
      form: <AddChannelPeopleForm channel={channel} />,
      formOptions: { defaultValues: { user_ids: [] } },
    });

  const leave = async () => {
    const ok = await confirm({ title: `Leave ${label}?`, message: "You stop seeing it until a member adds you back.", confirmLabel: "Leave" });
    if (ok) leaveChannel.mutate(channel);
  };

  return {
    setPrivate: (on: boolean) => void (on ? makePrivate() : makePublic()),
    addPeople: () => void addPeople(),
    leave: () => void leave(),
  };
};
