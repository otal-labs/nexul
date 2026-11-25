import { useRouter } from "expo-router";
import { useState } from "react";
import { View } from "react-native";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { Microheader } from "@/components/Microheader";
import { Button } from "@/components/ui/button";
import { Text } from "@/components/ui/text";
import { Textarea } from "@/components/ui/textarea";
import { useContinueRun } from "@/hooks/TrailHooks";
import { runRoute, type RunTab } from "@/models/Play";
import type { Trail } from "@/models/Trail";

// Sends one message as the ended run's next turn, without the play's instructions again (ADR 0128); a gone thread
// starts the play again as a new run, which then opens in this one's place.
export const RunContinueForm = ({ trail, tab }: { trail: Trail; tab: RunTab }) => {
  const router = useRouter();
  const [message, setMessage] = useState("");
  const send = useContinueRun();

  const submit = () =>
    send.mutate(
      { trailId: trail.id, message: message.trim() },
      {
        onSuccess: (next) => {
          setMessage("");
          if (next.id !== trail.id) router.replace(runRoute(tab, next.id));
        },
      },
    );

  return (
    <View className="gap-2">
      <Microheader>Continue this run</Microheader>
      <Textarea aria-label="What the agent should do next" placeholder="What should it do next?" value={message} onChangeText={setMessage} numberOfLines={5} className="text-body" />
      {send.error && <ErrorDisplay error={send.error} className="px-0" />}
      <Button disabled={message.trim() === "" || send.isPending} onPress={submit} className="self-end px-5">
        <Text>{send.isPending ? "Sending…" : "Send"}</Text>
      </Button>
    </View>
  );
};
