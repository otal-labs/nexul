import { useState } from "react";

import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { GitHubExistingAppForm } from "@/components/setup/GitHubExistingAppForm";
import { GitHubManifestSetup } from "@/components/setup/GitHubManifestSetup";

type AppPath = "create" | "existing";

interface GitHubAppFormProps {
  // Fixed by the domain step; this form runs on that origin so sign-in starts where the callback lands.
  instanceUrl: string;
}

// Two ways to the same App: let GitHub create it from a manifest, or paste one registered by hand.
export const GitHubAppForm = ({ instanceUrl }: GitHubAppFormProps) => {
  const [path, setPath] = useState<AppPath>("create");
  return (
    <div className="space-y-6">
      <ToggleGroup
        type="single"
        variant="segmented"
        size="sm"
        value={path}
        // Radix reports "" when the active item is clicked again; keep the current path.
        onValueChange={(value) => value && setPath(value as AppPath)}
        aria-label="How to connect the GitHub App"
        className="w-full [&>*]:flex-1"
      >
        <ToggleGroupItem value="create">Create on GitHub</ToggleGroupItem>
        <ToggleGroupItem value="existing">Use an existing App</ToggleGroupItem>
      </ToggleGroup>
      {path === "create" && <GitHubManifestSetup />}
      {path === "existing" && <GitHubExistingAppForm instanceUrl={instanceUrl} />}
    </div>
  );
};
