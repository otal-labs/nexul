import ChevronRight from "lucide-react-native/icons/chevron-right";
import CircleCheck from "lucide-react-native/icons/circle-check";
import CircleDashed from "lucide-react-native/icons/circle-dashed";
import TriangleAlert from "lucide-react-native/icons/triangle-alert";
import { useState, type ReactNode } from "react";
import { Pressable, View } from "react-native";
import Animated, { interpolate, useAnimatedStyle } from "react-native-reanimated";
import { useCSSVariable } from "uniwind";

import { formatElapsed, isFailedStep, turnCounts, turnSpanSeconds, turnSummary } from "@nexul/client-core/agentSteps";

import { HandoffStepRow } from "@/components/chat/HandoffStepRow";
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

const headline = (running: boolean, seconds: number | null): string => {
  const verb = running ? "Working" : "Worked";
  if (seconds === null) return verb;
  return `${verb} for ${formatElapsed(seconds)}`;
};

// A result replaces its call (harness CallID), so a call left in a turn that stopped never came back.
const TurnGlyph = ({ entries, running }: { entries: HandoffStep[]; running: boolean }) => {
  if (running) return <Orbit />;
  if (entries.some(isFailedStep)) return <Icon as={TriangleAlert} aria-label="A step failed" className="size-4 text-destructive" />;
  if (entries.some((e) => e.kind === "tool_call")) return <Icon as={CircleDashed} aria-label="Unfinished" className="size-4 text-muted-foreground" />;
  return <Icon as={CircleCheck} aria-label="Done" className="size-4 text-success" />;
};

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

// One turn of the helper agent: a status glyph, "Worked for" and its step counts, folding open to the steps behind a rule.
// Open while it runs, folded once it ends; the parent keys it by running so the fold resets when the turn finishes.
export const HandoffTurnGroup = ({ entries, running, from, until }: HandoffTurnGroupProps) => {
  const [open, setOpen] = useState(running);
  const [muted] = useCSSVariable(["--color-muted-foreground"]);
  const title = headline(running, running ? null : turnSpanSeconds(entries, until, from));
  const summary = turnSummary(turnCounts(entries));
  return (
    <View>
      <Pressable
        role="button"
        aria-expanded={open}
        aria-label={`${title}, ${summary}`}
        onPress={() => setOpen(!open)}
        className="min-h-11 flex-row items-center gap-2.5 rounded-md px-1 active:bg-accent"
      >
        <TurnGlyph entries={entries} running={running} />
        <View className="min-w-0 flex-1">
          <Text className="text-sm">{title}</Text>
          <Text className="text-xs text-muted-foreground">{summary}</Text>
        </View>
        <ChevronRight color={String(muted)} size={14} style={{ transform: [{ rotate: open ? "90deg" : "0deg" }] }} />
      </Pressable>
      {open && entries.length > 0 && (
        <Reveal>
          <View className="mb-2 ml-3 mt-1 border-l border-border pl-4">
            {entries.map((step, i) => (
              <HandoffStepRow key={`${step.call_id ?? step.at}-${i}`} step={step} />
            ))}
          </View>
        </Reveal>
      )}
    </View>
  );
};
