import ExternalLink from "lucide-react-native/icons/external-link";
import { Linking, Pressable, View } from "react-native";
import { useCSSVariable } from "uniwind";

import { isCancelled, REPEAT_LABELS, type Meeting } from "@nexul/client-core/calendar";

import { MeetingPerson } from "@/components/calendar/MeetingPerson";
import { FactRow } from "@/components/FactRow";
import { Text } from "@/components/ui/text";
import { useMeetingPlace } from "@/hooks/CalendarHooks";
import { formatDayLabel } from "@/lib/time";

const Place = ({ meeting }: { meeting: Meeting }) => {
  const place = useMeetingPlace(meeting);
  const [muted] = useCSSVariable(["--color-muted-foreground"]);
  const url = place?.url;
  return (
    <>
      {!place && <Text className="text-sm text-muted-foreground">None</Text>}
      {place && !url && (
        <Text numberOfLines={1} className="shrink text-sm">
          {place.label}
        </Text>
      )}
      {place && url && (
        <Pressable role="link" aria-label={`Open ${place.label}`} onPress={() => void Linking.openURL(url)} className="min-h-11 shrink flex-row items-center justify-end gap-1.5">
          <Text numberOfLines={1} className="shrink text-sm">
            {place.label}
          </Text>
          <ExternalLink size={14} color={String(muted)} />
        </Pressable>
      )}
    </>
  );
};

const repeatText = (meeting: Meeting) =>
  meeting.repeat_until ? `${REPEAT_LABELS[meeting.repeats]} until ${formatDayLabel(meeting.repeat_until)}` : REPEAT_LABELS[meeting.repeats];

// What and where, as the card a ticket's facts sit in; the time is in the header, in the phone's own zone.
export const MeetingFacts = ({ meeting }: { meeting: Meeting }) => {
  const cancelled = isCancelled(meeting);
  return (
    <View className="overflow-hidden rounded-xl border border-border bg-card">
      {cancelled && (
        <FactRow label="Status" first>
          <View className="size-2 rounded-full bg-destructive" />
          <Text className="text-sm">Cancelled</Text>
        </FactRow>
      )}
      <FactRow label="Repeats" first={!cancelled}>
        <Text className="text-sm">{repeatText(meeting)}</Text>
      </FactRow>
      <FactRow label="Where">
        <Place meeting={meeting} />
      </FactRow>
      <FactRow label="Organiser">
        <MeetingPerson userId={meeting.organiser_id} workspaceId={meeting.workspace_id} />
      </FactRow>
    </View>
  );
};
