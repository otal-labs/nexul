import type { ComponentType } from "react";
import { Unlink } from "lucide-react";

import { ConfirmDestroyButton } from "@/components/settings/ConfirmDestroyButton";
import { Button } from "@/components/ui/button";
import { DiscordMark, GithubMark, GoogleMark } from "@/components/ProviderMarks";
import { GitHubInstallShortcut } from "@/components/you/GitHubInstallShortcut";
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
  const account = identity && (provider === "github" ? `@${identity.login}` : identity.login);

  return (
    <li className="flex items-center gap-3 bg-card px-3 py-3">
      <span className="flex size-9 shrink-0 items-center justify-center rounded-md border border-border bg-muted/60 [&_svg]:size-4">
        <Mark />
      </span>
      <div className="min-w-0 flex-1">
        <p className="text-sm font-medium">{label}</p>
        <p className="truncate font-mono text-xs text-muted-foreground">{account ?? "Not linked"}</p>
        {provider === "github" && identity && <GitHubInstallShortcut />}
      </div>
      {identity && !onlyOne && (
        <ConfirmDestroyButton
          icon={Unlink}
          idleLabel={`Unlink ${label}`}
          confirmLabel="Unlink"
          disabled={unlink.isPending}
          onConfirm={() => unlink.mutate(provider)}
        />
      )}
      {identity && onlyOne && <span className="text-xs text-muted-foreground">Your only sign-in</span>}
      {!identity && (
        <Button variant="outline" size="sm" disabled={link.isPending} onClick={() => link.mutate(provider)}>
          Link {label}
        </Button>
      )}
    </li>
  );
};
