import { useState } from "react";
import { PlusIcon } from "lucide-react";
import { EnterList } from "@/components/EnterList";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { EmptyRow } from "@/components/EmptyRow";
import { AutomationSecretForm } from "@/components/automation/AutomationSecretForm";
import { AutomationSecretRow } from "@/components/automation/AutomationSecretRow";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { settingsRowListClass } from "@/components/settings/SettingsRow";
import { ActionButton } from "@/components/action/ActionButton";
import { useRowGlide } from "@/hooks/useRowGlide";
import { useFetchAutomationSecrets } from "@/hooks/AutomationSecretHooks";

// Shared workspace pool (ADR 0047). One form at a time: a new secret, or a new value for one already saved.
export const AutomationSecretsSection = () => {
  const { data, isPending, error } = useFetchAutomationSecrets();
  const [editing, setEditing] = useState<{ name?: string } | null>(null);
  const { ref: glideRef, prepare: prepareGlide } = useRowGlide();

  return (
    <SettingsCard
      id="automation-secrets"
      title="Automation secrets"
      description="Every automation reads these as ctx.secrets.NAME. A saved value can't be read back, only replaced or deleted."
      footer={
        !editing && (
          <ActionButton permission="automations:write" verb="add automation secrets" icon={PlusIcon} variant="outline" size="sm" onClick={() => setEditing({})}>
            Add secret
          </ActionButton>
        )
      }
    >
      <div className="space-y-4">
        {editing && (
          <AutomationSecretForm key={editing.name ?? "new"} {...(editing.name ? { name: editing.name } : {})} onDone={() => setEditing(null)} />
        )}
        {isPending && <LoadingDisplay />}
        {error && <ErrorDisplay error={error} />}
        {data && data.length === 0 && !editing && <EmptyRow>No secrets yet. Add one for automations to read.</EmptyRow>}
        {data && data.length > 0 && (
          <div ref={glideRef}>
            <EnterList className={settingsRowListClass}>
              {data.map((secret) => (
                <AutomationSecretRow key={secret.name} secret={secret} onReplace={() => setEditing({ name: secret.name })} onLeave={prepareGlide} />
              ))}
            </EnterList>
          </div>
        )}
      </div>
    </SettingsCard>
  );
};
