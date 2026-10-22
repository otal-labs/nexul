import { useState } from "react";

import { EmptyRow } from "@/components/EmptyRow";
import { PairingDefaultsRow } from "@/components/settings/PairingDefaultsRow";
import { PairingProjectGroup } from "@/components/settings/PairingProjectGroup";
import type { Workspace } from "@/models/Workspace";

const DEFAULTS = "defaults";

interface PairingProjectsFeedProps {
  workspaces: Workspace[];
}

// Defaults, then every project of every workspace in one list, one row open at a time.
export const PairingProjectsFeed = ({ workspaces }: PairingProjectsFeedProps) => {
  const [openId, setOpenId] = useState<string>();
  const toggle = (id: string) => setOpenId((current) => (current === id ? undefined : id));

  return (
    <div className="space-y-3">
      <ul aria-label="Defaults and projects" className="divide-y divide-border overflow-hidden rounded-md border border-border">
        <PairingDefaultsRow open={openId === DEFAULTS} onToggle={() => toggle(DEFAULTS)} />
        {workspaces.map((workspace) => (
          <PairingProjectGroup key={workspace.id} workspace={workspace} openId={openId} onToggle={toggle} />
        ))}
      </ul>
      {workspaces.length === 0 && <EmptyRow>No projects you can open yet.</EmptyRow>}
    </div>
  );
};
