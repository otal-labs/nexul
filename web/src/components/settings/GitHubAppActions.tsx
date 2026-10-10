import { ConnectorAppEditDialog } from "@/components/settings/ConnectorAppEditDialog";
import { GitHubAppKeyDialog } from "@/components/settings/GitHubAppKeyDialog";
import { Button } from "@/components/ui/button";
import { useRemovePrivateKey } from "@/hooks/ConnectorsHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import type { AppConfigStatus } from "@/models/Connectors";

interface GitHubAppActionsProps {
  app: AppConfigStatus;
}

// The registered App's footer: its private key, added, replaced or removed, beside Edit.
export const GitHubAppActions = ({ app }: GitHubAppActionsProps) => {
  const removeKey = useRemovePrivateKey();
  const { open: confirm } = useConfirmationDialog();

  const remove = async () => {
    const ok = await confirm({
      title: "Remove the private key?",
      message:
        "Nexul goes back to reading GitHub as the connected account: every workspace lists what that account can open, and installations on accounts it cannot open stop listing.",
      confirmLabel: "Remove",
      destructive: true,
    });
    if (ok) removeKey.mutate("github");
  };

  return (
    <div className="ml-auto flex flex-wrap items-center gap-2">
      {app.private_key_set && (
        <Button
          variant="ghost"
          size="sm"
          className="text-muted-foreground hover:text-destructive"
          loading={removeKey.isPending}
          onClick={() => void remove()}
        >
          Remove key
        </Button>
      )}
      <GitHubAppKeyDialog replacing={!!app.private_key_set} />
      <ConnectorAppEditDialog connectorId="github" current={app} />
    </div>
  );
};
