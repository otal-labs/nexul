import { KeyRound } from "lucide-react";

import { EmptyState } from "@/components/EmptyState";
import { WorkspaceOwners } from "@/components/home/WorkspaceOwners";
import type { Workspace } from "@/models/Workspace";

export const NoRolesState = ({ workspace }: { workspace: Workspace }) => (
  <div className="flex min-h-full items-center justify-center py-8">
    <EmptyState
      icon={KeyRound}
      title={`No roles in ${workspace.name} yet`}
      message="Your roles decide what you see here, so there are no projects or plays for you yet. Chat works as usual. An Owner can give you a role."
      action={<WorkspaceOwners workspace={workspace} />}
    />
  </div>
);
