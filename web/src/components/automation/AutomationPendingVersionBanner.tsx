import { Clock } from "lucide-react";
import { Link } from "react-router";

import { useFetchAutomationVersionDiff } from "@/hooks/AutomationVersionHooks";

interface AutomationPendingVersionBannerProps {
  automationId: string;
}

// A push always lands pending, never activating itself — this surfaces
// that there's a diff waiting for a decision without leaving the Overview tab.
export const AutomationPendingVersionBanner = ({ automationId }: AutomationPendingVersionBannerProps) => {
  const { data } = useFetchAutomationVersionDiff(automationId);
  if (!data?.pending) return null;

  return (
    <Link
      to={{ search: "?tab=versions" }}
      className="flex items-center gap-2 rounded-md border border-warning/30 bg-warning/10 px-4 py-3 text-sm text-warning transition-colors duration-150 ease-standard hover:bg-warning/15"
    >
      <Clock className="size-4 shrink-0" aria-hidden />
      A new version is pending review — see the diff on the Versions tab.
    </Link>
  );
};
