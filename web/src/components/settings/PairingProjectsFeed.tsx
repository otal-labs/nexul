import { useState } from "react";

import { EmptyRow } from "@/components/EmptyRow";
import { PairingProjectGroup } from "@/components/settings/PairingProjectGroup";
import type { Workspace } from "@/models/Workspace";

interface PairingProjectsFeedProps {
  workspaces: Workspace[];
}

// One project's form open at a time, across every workspace's group.
export const PairingProjectsFeed = ({ workspaces }: PairingProjectsFeedProps) => {
  const [openId, setOpenId] = useState<string>();
  const labelled = workspaces.length > 1;
  const toggle = (projectId: string) => setOpenId((current) => (current === projectId ? undefined : projectId));

  return (
    <div className="space-y-6">
      {workspaces.length === 0 && <EmptyRow>No projects you can open yet.</EmptyRow>}
      {workspaces.map((workspace) => (
        <PairingProjectGroup key={workspace.id} workspace={workspace} labelled={labelled} openId={openId} onToggle={toggle} />
      ))}
    </div>
  );
};
