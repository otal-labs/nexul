import type { ComponentType } from "react";
import { Settings, Unlink } from "lucide-react";
import { Link } from "react-router";

import { ConfirmDestroyButton } from "@/components/settings/ConfirmDestroyButton";
import { SettingsStatus } from "@/components/settings/SettingsStatus";
import { Button } from "@/components/ui/button";
import { DiscordMark, GithubMark, GoogleMark } from "@/components/ProviderMarks";
import { useCanOpenSection } from "@/hooks/AccessHooks";
import { useStartIdentityLink, useUnlinkIdentity } from "@/hooks/AuthHooks";
import { providerLabel, type Identity, type Provider } from "@/models/User";

const marks: Record<Provider, ComponentType> = { github: GithubMark, google: GoogleMark, discord: DiscordMark };

interface IdentityRowProps {
  provider: Provider;
  identity?: Identity | undefined;
  onlyOne: boolean;
}

export const IdentityRow = ({ provider, identity, onlyOne }: IdentityRowProps) => {
  const Mark = marks[provider];
  const label = providerLabel[provider];
  const link = useStartIdentityLink();
  const unlink = useUnlinkIdentity();
  const canOpenConnectors = useCanOpenSection("connectors");
  const account = identity && (provider === "github" ? `@${identity.login}` : identity.login);

  return (
    <li className="flex items-center gap-3 bg-card px-3 py-3 transition-colors duration-150 ease-standard hover:bg-accent/40">
      <span className="flex size-9 shrink-0 items-center justify-center rounded-md border border-border bg-muted/60 [&_svg]:size-4">
        <Mark />
      </span>
      <div className="min-w-0 flex-1">
        <p className="text-sm font-medium">{label}</p>
        {account && (
          <SettingsStatus tone="success" detail={<span className="font-mono">{account}</span>}>
            Linked
          </SettingsStatus>
        )}
        {!account && <SettingsStatus tone="muted">Not linked</SettingsStatus>}
      </div>
      {identity && provider === "github" && canOpenConnectors && (
        <Button asChild variant="ghost" size="sm" aria-label="GitHub App settings" title="GitHub App settings">
          <Link to="/settings/connectors/github-app">
            <Settings className="size-4" />
          </Link>
        </Button>
      )}
      {identity && !onlyOne && (
        <ConfirmDestroyButton
          icon={Unlink}
          idleLabel={`Unlink ${label}`}
          confirmLabel="Unlink"
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
