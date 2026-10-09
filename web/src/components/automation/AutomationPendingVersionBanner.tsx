import { Clock } from "lucide-react";
import { Link } from "react-router";

import { useFetchAutomationVersionDiff } from "@/hooks/AutomationVersionHooks";
import { useTabPath } from "@/hooks/useTabPath";

interface AutomationPendingVersionBannerProps {
  automationId: string;
}

// A push always lands pending, never activating itself — this surfaces
// that there's a diff waiting for a decision without leaving the Configuration tab.
export const AutomationPendingVersionBanner = ({ automationId }: AutomationPendingVersionBannerProps) => {
  const { data } = useFetchAutomationVersionDiff(automationId);
  const { tabPath } = useTabPath();
  if (!data?.pending) return null;

  return (
    <Link
      to={tabPath("versions")}
      className="flex items-center gap-2 rounded-md border border-border px-4 py-3 text-sm transition-colors duration-150 ease-standard hover:bg-accent/40"
    >
      <Clock className="size-4 shrink-0 text-warning" aria-hidden />
      A pending version is waiting to be merged. See the diff on the Versions tab.
    </Link>
  );
};
