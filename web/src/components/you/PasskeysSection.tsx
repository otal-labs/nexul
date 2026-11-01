import { KeyRound } from "lucide-react";

import { EmptyRow } from "@/components/EmptyRow";
import { EnterList } from "@/components/EnterList";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { SettingsHint } from "@/components/settings/SettingsHint";
import { settingsListClass } from "@/components/settings/SettingsRow";
import { Button } from "@/components/ui/button";
import { PasskeyRow } from "@/components/you/PasskeyRow";
import { useAddPasskey, useFetchPasskeys } from "@/hooks/PasskeyHooks";
import { passkeysSupported } from "@/lib/passkeys";

export const PasskeysSection = () => {
  const { data, isPending, error } = useFetchPasskeys();
  const add = useAddPasskey();
  const browserReady = passkeysSupported();
  const canAdd = !!data?.available && browserReady;

  return (
    <SettingsCard
      id="passkeys"
      title="Passkeys"
      description="Sign in with your fingerprint, face, or security key. Your linked accounts keep working."
      footer={
        canAdd && (
          <Button variant="outline" loading={add.isPending} onClick={() => add.mutate()}>
            <KeyRound className="size-4" />
            Add a passkey
          </Button>
        )
      }
    >
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {data && (
        <div className="space-y-3">
          {!data.available && <SettingsHint>Passkeys need Nexul on an https address.</SettingsHint>}
          {data.available && !browserReady && <SettingsHint>This browser can't make passkeys. Try a current Chrome, Edge, Firefox or Safari.</SettingsHint>}
          {data.passkeys.length === 0 && <EmptyRow flush>No passkeys yet.</EmptyRow>}
          {data.passkeys.length > 0 && (
            <EnterList className={settingsListClass}>
              {data.passkeys.map((passkey) => (
                <PasskeyRow key={passkey.id} passkey={passkey} />
              ))}
            </EnterList>
          )}
        </div>
      )}
    </SettingsCard>
  );
};
