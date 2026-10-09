import { Sparkles, User } from "lucide-react";

import { NoFillBadge } from "@/components/ui/badge";
import { AutomationKind } from "@/enums/Automation";

interface AutomationKindBadgeProps {
  kind: AutomationKind;
}

// Kind is a tag, not a status, so both read as a muted icon and text.
export const AutomationKindBadge = ({ kind }: AutomationKindBadgeProps) => {
  if (kind === AutomationKind.Default) {
    return (
      <NoFillBadge icon={Sparkles} color="text-muted-foreground">
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
