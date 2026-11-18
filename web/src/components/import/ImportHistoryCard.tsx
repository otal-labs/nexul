import { ImportHistoryRow } from "@/components/import/ImportHistoryRow";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { settingsListClass } from "@/components/settings/SettingsRow";
import type { Import } from "@/models/Import";

interface ImportHistoryCardProps {
  imports: Import[];
}

// The imports that have ended, newest first; their credentials are already gone.
export const ImportHistoryCard = ({ imports }: ImportHistoryCardProps) => (
  <SettingsCard id="import-history" title="Earlier imports">
    <ul className={settingsListClass}>
      {imports.map((imp) => (
        <ImportHistoryRow key={imp.id} imp={imp} />
      ))}
    </ul>
  </SettingsCard>
);
