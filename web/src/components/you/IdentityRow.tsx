import { Unlink } from "lucide-react";
import { Link } from "react-router";

import { ConfirmDestroyButton } from "@/components/settings/ConfirmDestroyButton";
import { SettingsStatus } from "@/components/settings/SettingsStatus";
import { Button } from "@/components/ui/button";
import { ProviderMark } from "@/components/ProviderMarks";
import { useCanOpenSection } from "@/hooks/AccessHooks";
import { useStartIdentityLink, useUnlinkIdentity } from "@/hooks/AuthHooks";
import { identityAccount, providerLabel, type Identity, type Provider } from "@/models/User";

interface IdentityRowProps {
  provider: Provider;
  identity?: Identity | undefined;
  onlyOne: boolean;
}

export const IdentityRow = ({ provider, identity, onlyOne }: IdentityRowProps) => {
  const label = providerLabel[provider];
  const link = useStartIdentityLink();
  const unlink = useUnlinkIdentity();
  const canOpenConnectors = useCanOpenSection("connectors");
  const account = identity && identityAccount(identity);

  return (
    <li className="flex min-h-14 items-center gap-3 py-2.5">
      <span className="flex size-9 shrink-0 items-center justify-center rounded-md border border-border bg-muted/60 [&_svg]:size-4">
        <ProviderMark provider={provider} />
      </span>
      <div className="min-w-0 flex-1">
        <p className="text-sm font-medium">{label}</p>
        <p className="flex min-w-0">
          {account && (
            <SettingsStatus tone="success" detail={<span className="font-mono">{account}</span>}>
              Linked
            </SettingsStatus>
          )}
          {!account && <SettingsStatus tone="muted">Not linked</SettingsStatus>}
          {/* In the meta line, not the action column, so every row ends on the same one action. */}
          {identity && provider === "github" && canOpenConnectors && (
            <Link
              to="/settings/connectors/github-app"
              aria-label="GitHub App settings"
              className="ml-4 text-xs text-muted-foreground underline decoration-border underline-offset-2 transition-colors duration-150 ease-standard hover:text-foreground"
            >
              App settings
            </Link>
          )}
        </p>
      </div>
      {identity && !onlyOne && (
        <ConfirmDestroyButton
          icon={Unlink}
          idleLabel="Unlink"
          label={`Unlink ${label}`}
          confirmLabel="Unlink"
          labelled
          loading={unlink.isPending}
          onConfirm={() => unlink.mutate(provider)}
        />
      )}
      {identity && onlyOne && <span className="text-xs text-muted-foreground">Your only sign-in</span>}
      {!identity && (
        <Button variant="outline" size="sm" loading={link.isPending} onClick={() => link.mutate(provider)}>
          Link {label}
        </Button>
      )}
    </li>
  );
};
