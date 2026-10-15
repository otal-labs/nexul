import CircleQuestionMark from "lucide-react-native/icons/circle-question-mark";
import Dot from "lucide-react-native/icons/dot";
import FileText from "lucide-react-native/icons/file-text";
import Terminal from "lucide-react-native/icons/terminal";
import Wrench from "lucide-react-native/icons/wrench";
import type { LucideIcon } from "lucide-react-native";
import { View } from "react-native";

import { isCommandTool, isFailedStep, isFileTool, isToolStep, stepLabel } from "@nexul/client-core/agentSteps";

import { Icon } from "@/components/ui/icon";
import { Text } from "@/components/ui/text";
import { cn } from "@/lib/utils";
import type { HandoffStep } from "@/models/Chat";

interface HandoffStepRowProps {
  step: HandoffStep;
}

// The web's step icons: a terminal for a command, a file for a file tool, a wrench for any other tool.
const stepIcon = (step: HandoffStep): LucideIcon => {
  if (isToolStep(step) && isCommandTool(step.tool)) return Terminal;
  if (isToolStep(step) && isFileTool(step.tool)) return FileText;
  if (isToolStep(step)) return Wrench;
  if (step.kind === "question") return CircleQuestionMark;
  return Dot;
};

export const HandoffStepRow = ({ step }: HandoffStepRowProps) => {
  const failed = isFailedStep(step);
  const label = stepLabel(step);
  return (
    <View accessible aria-label={failed ? `${label}, failed` : label} className="flex-row items-start gap-2 py-1">
      <Icon as={stepIcon(step)} className={cn("mt-px size-3.5 text-muted-foreground", failed && "text-destructive")} />
      <Text
        numberOfLines={2}
        className={cn("min-w-0 flex-1 font-mono text-xs text-muted-foreground", step.kind === "note" && "font-sans italic")}
      >
        {label}
      </Text>
    </View>
  );
};
