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
      No irreversible workspace-wide action exists on this instance yet. Revoking a personal
      access token (above) is the one truly irreversible action on this page today, and it
      already requires the same confirm-before-continue step anything added here would use.
    </p>
  </SettingsCard>
);
