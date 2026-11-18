import { Button } from "@/components/ui/button";
import { ImportFailures } from "@/components/import/ImportFailures";
import { ImportProgress } from "@/components/import/ImportProgress";
import { ImportScanSummary } from "@/components/import/ImportScanSummary";
import { ImportStateStatus } from "@/components/import/ImportStateStatus";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { SettingsHint } from "@/components/settings/SettingsHint";
import { useCancelImport, useConfirmImport, useImport } from "@/hooks/ImportHooks";
import { ImportState, importSourceLabel } from "@/models/Import";

interface ImportRunCardProps {
  id: string;
}

// A scan finds tickets or docs; the button names whichever it found.
const recordCount = (n: number, docs: boolean) => {
  const [one, many] = docs ? ["doc", "docs"] : ["ticket", "tickets"];
  return `${n.toLocaleString()} ${n === 1 ? one : many}`;
};

// The open import, live: a scan under way, a scan to confirm within 24 hours, or records coming in.
export const ImportRunCard = ({ id }: ImportRunCardProps) => {
  const imp = useImport(id);
  const confirm = useConfirmImport();
  const cancel = useCancelImport();
  if (!imp) return null;
  const ready = imp.state === ImportState.Ready;

  return (
    <SettingsCard
      id="import-run"
      title={`${importSourceLabel(imp.source)}${imp.from ? ` from ${imp.from}` : ""}`}
      aside={<ImportStateStatus state={imp.state} />}
      footer={
        <>
          {ready && <SettingsHint>Waits 24 hours to be confirmed, then cancels itself.</SettingsHint>}
          <div className="ml-auto flex items-center gap-2">
            <Button variant="ghost" loading={cancel.isPending} onClick={() => cancel.mutate(imp.id)}>
              Cancel import
            </Button>
            {ready && (
              <Button loading={confirm.isPending} disabled={imp.total === 0} onClick={() => confirm.mutate(imp.id)}>
                Import {recordCount(imp.total, imp.report.docs > 0 && imp.report.tickets === 0)}
              </Button>
            )}
          </div>
        </>
      }
    >
      {imp.state === ImportState.Scanning && (
        <SettingsHint>Counting what the source holds and who it names. This page updates by itself.</SettingsHint>
      )}
      {ready && <ImportScanSummary id={imp.id} />}
      {imp.state === ImportState.Importing && <ImportProgress id={imp.id} />}
      {imp.report.failures.length > 0 && <ImportFailures id={imp.id} />}
    </SettingsCard>
  );
};
