import { useLocalSearchParams, useRouter } from "expo-router";
import { useState } from "react";

import { SheetBody } from "@/components/SheetBody";
import { isNeedsLocation } from "@/api/errors";
import { ActionButton } from "@/components/action/ActionButton";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { SheetTitle } from "@/components/SheetTitle";
import { Text } from "@/components/ui/text";
import { Textarea } from "@/components/ui/textarea";
import { useRunPlay } from "@/hooks/TrailHooks";
import { runRoute, type PlayType } from "@/models/Play";

// Starts the play on this ticket or doc with this run's own instructions, then opens the run to watch it.
export const RunPlaySheet = () => {
  const router = useRouter();
  const { playId, label, targetType, targetId, projectId } = useLocalSearchParams<{
    playId: string;
    label: string;
    targetType: PlayType;
    targetId: string;
    projectId: string;
  }>();
  const [instructions, setInstructions] = useState("");
  const run = useRunPlay();

  const start = () =>
    run.mutate(
      { playId, input: { target_type: targetType, target_id: targetId, custom_instructions: instructions.trim() } },
      {
        onSuccess: (trail) => {
          router.back();
          router.push(runRoute(targetType === "ticket" ? "board" : "more", trail.id));
        },
      },
    );

  return (
    <SheetBody className="gap-4 px-5">
      <SheetTitle title={`Run ${label}`} className="px-0 pb-0" />
      <Textarea
        aria-label="Instructions for this run"
        placeholder="Instructions for this run (optional)"
        value={instructions}
        onChangeText={setInstructions}
        numberOfLines={6}
        className="text-body"
      />
      {run.error && <ErrorDisplay error={run.error} className="px-0" />}
      {isNeedsLocation(run.error) && (
        <Text className="text-row text-muted-foreground">Pick the computer on the web once, then runs from here go there too.</Text>
      )}
      <ActionButton permission="plays:run" projectId={projectId} verb="run plays" name={`Run ${label}`} disabled={run.isPending} onPress={start}>
        <Text>{run.isPending ? "Starting…" : "Run"}</Text>
      </ActionButton>
    </SheetBody>
  );
};
