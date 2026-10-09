import { TriangleAlert } from "lucide-react";

import { SettingsCard } from "@/components/settings/SettingsCard";

// Understated destructive tint over a heavy warning box, via SettingsCard's danger/icon props.
// No workspace-wide destructive action exists yet; add one via PATRow's ConfirmDestroyButton pattern.
export const DangerZoneSection = () => (
  <SettingsCard
    id="danger-zone"
    title="Danger zone"
    danger
    icon={TriangleAlert}
    description="Workspace actions that can't be undone."
  >
    <p className="text-sm text-muted-foreground">None yet. Each one added here will ask you to confirm first.</p>
  </SettingsCard>
);
