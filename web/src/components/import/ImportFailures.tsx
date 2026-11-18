import { ImportFailureRow } from "@/components/import/ImportFailureRow";
import { SettingsHint } from "@/components/settings/SettingsHint";
import { settingsListClass } from "@/components/settings/SettingsRow";
import { useImport } from "@/hooks/ImportHooks";

interface ImportFailuresProps {
  id: string;
}

// The records that did not come in and why; past the first 200 only the count goes on.
export const ImportFailures = ({ id }: ImportFailuresProps) => {
  const imp = useImport(id);
  if (!imp) return null;
  return (
    <section aria-labelledby={`import-failures-${imp.id}`} className="mt-6">
      <h3 id={`import-failures-${imp.id}`} className="text-sm font-medium text-foreground">
        {imp.failed.toLocaleString()} did not come in
      </h3>
      {imp.failed > imp.report.failures.length && (
        <SettingsHint className="mt-1">The first {imp.report.failures.length} are listed.</SettingsHint>
      )}
      <ul className={`${settingsListClass} mt-2`}>
        {imp.report.failures.map((f) => (
          <ImportFailureRow key={`${f.kind}:${f.source_id}`} failure={f} />
        ))}
      </ul>
    </section>
  );
};
