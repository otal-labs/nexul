import { FixLink } from "@/components/FixLink";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import type { ReadinessLink } from "@/models/Pairing";

// A readiness link names a place on the Agent page, so it opens in the workspace in view.
export const ReadinessFixLink = ({ to, label }: ReadinessLink) => {
  const wsPath = useWorkspacePath();
  return <FixLink to={wsPath(to)} label={label} />;
};
