import { useSetAutomationEnabled } from "@/hooks/AutomationHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { missingRequiredFields, type Automation } from "@/models/Automation";

const fieldList = new Intl.ListFormat("en", { type: "conjunction" });

// A required setting left empty is a signal, never a gate: switching on asks first, since every run fails until it is set.
export const useSwitchAutomation = () => {
  const setEnabled = useSetAutomationEnabled();
  const { open: confirm } = useConfirmationDialog();

  const switchTo = async (automation: Automation, enabled: boolean) => {
    const missing = enabled ? missingRequiredFields(automation) : [];
    if (missing.length > 0) {
      const ok = await confirm({
        title: "Switch it on unconfigured?",
        message: `${fieldList.format(missing.map((f) => f.label))} ${missing.length === 1 ? "is" : "are"} empty, so every run fails until you set ${missing.length === 1 ? "it" : "them"} in Settings.`,
        confirmLabel: "Switch on",
        destructive: false,
      });
      if (!ok) return;
    }
    setEnabled.mutate({ id: automation.id, enabled });
  };

  return { switchTo, isPending: setEnabled.isPending };
};
