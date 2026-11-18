import { ImportPersonRow } from "@/components/import/ImportPersonRow";
import { ImportStatusRow } from "@/components/import/ImportStatusRow";
import { SettingsHint } from "@/components/settings/SettingsHint";
import { SettingsRow, SettingsRows, settingsListClass } from "@/components/settings/SettingsRow";
import { useImport } from "@/hooks/ImportHooks";

interface ImportScanSummaryProps {
  id: string;
}

// What the scan found and where it goes: counts, then each status's column, then each person's member.
export const ImportScanSummary = ({ id }: ImportScanSummaryProps) => {
  const imp = useImport(id);
  if (!imp) return null;
  const { report } = imp;
  const unmatched = report.people.filter((p) => !p.member).length;
  return (
    <div className="space-y-6">
      <SettingsRows>
        <SettingsRow label="Found" description="What an earlier import brought in is skipped.">
          <span className="text-sm tabular-nums">
            {report.docs > 0 && `${report.docs.toLocaleString()} docs`}
            {report.docs === 0 && `${report.tickets.toLocaleString()} tickets · ${report.comments.toLocaleString()} comments`}
          </span>
        </SettingsRow>
      </SettingsRows>
      {report.statuses.length > 0 && (
        <section aria-labelledby="import-statuses">
          <h3 id="import-statuses" className="text-sm font-medium text-foreground">
            Statuses
          </h3>
          <ul className={`${settingsListClass} mt-2`}>
            {report.statuses.map((s) => (
              <ImportStatusRow key={`${s.kind}:${s.name}`} status={s} />
            ))}
          </ul>
        </section>
      )}
      {report.people.length > 0 && (
        <section aria-labelledby="import-people">
          <h3 id="import-people" className="text-sm font-medium text-foreground">
            People
          </h3>
          {unmatched > 0 && (
            <SettingsHint className="mt-1">
              {unmatched} {unmatched === 1 ? "person matches" : "people match"} no member who can open the project; their tickets stay
              unassigned and name them in the body.
            </SettingsHint>
          )}
          <ul className={`${settingsListClass} mt-2`}>
            {report.people.map((p) => (
              <ImportPersonRow key={p.login ?? p.email ?? p.name} person={p} />
            ))}
          </ul>
        </section>
      )}
    </div>
  );
};
