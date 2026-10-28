import { CopyButton } from "@/components/settings/CopyButton";
import { SettingsRow } from "@/components/settings/SettingsRow";
import { useCopyConnectionToken } from "@/hooks/AuthHooks";

export const ConnectDesktopRow = () => {
  const copyToken = useCopyConnectionToken();

  return (
    <SettingsRow label="Desktop app" description="Paste the token into the desktop app, then sign in there. It holds this server's address, not your account.">
      <CopyButton label="Copy connection token" value={() => copyToken.mutateAsync()} loading={copyToken.isPending} />
    </SettingsRow>
  );
};
