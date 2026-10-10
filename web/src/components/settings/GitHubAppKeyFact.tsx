import { Fact } from "@/components/Fact";
import { SettingsStatus } from "@/components/settings/SettingsStatus";

interface GitHubAppKeyFactProps {
  set: boolean;
}

// Whether Nexul reads GitHub as the App or, without a key, only as the account connected under Connectors.
export const GitHubAppKeyFact = ({ set }: GitHubAppKeyFactProps) => (
  <div className="sm:col-span-3">
    <Fact label="Private key">
      {set && <SettingsStatus tone="success" detail="Nexul reads every installation as the App">Set</SettingsStatus>}
      {!set && (
        <SettingsStatus tone="warning" detail="only the connected account's repositories are visible">
          Not set
        </SettingsStatus>
      )}
    </Fact>
  </div>
);
