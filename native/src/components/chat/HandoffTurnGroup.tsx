import ChevronRight from "lucide-react-native/icons/chevron-right";
import CircleCheck from "lucide-react-native/icons/circle-check";
import CircleDashed from "lucide-react-native/icons/circle-dashed";
import TriangleAlert from "lucide-react-native/icons/triangle-alert";
import { useState, type ReactNode } from "react";
import { Pressable, View } from "react-native";
import Animated, { interpolate, useAnimatedStyle } from "react-native-reanimated";
import { useCSSVariable } from "uniwind";

import { formatElapsed, groupSteps, liveStepLabel, runDirectory, turnOutcome, turnPhrase, turnSpanSeconds, type TurnOutcome } from "@nexul/client-core/agentSteps";

import { HandoffStepGroup } from "@/components/chat/HandoffStepGroup";
import { Orbit } from "@/components/LoadingDisplay";
import { Icon } from "@/components/ui/icon";
import { Text } from "@/components/ui/text";
import { useEntrance, useReducedMotion } from "@/lib/motion";
import type { HandoffStep } from "@/models/Chat";

interface HandoffTurnGroupProps {
  entries: HandoffStep[];
  running: boolean;
  from: string | null;
  until: string | null;
}

const worked = (seconds: number | null): string => (seconds === null ? "Worked" : `Worked for ${formatElapsed(seconds)}`);

const sentence = (phrase: string): string => `${phrase.charAt(0).toUpperCase()}${phrase.slice(1)}`;

// The web's outcome mark: only a failed step takes a status hue.
const TurnGlyph = ({ outcome }: { outcome: TurnOutcome }) => (
  <>
    {outcome === "running" && <Orbit />}
    {outcome === "failed" && <Icon as={TriangleAlert} aria-label="A step failed" className="size-4 text-destructive" />}
    {outcome === "unfinished" && <Icon as={CircleDashed} aria-label="Unfinished" className="size-4 text-muted-foreground" />}
    {outcome === "done" && <Icon as={CircleCheck} aria-label="Done" className="size-4 text-muted-foreground" />}
  </>
);

// The opened steps fade and rise 4px over 200ms, the compact empty state's entrance; a 150ms fade under reduced motion.
const Reveal = ({ children }: { children: ReactNode }) => {
  const reduced = useReducedMotion();
  const progress = useEntrance(reduced ? 150 : 200);
  const style = useAnimatedStyle(() => ({
    opacity: progress.get(),
    transform: [{ translateY: reduced ? 0 : interpolate(progress.get(), [0, 1], [4, 0]) }],
  }));
  return <Animated.View style={style}>{children}</Animated.View>;
};

// One turn of the helper agent: its outcome mark, the step in flight or "Worked for", and what it did, folding open to
// the steps on a rule dropped from the mark. Open while it runs, folded once it ends; the parent keys it by running so
// the fold resets when the turn finishes.
export const HandoffTurnGroup = ({ entries, running, from, until }: HandoffTurnGroupProps) => {
  const [open, setOpen] = useState(running);
  const [muted] = useCSSVariable(["--color-muted-foreground"]);
  const title = running ? liveStepLabel(entries) : worked(turnSpanSeconds(entries, until, from));
  const phrase = sentence(turnPhrase(entries));
  const root = runDirectory(entries);
  return (
    <View>
      <Pressable
        role="button"
        aria-expanded={open}
        aria-label={phrase === "" ? title : `${title}, ${phrase}`}
        onPress={() => setOpen(!open)}
        className="min-h-11 flex-row items-center gap-2.5 rounded-md px-1 active:bg-accent"
      >
        <TurnGlyph outcome={turnOutcome(entries, running)} />
        <View className="min-w-0 flex-1">
          <Text numberOfLines={1} className={running ? "text-sm text-muted-foreground" : "text-sm font-medium"}>
            {title}
          </Text>
          {phrase !== "" && <Text className="text-xs text-muted-foreground">{phrase}</Text>}
        </View>
        <ChevronRight color={String(muted)} size={14} style={{ transform: [{ rotate: open ? "90deg" : "0deg" }] }} />
      </Pressable>
      {open && entries.length > 0 && (
        <Reveal>
          <View className="mb-2 ml-[11px] mt-1 border-l border-border pl-[18px]">
            {groupSteps(entries).map((group, i) => (
              <HandoffStepGroup key={`${group.act}-${i}`} group={group} root={root} running={running} />
            ))}
          </View>
        </Reveal>
      )}
    </View>
  );
};
