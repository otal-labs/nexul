import { ExternalLink } from "lucide-react";

import { useFetchConnectorAppConfig } from "@/hooks/ConnectorsHooks";
import { githubAppInstallURL } from "@/models/Connectors";

// Renders nothing until the instance's GitHub App has a slug, since there is no install page to send anyone to.
export const GitHubInstallShortcut = () => {
  const { data: app } = useFetchConnectorAppConfig("github");
  if (!app?.app_slug) return null;

  return (
    <div className="mt-2 space-y-0.5">
      <a
        href={githubAppInstallURL(app)}
        target="_blank"
        rel="noreferrer"
        className="inline-flex items-center gap-1 text-xs font-medium underline-offset-2 hover:underline"
      >
        Let Nexul deploy your repositories
        <ExternalLink className="size-3" aria-hidden />
      </a>
      <p className="text-xs text-muted-foreground">Installs Nexul's GitHub App on your account; pick which repositories.</p>
    </div>
  );
};
