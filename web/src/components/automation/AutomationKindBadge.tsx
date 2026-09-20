import { Sparkles, User } from "lucide-react";

import { NoFillBadge } from "@/components/ui/badge";
import { AutomationKind } from "@/enums/Automation";

interface AutomationKindBadgeProps {
  kind: AutomationKind;
}

// Bounded enum (Default/Custom) → colored icon + text, per the Mono Console
// badge convention (the Mono Console spec) — never a filled chip.
export const AutomationKindBadge = ({ kind }: AutomationKindBadgeProps) => {
  if (kind === AutomationKind.Default) {
    return (
      <NoFillBadge icon={Sparkles} color="text-info">
        Default
      </NoFillBadge>
    );
  }
  return (
    <NoFillBadge icon={User} color="text-muted-foreground">
      Custom
    </NoFillBadge>
  );
};
