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
    description="Irreversible workspace actions live here, separated from daily settings so nothing destructive is one stray click away."
  >
    <p className="text-sm text-muted-foreground">
      No irreversible workspace-wide action exists on this instance yet. Anything added here
      will use the same confirm-before-continue step revoking a personal access token does.
    </p>
  </SettingsCard>
);
