import { View } from "react-native";

import { groupReplies, type Invitee, type Meeting } from "@nexul/client-core/calendar";

import { MeetingPerson } from "@/components/calendar/MeetingPerson";
import { SettingsCard } from "@/components/SettingsCard";
import { cn } from "@/lib/utils";

const InviteeRow = ({ invitee, workspaceId, first }: { invitee: Invitee; workspaceId: string; first: boolean }) => (
  <View className={cn("min-h-12 flex-row items-center gap-3 px-4 py-2", !first && "border-t border-border")}>
    <MeetingPerson userId={invitee.user_id} workspaceId={workspaceId} />
  </View>
);

const ReplyGroup = ({ title, invitees, workspaceId }: { title: string; invitees: Invitee[]; workspaceId: string }) => (
  <SettingsCard title={`${title} · ${invitees.length}`}>
    {invitees.map((invitee, index) => (
      <InviteeRow key={invitee.user_id} invitee={invitee} workspaceId={workspaceId} first={index === 0} />
    ))}
  </SettingsCard>
);

// The invitees by what they answered; a group nobody is in is left out.
export const MeetingRepliesSection = ({ meeting }: { meeting: Meeting }) => {
  const { accepted, declined, pending } = groupReplies(meeting);
  return (
    <>
      {accepted.length > 0 && <ReplyGroup title="Accepted" invitees={accepted} workspaceId={meeting.workspace_id} />}
      {declined.length > 0 && <ReplyGroup title="Declined" invitees={declined} workspaceId={meeting.workspace_id} />}
      {pending.length > 0 && <ReplyGroup title="No reply" invitees={pending} workspaceId={meeting.workspace_id} />}
    </>
  );
};
