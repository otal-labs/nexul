import { KeyRoundIcon } from "lucide-react";

import { secretManagerName } from "@nexul/client-core/secretReference";

import { NoFillBadge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import type { ManagerState } from "@/models/Runner";

const states = {
  ready: { color: "text-success", suffix: "" },
  "token rejected": { color: "text-destructive", suffix: " token rejected" },
  unreachable: { color: "text-warning", suffix: " unreachable" },
} as const;

interface RunnerManagerTagProps {
  manager: ManagerState;
  className?: string;
}

// What a runner can read: the manager's name in the hue of its state, "1Password token rejected" or "1Password unreachable" when it cannot.
export const RunnerManagerTag = ({ manager, className }: RunnerManagerTagProps) => {
  const { color, suffix } = states[manager.state];
  const name = secretManagerName(manager.manager);
  return (
    <NoFillBadge icon={KeyRoundIcon} color={color} className={cn("shrink-0", className)}>
      {name}
      {suffix}
    </NoFillBadge>
  );
};
