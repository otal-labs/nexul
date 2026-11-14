import { SettingsRow } from "@/components/settings/SettingsRow";
import { Switch } from "@/components/ui/switch";
import { useActionState } from "@/hooks/useActionState";
import { useSetSsoOnly } from "@/ee/hooks/SsoHooks";

interface SsoOnlyRowProps {
  only: boolean;
}

// Saved on the flip; the server turns it down until an instance Owner holds a passkey, and the toast says so.
export const SsoOnlyRow = ({ only }: SsoOnlyRowProps) => {
  const setOnly = useSetSsoOnly();
  const writable = useActionState({ permission: "instance:write" }).state === "live";
  return (
    <SettingsRow
      label="Only single sign-on"
      htmlFor="sso_only"
      description="Hides every other way to sign in and signs out everyone who didn't use single sign-on. Instance Owners stay signed in, and their passkey still works, so a broken provider never locks everyone out."
    >
      <Switch id="sso_only" checked={only} disabled={!writable || setOnly.isPending} onCheckedChange={(next) => setOnly.mutate(next)} />
    </SettingsRow>
  );
};
