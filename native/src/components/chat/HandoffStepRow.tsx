import { View } from "react-native";

import { Text } from "@/components/ui/text";
import { cn } from "@/lib/utils";
import type { HandoffStep } from "@/models/Chat";

interface HandoffStepRowProps {
  step: HandoffStep;
}

export const HandoffStepRow = ({ step }: HandoffStepRowProps) => (
  <View className="flex-row items-baseline gap-2 border-b border-border py-2">
    {!!step.tool && <Text className="font-mono text-xs text-muted-foreground">{step.tool}</Text>}
    <Text numberOfLines={2} className={cn("min-w-0 flex-1 text-sm", step.kind === "note" && "italic text-muted-foreground")}>
      {step.summary}
    </Text>
  </View>
);
