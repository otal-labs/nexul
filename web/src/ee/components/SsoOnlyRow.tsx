import { SettingsRow } from "@/components/settings/SettingsRow";
import { Switch } from "@/components/ui/switch";
import { useActionState } from "@/hooks/useActionState";

interface SsoOnlyRowProps {
  only: boolean;
  // The provider's own save of the switch: OpenID Connect's or SAML's.
  setOnly: { mutate: (only: boolean) => void; isPending: boolean };
}

// Saved on the flip; the server turns it down until an instance Owner holds a passkey, and the toast says so.
export const SsoOnlyRow = ({ only, setOnly }: SsoOnlyRowProps) => {
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
