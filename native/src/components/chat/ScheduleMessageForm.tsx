import { useRouter } from "expo-router";
import { useState } from "react";
import { View } from "react-native";

import { deviceTimezone, nextScheduleSlot, type ScheduledMessage } from "@nexul/client-core/chat";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { ScheduleRepeatSwitch } from "@/components/chat/ScheduleRepeatSwitch";
import { ScheduleWhenFields } from "@/components/chat/ScheduleWhenFields";
import { Button } from "@/components/ui/button";
import { Text } from "@/components/ui/text";
import { Textarea } from "@/components/ui/textarea";
import { useScheduleMessage, useUpdateScheduledMessage } from "@/hooks/ScheduledMessageHooks";
import { useScheduleStore } from "@/stores/scheduleStore";

interface ScheduleMessageFormProps {
  conversationId: string;
  // body is the composer's text a new schedule opens on; editing is the scheduled message being changed.
  body: string;
  editing: ScheduledMessage | undefined;
}

export const ScheduleMessageForm = ({ conversationId, body: draft, editing }: ScheduleMessageFormProps) => {
  const router = useRouter();
  const [body, setBody] = useState(editing?.body ?? draft);
  const [at, setAt] = useState(() => (editing ? new Date(editing.send_at) : nextScheduleSlot()));
  const [repeats, setRepeats] = useState(editing?.repeats ?? "none");
  // Checked against when the sheet opened; the server refuses a time that has passed since.
  const [openedAt] = useState(() => Date.now());
  const schedule = useScheduleMessage(conversationId);
  const update = useUpdateScheduledMessage();
  const handOff = useScheduleStore((s) => s.handOff);
  const pending = schedule.isPending || update.isPending;
  const moved = !editing || at.getTime() !== Date.parse(editing.send_at);
  const past = moved && at.getTime() <= openedAt;
  const canSave = body.trim() !== "" && !past && !pending;
  const error = schedule.error ?? update.error;
  const saveLabel = editing ? "Save" : "Schedule";

  const save = () => {
    const input = { body: body.trim(), repeats, timezone: deviceTimezone() };
    if (editing) {
      // An unmoved time stays out, so a repeat keeps the day it steps from.
      update.mutate({ id: editing.id, change: moved ? { ...input, send_at: at.toISOString() } : input }, { onSuccess: () => router.back() });
      return;
    }
    schedule.mutate({ ...input, send_at: at.toISOString() }, {
      onSuccess: () => {
        handOff(conversationId);
        router.back();
      },
    });
  };

  return (
    <View className="gap-4">
      <Textarea value={body} onChangeText={setBody} placeholder="Message" aria-label="Message" numberOfLines={6} autoFocus={!editing} />
      <ScheduleWhenFields at={at} onChange={setAt} />
      {past && <Text className="text-row text-destructive">Pick a time ahead.</Text>}
      <ScheduleRepeatSwitch value={repeats} onChange={setRepeats} />
      {error && <ErrorDisplay error={error} className="px-0" />}
      <Button disabled={!canSave} onPress={save}>
        <Text>{pending ? "Saving…" : saveLabel}</Text>
      </Button>
      <Button variant="outline" disabled={pending} onPress={() => router.back()}>
        <Text>Cancel</Text>
      </Button>
    </View>
  );
};
