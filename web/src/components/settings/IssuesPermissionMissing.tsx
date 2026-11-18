import { SettingsStatus } from "@/components/settings/SettingsStatus";

interface IssuesPermissionMissingProps {
  // Where the change is accepted, as the place showing it says it.
  accept: string;
}

// An installation that has not accepted the App's Issues permission yet, and how to give it.
export const IssuesPermissionMissing = ({ accept }: IssuesPermissionMissingProps) => (
  <div className="mt-1">
    <SettingsStatus tone="warning">Issues permission missing</SettingsStatus>
    <p className="text-xs text-muted-foreground">Give the App Issues: Read and write in its settings on GitHub, then accept the change {accept}.</p>
  </div>
);
