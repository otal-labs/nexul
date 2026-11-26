import { useState } from "react";
import { KeyRound, PlusIcon } from "lucide-react";
import { EnterList } from "@/components/EnterList";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { EmptyState } from "@/components/EmptyState";
import { AutomationListBar } from "@/components/automation/AutomationListBar";
import { AutomationSecretForm } from "@/components/automation/AutomationSecretForm";
import { AutomationSecretRow } from "@/components/automation/AutomationSecretRow";
import { ActionButton } from "@/components/action/ActionButton";
import { useRowGlide } from "@/hooks/useRowGlide";
import { useFetchAutomationSecrets } from "@/hooks/AutomationSecretHooks";

// Shared workspace pool (ADR 0047). One form at a time: a new secret, or a new value for one already saved.
export const AutomationSecretsSection = () => {
  const { data, isPending, error } = useFetchAutomationSecrets();
  const [editing, setEditing] = useState<{ name?: string } | null>(null);
  const { ref: glideRef, prepare: prepareGlide } = useRowGlide();
  const empty = !!data && data.length === 0 && !editing;
  const addSecret = (
    <ActionButton permission="automations:write" verb="add automation secrets" icon={PlusIcon} variant="outline" size="sm" onClick={() => setEditing({})}>
      Add secret
    </ActionButton>
  );

  return (
    <section aria-label="Secrets" className="space-y-1">
      {data && !empty && (
        <AutomationListBar
          summary={
            <>
              {data.length > 0 && (
                <span className="font-mono tabular-nums">
                  {data.length} {data.length === 1 ? "secret" : "secrets"} ·{" "}
                </span>
              )}
              Every automation reads them as <code className="font-mono text-foreground">ctx.secrets.NAME</code>.
            </>
          }
        >
          {!editing && addSecret}
        </AutomationListBar>
      )}
      {editing && (
        <div className="px-4 pt-1 pb-2">
          <AutomationSecretForm key={editing.name ?? "new"} {...(editing.name ? { name: editing.name } : {})} onDone={() => setEditing(null)} />
        </div>
      )}
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {empty && (
        <EmptyState
          icon={KeyRound}
          title="No secrets yet"
          message="Save an API key or a webhook URL once, and every automation reads it as ctx.secrets.NAME without it ever sitting in its code."
          action={addSecret}
        />
      )}
      {data && data.length > 0 && (
        <div ref={glideRef}>
          <EnterList className="divide-y divide-border border-y border-border">
            {data.map((secret) => (
              <AutomationSecretRow key={secret.name} secret={secret} onReplace={() => setEditing({ name: secret.name })} onLeave={prepareGlide} />
            ))}
          </EnterList>
        </div>
      )}
    </section>
  );
};
