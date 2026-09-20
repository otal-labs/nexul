import { ConnectorAppConfigForm } from "@/components/settings/ConnectorAppConfigForm";
import { SettingsCard } from "@/components/settings/SettingsCard";

// GitHub only: bootstrap seeds this app, so the card exists for rotation. Other connectors register from their card's "Set up app".
export const ConnectorAppConfigSection = () => (
  <SettingsCard
    id="connector-app-config"
    title="GitHub App"
    description="The GitHub App this instance uses to let people connect their own GitHub account. Register
        your own App and paste its credentials here — Nexul never holds or manages the App itself."
  >
    <ConnectorAppConfigForm connectorId="github" />
  </SettingsCard>
);
