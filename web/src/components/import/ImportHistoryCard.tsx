import { EmptyRow } from "@/components/EmptyRow";
import { ImportHistoryRow } from "@/components/import/ImportHistoryRow";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { settingsListClass } from "@/components/settings/SettingsRow";
import type { Import } from "@/models/Import";

interface ImportHistoryCardProps {
  imports: Import[];
}

// The imports that have ended, newest first; their credentials are already gone.
export const ImportHistoryCard = ({ imports }: ImportHistoryCardProps) => (
  <SettingsCard id="import-history" title="Past imports" description="Running an import again skips what came in before.">
    {imports.length === 0 && (
      <EmptyRow flush className="border-t border-border pt-4 text-xs">
        None yet. An import is listed here once it ends.
      </EmptyRow>
    )}
    {imports.length > 0 && (
      <ul className={settingsListClass}>
        {imports.map((imp) => (
          <ImportHistoryRow key={imp.id} imp={imp} />
        ))}
      </ul>
    )}
  </SettingsCard>
);
