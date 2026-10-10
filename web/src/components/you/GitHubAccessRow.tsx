import { Unlink } from "lucide-react";

import { GithubMark } from "@/components/ProviderMarks";
import { ConfirmDestroyButton } from "@/components/settings/ConfirmDestroyButton";
import { SettingsStatus } from "@/components/settings/SettingsStatus";
import { Button } from "@/components/ui/button";
import { useStartIdentityLink } from "@/hooks/AuthHooks";
import { useDisconnectGitHub } from "@/hooks/GitHubLinkHooks";
import type { GitHubLinkStatus } from "@/models/GitHubLink";

interface GitHubAccessRowProps {
  link: GitHubLinkStatus;
}

// Connect runs GitHub's sign-in in link mode, so it joins this profile and never makes a second account.
export const GitHubAccessRow = ({ link }: GitHubAccessRowProps) => {
  const connect = useStartIdentityLink();
  const disconnect = useDisconnectGitHub();
  const account = link.login ? `@${link.login}` : undefined;

  return (
    <li className="flex items-center gap-3 bg-card px-3 py-3">
      <span className="flex size-9 shrink-0 items-center justify-center rounded-md border border-border bg-muted/60">
        <GithubMark />
      </span>
      <div className="min-w-0 flex-1">
        <p className="text-sm font-medium">Your GitHub repositories</p>
        {link.state === "connected" && (
          <SettingsStatus tone="success" detail={account && <span className="font-mono">{account}</span>}>
            Connected
          </SettingsStatus>
        )}
        {link.state === "reconnect" && (
          <SettingsStatus tone="warning" detail="GitHub asked to be connected again">
            Reconnect needed
          </SettingsStatus>
        )}
        {link.state === "none" && <SettingsStatus tone="muted">Not connected</SettingsStatus>}
      </div>
      {link.state !== "connected" && (
        <Button variant="outline" size="sm" loading={connect.isPending} onClick={() => connect.mutate("github")}>
          {link.state === "reconnect" ? "Reconnect" : "Connect GitHub"}
        </Button>
      )}
      {link.state !== "none" && (
        <ConfirmDestroyButton
          icon={Unlink}
          idleLabel="Disconnect GitHub"
          confirmLabel="Disconnect"
          loading={disconnect.isPending}
          onConfirm={() => disconnect.mutate()}
        />
      )}
    </li>
  );
};
