import { AlertTriangle } from "lucide-react";

import { NoFillBadge } from "@/components/ui/badge";
import { automationNeedsConfiguration } from "@/models/Automation";
import type { Automation } from "@/models/Automation";

interface AutomationConfigStatusBadgeProps {
  automation: Automation;
}

// "Needs configuration" is distinct from deliberately off; never renders for a merely-disabled automation.
export const AutomationConfigStatusBadge = ({ automation }: AutomationConfigStatusBadgeProps) => {
  if (!automationNeedsConfiguration(automation)) return null;
  return (
    <NoFillBadge icon={AlertTriangle} color="text-warning">
      Needs configuration
    </NoFillBadge>
  );
};
