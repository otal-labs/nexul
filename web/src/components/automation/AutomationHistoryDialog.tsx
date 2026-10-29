import { AutomationHistoryFooter } from "@/components/automation/AutomationHistoryFooter";
import { HistoryDialog } from "@/components/history/HistoryDialog";
import { automationHistoryUrl, automationVersionUrl } from "@/hooks/AutomationVersionHooks";

interface AutomationHistoryDialogProps {
  automationId: string;
  name: string;
  onClose: () => void;
}

// An automation's code is compared line by line, and the footer merges the pending version or rolls back to an older one in place of Restore.
export const AutomationHistoryDialog = ({ automationId, name, onClose }: AutomationHistoryDialogProps) => (
  <HistoryDialog
    historyUrl={automationHistoryUrl(automationId)}
    versionUrl={(sequence) => automationVersionUrl(automationId, sequence)}
    name={name}
    bodyKind="code"
    retention="all"
    footer={(version) => <AutomationHistoryFooter automationId={automationId} version={version} onDone={onClose} />}
    onClose={onClose}
  />
);
