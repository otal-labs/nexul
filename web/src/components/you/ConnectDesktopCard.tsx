import { CopyButton } from "@/components/settings/CopyButton";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { useCopyConnectionToken } from "@/hooks/AuthHooks";

export const ConnectDesktopCard = () => {
  const copyToken = useCopyConnectionToken();

  return (
    <SettingsCard id="connect-desktop" title="Connect the desktop app" description="Paste the token into the desktop app, then sign in there.">
      <div className="space-y-3 text-sm">
        <p className="text-muted-foreground">It holds this server's address, not your account.</p>
        <CopyButton label="Copy connection token" value={() => copyToken.mutateAsync()} loading={copyToken.isPending} />
      </div>
    </SettingsCard>
  );
};
