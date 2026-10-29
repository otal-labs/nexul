import { Button } from "@/components/ui/button";
import { NoDataDisplay } from "@/components/NoDataDisplay";
import { ImportContainerRow } from "@/components/wizard/ImportContainerRow";
import { ImportGatewayRow } from "@/components/wizard/ImportGatewayRow";
import { ImportStackGroupSection } from "@/components/wizard/ImportStackGroupSection";
import { WizardCardList, wizardCardClass } from "@/components/wizard/WizardCardList";
import { WizardFooter } from "@/components/wizard/WizardFooter";
import type { useImportSelection } from "@/hooks/useImportSelection";
import type { GroupedDiscovery } from "@/models/Machine";

interface ImportAdoptStepProps {
  report: GroupedDiscovery;
  pick: ReturnType<typeof useImportSelection>;
  onBack: () => void;
  onImport: () => void;
  importing: boolean;
}

// What the machine runs, ticked by default: each compose project, each lone container, each gateway, as its own card.
export const ImportAdoptStep = ({ report, pick, onBack, onImport, importing }: ImportAdoptStepProps) => {
  const { selection, toggleContainer, toggleStandalone, toggleGateway } = pick;
  const nothingFound = report.stacks.length === 0 && report.standalone.length === 0 && report.gateways.length === 0;
  const picked =
    Object.values(selection.stacks).reduce((n, names) => n + names.size, 0) + selection.standalone.size + selection.gateways.size;

  return (
    <div>
      {nothingFound && <NoDataDisplay message="Nothing running on this machine to import." />}
      {!nothingFound && (
        <WizardCardList aria-label="What to adopt">
          {report.stacks.map((group) => (
            <li key={group.project} className={wizardCardClass}>
              <ImportStackGroupSection
                group={group}
                selected={selection.stacks[group.project] ?? new Set()}
                onToggleContainer={(name) => toggleContainer(group.project, name)}
              />
            </li>
          ))}
          {report.standalone.map((c) => (
            <li key={c.name} className={wizardCardClass}>
              <ImportContainerRow container={c} checked={selection.standalone.has(c.name)} onToggle={() => toggleStandalone(c.name)} />
            </li>
          ))}
          {report.gateways.map((c) => (
            <li key={c.name} className={wizardCardClass}>
              <ImportGatewayRow container={c} checked={selection.gateways.has(c.name)} onToggle={() => toggleGateway(c.name)} />
            </li>
          ))}
        </WizardCardList>
      )}
      <WizardFooter onBack={onBack}>
        {!nothingFound && (
          <Button onClick={onImport} loading={importing} disabled={picked === 0}>
            Import {picked}
          </Button>
        )}
      </WizardFooter>
    </div>
  );
};
